package connector_role

import (
	runtime_entity "github.com/UFFeScience/akoflow/internal/domain/resource/runtime"
	nfs "github.com/UFFeScience/akoflow/internal/infrastructure/kubernetes/entity/nfs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func roleConnector(s *httptest.Server) *ConnectorRole {
	r := runtime_entity.NewRuntime("k8s", 1, map[string]string{"K8S_API_SERVER_HOST": strings.TrimPrefix(s.URL, "https://"), "K8S_API_SERVER_TOKEN": "t"}, "", "")
	return &ConnectorRole{client: s.Client(), runtime: r}
}
func TestCreateRole(t *testing.T) {
	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(201)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer s.Close()
	c := roleConnector(s)
	if !c.CreateRole(nfs.Role{Metadata: nfs.Metadata{Namespace: "ns"}}).Success {
		t.Fatal("create")
	}
	if New(c.runtime) == nil || newClient() == nil {
		t.Fatal("constructors")
	}
}
func TestCreateRoleFailures(t *testing.T) {
	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500) }))
	c := roleConnector(s)
	v := nfs.Role{Metadata: nfs.Metadata{Namespace: "ns"}}
	if c.CreateRole(v).Success {
		t.Fatal("status")
	}
	s.Close()
	if c.CreateRole(v).Success {
		t.Fatal("network")
	}
	bad := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(201); _, _ = w.Write([]byte("{")) }))
	defer bad.Close()
	if roleConnector(bad).CreateRole(v).Success {
		t.Fatal("JSON")
	}
}
