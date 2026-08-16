package kubernetes_runtime_service

import (
	"errors"
	"github.com/UFFeScience/akoflow/internal/domain"
	runtime_entity "github.com/UFFeScience/akoflow/internal/domain/resource/runtime"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/resource_repository"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/runtime_repository"
	connector_k8s "github.com/UFFeScience/akoflow/internal/infrastructure/kubernetes/connector"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type healthResources struct {
	resource_repository.IRepository
	resource                        *domain.Resource
	findErr, upsertErr, snapshotErr error
	upserts, snapshots              int
}

func (r *healthResources) FindByProviderID(string, string) (*domain.Resource, error) {
	return r.resource, r.findErr
}
func (r *healthResources) Upsert(domain.Resource) error { r.upserts++; return r.upsertErr }
func (r *healthResources) CreateSnapshot(domain.ResourceSnapshot) error {
	r.snapshots++
	return r.snapshotErr
}

type healthRuntimeRepo struct {
	runtime_repository.IRuntimeRepository
	runtime *runtime_entity.Runtime
	err     error
	status  int
}

func (r *healthRuntimeRepo) GetByName(string) (*runtime_entity.Runtime, error) {
	return r.runtime, r.err
}
func (r *healthRuntimeRepo) UpdateStatus(_ *runtime_entity.Runtime, s int) error {
	r.status = s
	return nil
}

func healthServer() *httptest.Server {
	return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/healthz":
			_, _ = w.Write([]byte("ok"))
		case strings.Contains(r.URL.Path, "metrics.k8s.io"):
			_, _ = w.Write([]byte(`{"items":[{"metadata":{"name":"node-a"},"usage":{"cpu":"10m","memory":"1024Ki"}}]}`))
		case r.URL.Path == "/api/v1/nodes":
			_, _ = w.Write([]byte(`{"items":[{"metadata":{"name":"node-a","creationTimestamp":"now"},"status":{"allocatable":{"cpu":"4","memory":"2048Ki","ephemeral-storage":"10"},"capacity":{"ephemeral-storage":"20"},"nodeInfo":{"osImage":"Linux"}}}]}`))
		default:
			w.WriteHeader(404)
		}
	}))
}
func TestK8sHealthAndDiscovery(t *testing.T) {
	server := healthServer()
	defer server.Close()
	host := strings.TrimPrefix(server.URL, "https://")
	rt := runtime_entity.NewRuntime("k8s", 1, map[string]string{"K8S_API_SERVER_HOST": host, "K8S_API_SERVER_TOKEN": "t", "K8S_ENVIRONMENT_VERSION_ID": "env"}, "", "")
	rr := &healthRuntimeRepo{runtime: rt}
	res := &healthResources{resource: &domain.Resource{ID: "r1"}}
	s := &HealthCheckRuntimeK8sService{k8sConnector: connector_k8s.New(), runtimeRepository: rr, resources: res}
	if !s.HealthCheck("k8s") || rr.status != runtime_repository.STATUS_READY || res.snapshots != 1 {
		t.Fatal("health")
	}
	if !s.DiscoverResources("k8s") || res.upserts != 1 {
		t.Fatal("discover")
	}
}
func TestK8sHealthBoundaries(t *testing.T) {
	rt := runtime_entity.NewRuntime("k8s", 1, map[string]string{}, "", "")
	rr := &healthRuntimeRepo{runtime: rt}
	res := &healthResources{}
	s := &HealthCheckRuntimeK8sService{k8sConnector: connector_k8s.New(), runtimeRepository: rr, resources: res}
	rr.err = errors.New("db")
	if s.HealthCheck("k8s") || s.DiscoverResources("k8s") {
		t.Fatal("runtime error")
	}
	rr.err = nil
	if s.HealthCheck("k8s") || s.DiscoverResources("k8s") {
		t.Fatal("missing metadata")
	}
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500) }))
	defer server.Close()
	rt.Metadata = map[string]string{"K8S_API_SERVER_HOST": strings.TrimPrefix(server.URL, "https://"), "K8S_ENVIRONMENT_VERSION_ID": "env"}
	if s.HealthCheck("k8s") || s.DiscoverResources("k8s") {
		t.Fatal("API failure")
	}
}
