package connector_job_k8s

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	runtime_entity "github.com/UFFeScience/akoflow/internal/domain/resource/runtime"
	k8s_job_entity "github.com/UFFeScience/akoflow/internal/infrastructure/kubernetes/entity/job"
)

func jobConnector(s *httptest.Server) *ConnectorJobK8s {
	r := runtime_entity.NewRuntime("k8s", 1, map[string]string{"K8S_API_SERVER_HOST": strings.TrimPrefix(s.URL, "https://"), "K8S_API_SERVER_TOKEN": "t"}, "", "")
	return &ConnectorJobK8s{client: s.Client(), runtime: r}
}

func TestJobOperations(t *testing.T) {
	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.WriteHeader(201)
		} else {
			w.WriteHeader(200)
		}
		_, _ = w.Write([]byte(`{"kind":"Job"}`))
	}))
	defer s.Close()
	c := jobConnector(s)
	if c.ApplyJob("ns", k8s_job_entity.K8sJob{}) == nil {
		t.Fatal("apply")
	}
	if _, err := c.GetJob("ns", "job"); err != nil {
		t.Fatal(err)
	}
	if err := c.DeleteJob("job", "ns"); err != nil {
		t.Fatal(err)
	}
	if New(c.runtime) == nil || newClient() == nil {
		t.Fatal("constructors")
	}
}

func TestJobFailures(t *testing.T) {
	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.WriteHeader(404)
		} else {
			w.WriteHeader(500)
		}
	}))
	c := jobConnector(s)
	if c.ApplyJob("ns", k8s_job_entity.K8sJob{}) != nil {
		t.Fatal("status")
	}
	if _, err := c.GetJob("ns", "job"); !errors.Is(err, ErrJobNotFound) {
		t.Fatal("not found")
	}
	if err := c.DeleteJob("job", "ns"); err == nil {
		t.Fatal("status")
	}
	s.Close()
	if c.ApplyJob("ns", k8s_job_entity.K8sJob{}) != nil {
		t.Fatal("network")
	}
	if _, err := c.GetJob("ns", "job"); err == nil {
		t.Fatal("network")
	}
	bad := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200); _, _ = w.Write([]byte("{")) }))
	defer bad.Close()
	if _, err := jobConnector(bad).GetJob("ns", "job"); err == nil {
		t.Fatal("JSON")
	}
}
