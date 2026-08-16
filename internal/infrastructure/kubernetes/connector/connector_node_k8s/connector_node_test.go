package connector_node_k8s

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	runtime_entity "github.com/UFFeScience/akoflow/internal/domain/resource/runtime"
)

func nodeConnector(s *httptest.Server) ConnectorNodeK8s {
	r := runtime_entity.NewRuntime("k8s", 1, map[string]string{"K8S_API_SERVER_HOST": strings.TrimPrefix(s.URL, "https://"), "K8S_API_SERVER_TOKEN": "t"}, "", "")
	return ConnectorNodeK8s{client: s.Client(), runtime: r}
}

func TestNodeValuesAndList(t *testing.T) {
	n := Node{CpuMax: "4", MemoryMax: "2048Ki", NetworkMax: "100"}
	if n.GetCpuMax() != 4 || n.GetNodeMemoryMax() != 2 || n.GetNodeNetworkMax() != 100 {
		t.Fatal("node conversion")
	}
	if (Node{CpuMax: "x"}).GetCpuMax() != 0 || (Node{MemoryMax: "x"}).GetNodeMemoryMax() != 0 || (Node{NetworkMax: "x"}).GetNodeNetworkMax() != 0 {
		t.Fatal("invalid conversion")
	}
	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"items":[{"metadata":{"name":"n1","creationTimestamp":"now"},"status":{"allocatable":{"cpu":"4","memory":"2048Ki","ephemeral-storage":"10"},"capacity":{"ephemeral-storage":"20"},"nodeInfo":{"osImage":"Linux"}}}]}`))
	}))
	defer s.Close()
	if !nodeConnector(s).ListNodes().Success {
		t.Fatal("list")
	}
	if New(nodeConnector(s).runtime) == nil || newClient() == nil {
		t.Fatal("constructors")
	}
}

func TestNodeListFailures(t *testing.T) {
	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500) }))
	c := nodeConnector(s)
	if c.ListNodes().Success {
		t.Fatal("status accepted")
	}
	s.Close()
	if c.ListNodes().Success {
		t.Fatal("network accepted")
	}
	bad := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("{")) }))
	defer bad.Close()
	if nodeConnector(bad).ListNodes().Success {
		t.Fatal("JSON accepted")
	}
}
