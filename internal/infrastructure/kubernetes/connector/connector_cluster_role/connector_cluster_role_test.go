package connector_cluster_role

import (
	runtime_entity "github.com/UFFeScience/akoflow/internal/domain/resource/runtime"
	nfs "github.com/UFFeScience/akoflow/internal/infrastructure/kubernetes/entity/nfs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func clusterRoleConnector(s *httptest.Server) *ConnectorClusterRole {
	r := runtime_entity.NewRuntime("k8s", 1, map[string]string{"K8S_API_SERVER_HOST": strings.TrimPrefix(s.URL, "https://"), "K8S_API_SERVER_TOKEN": "t"}, "", "")
	return &ConnectorClusterRole{client: s.Client(), runtime: r}
}
func TestCreateClusterRole(t *testing.T) {
	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(201)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer s.Close()
	c := clusterRoleConnector(s)
	if !c.CreateClusterRole(nfs.ClusterRole{}).Success {
		t.Fatal("create")
	}
	if New(c.runtime) == nil || newClient() == nil {
		t.Fatal("constructors")
	}
}
func TestCreateClusterRoleFailures(t *testing.T) {
	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500) }))
	c := clusterRoleConnector(s)
	if c.CreateClusterRole(nfs.ClusterRole{}).Success {
		t.Fatal("status")
	}
	s.Close()
	if c.CreateClusterRole(nfs.ClusterRole{}).Success {
		t.Fatal("network")
	}
	bad := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(201); _, _ = w.Write([]byte("{")) }))
	defer bad.Close()
	if clusterRoleConnector(bad).CreateClusterRole(nfs.ClusterRole{}).Success {
		t.Fatal("JSON")
	}
}
