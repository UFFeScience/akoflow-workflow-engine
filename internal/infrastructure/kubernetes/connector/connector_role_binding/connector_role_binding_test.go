package connector_role_binding

import (
	runtime_entity "github.com/UFFeScience/akoflow/internal/domain/resource/runtime"
	nfs "github.com/UFFeScience/akoflow/internal/infrastructure/kubernetes/entity/nfs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func bindingConnector(s *httptest.Server) *ConnectorRoleBinding {
	r := runtime_entity.NewRuntime("k8s", 1, map[string]string{"K8S_API_SERVER_HOST": strings.TrimPrefix(s.URL, "https://"), "K8S_API_SERVER_TOKEN": "t"}, "", "")
	return &ConnectorRoleBinding{client: s.Client(), runtime: r}
}
func TestCreateRoleBinding(t *testing.T) {
	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(201)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer s.Close()
	c := bindingConnector(s)
	if !c.CreateRoleBinding(nfs.RoleBinding{Metadata: nfs.Metadata{Namespace: "ns"}}).Success {
		t.Fatal("create")
	}
	if New(c.runtime) == nil || newClient() == nil {
		t.Fatal("constructors")
	}
}
func TestCreateRoleBindingFailures(t *testing.T) {
	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500) }))
	c := bindingConnector(s)
	v := nfs.RoleBinding{Metadata: nfs.Metadata{Namespace: "ns"}}
	if c.CreateRoleBinding(v).Success {
		t.Fatal("status")
	}
	s.Close()
	if c.CreateRoleBinding(v).Success {
		t.Fatal("network")
	}
	bad := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(201); _, _ = w.Write([]byte("{")) }))
	defer bad.Close()
	if bindingConnector(bad).CreateRoleBinding(v).Success {
		t.Fatal("JSON")
	}
}
