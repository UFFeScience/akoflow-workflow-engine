package connector_service_account

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	runtime_entity "github.com/UFFeScience/akoflow/internal/domain/resource/runtime"
	nfs "github.com/UFFeScience/akoflow/internal/infrastructure/kubernetes/entity/nfs"
)

func testConnector(server *httptest.Server) *ConnectorServiceAccount {
	host := strings.TrimPrefix(server.URL, "https://")
	r := runtime_entity.NewRuntime("k8s", 1, map[string]string{"K8S_API_SERVER_HOST": host, "K8S_API_SERVER_TOKEN": "token"}, "", "")
	return &ConnectorServiceAccount{client: server.Client(), runtime: r}
}

func TestServiceAccountCRUD(t *testing.T) {
	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusCreated)
		} else {
			w.WriteHeader(http.StatusOK)
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer s.Close()
	c := testConnector(s)
	v := nfs.ServiceAccount{Metadata: nfs.Metadata{Namespace: "ns", Name: "sa"}}
	if !c.CreateServiceAccount(v).Success || !c.ListServiceAccount("ns").Success || !c.UpdateServiceAccount(v).Success || !c.DeleteServiceAccount("ns", "sa").Success {
		t.Fatal("CRUD failed")
	}
	if New(c.runtime) == nil || newClient() == nil {
		t.Fatal("constructors")
	}
}

func TestServiceAccountFailures(t *testing.T) {
	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500); _, _ = w.Write([]byte("fail")) }))
	c := testConnector(s)
	v := nfs.ServiceAccount{Metadata: nfs.Metadata{Namespace: "ns", Name: "sa"}}
	if c.CreateServiceAccount(v).Success || c.ListServiceAccount("ns").Success || c.UpdateServiceAccount(v).Success || c.DeleteServiceAccount("ns", "sa").Success {
		t.Fatal("HTTP errors accepted")
	}
	s.Close()
	if c.ListServiceAccount("ns").Success {
		t.Fatal("network error accepted")
	}
	bad := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200); _, _ = w.Write([]byte("{")) }))
	defer bad.Close()
	if testConnector(bad).ListServiceAccount("ns").Success {
		t.Fatal("invalid JSON accepted")
	}
}
