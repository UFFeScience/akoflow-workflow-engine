package connector_namespace_k8s

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	runtime_entity "github.com/UFFeScience/akoflow/internal/domain/resource/runtime"
)

func namespaceConnector(s *httptest.Server) *ConnectorNamespaceK8s {
	r := runtime_entity.NewRuntime("k8s", 1, map[string]string{"K8S_API_SERVER_HOST": strings.TrimPrefix(s.URL, "https://"), "K8S_API_SERVER_TOKEN": "t"}, "", "")
	return &ConnectorNamespaceK8s{client: s.Client(), runtime: r}
}

func TestNamespaceOperations(t *testing.T) {
	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"kind":"Namespace","metadata":{"name":"lab"}}`))
	}))
	defer s.Close()
	c := namespaceConnector(s)
	if _, err := c.GetNamespace("lab"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.CreateNamespace("lab"); err != nil {
		t.Fatal(err)
	}
	if New(c.runtime) == nil || newClient() == nil {
		t.Fatal("constructors")
	}
}

func TestNamespaceFailures(t *testing.T) {
	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("{")) }))
	c := namespaceConnector(s)
	if _, err := c.GetNamespace("lab"); err == nil {
		t.Fatal("JSON")
	}
	if _, err := c.CreateNamespace("lab"); err == nil {
		t.Fatal("JSON")
	}
	s.Close()
	if _, err := c.GetNamespace("lab"); err == nil {
		t.Fatal("network")
	}
}
