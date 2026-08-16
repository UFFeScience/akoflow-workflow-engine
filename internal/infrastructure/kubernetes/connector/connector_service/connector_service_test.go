package connector_service

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	runtime_entity "github.com/UFFeScience/akoflow/internal/domain/resource/runtime"
	nfs "github.com/UFFeScience/akoflow/internal/infrastructure/kubernetes/entity/nfs"
)

func testConnector(server *httptest.Server) *ConnectorService {
	host := strings.TrimPrefix(server.URL, "https://")
	r := runtime_entity.NewRuntime("k8s", 1, map[string]string{"K8S_API_SERVER_HOST": host, "K8S_API_SERVER_TOKEN": "token"}, "", "")
	return &ConnectorService{client: server.Client(), runtime: r}
}

func TestServiceCRUD(t *testing.T) {
	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer token" {
			t.Error("missing token")
		}
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusCreated)
		} else {
			w.WriteHeader(http.StatusOK)
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer s.Close()
	c := testConnector(s)
	v := nfs.Service{Metadata: nfs.Metadata{Namespace: "ns", Name: "svc"}}
	if !c.CreateService(v).Success || !c.ListService("ns").Success || !c.UpdateService(v).Success || !c.DeleteService("ns", "svc").Success {
		t.Fatal("CRUD failed")
	}
	if New(c.runtime) == nil || newClient() == nil {
		t.Fatal("constructors")
	}
}

func TestServiceFailures(t *testing.T) {
	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500); _, _ = w.Write([]byte("fail")) }))
	c := testConnector(s)
	v := nfs.Service{Metadata: nfs.Metadata{Namespace: "ns", Name: "svc"}}
	if c.CreateService(v).Success || c.ListService("ns").Success || c.UpdateService(v).Success || c.DeleteService("ns", "svc").Success {
		t.Fatal("HTTP errors accepted")
	}
	s.Close()
	if c.ListService("ns").Success {
		t.Fatal("network error accepted")
	}
	bad := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200); _, _ = w.Write([]byte("{")) }))
	defer bad.Close()
	if testConnector(bad).ListService("ns").Success {
		t.Fatal("invalid JSON accepted")
	}
	r := runtime_entity.NewRuntime("k8s", 1, map[string]string{"K8S_API_SERVER_HOST": "bad host"}, "", "")
	invalid := &ConnectorService{client: http.DefaultClient, runtime: r}
	if invalid.ListService("ns").Success || invalid.DeleteService("ns", "svc").Success {
		t.Fatal("invalid URL accepted")
	}
}
