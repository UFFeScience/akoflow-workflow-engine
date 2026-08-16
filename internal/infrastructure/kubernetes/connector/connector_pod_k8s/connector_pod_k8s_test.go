package connector_pod_k8s

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	runtime_entity "github.com/UFFeScience/akoflow/internal/domain/resource/runtime"
)

func podConnector(s *httptest.Server) *ConnectorPodK8s {
	r := runtime_entity.NewRuntime("k8s", 1, map[string]string{"K8S_API_SERVER_HOST": strings.TrimPrefix(s.URL, "https://"), "K8S_API_SERVER_TOKEN": "t"}, "", "")
	return &ConnectorPodK8s{client: s.Client(), runtime: r}
}

func TestPodOperations(t *testing.T) {
	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/log") {
			_, _ = w.Write([]byte("hello"))
			return
		}
		if r.Method == http.MethodDelete {
			w.WriteHeader(200)
			return
		}
		_, _ = w.Write([]byte(`{"items":[{"metadata":{"name":"pod-a"}}]}`))
	}))
	defer s.Close()
	c := podConnector(s)
	pods, err := c.GetPodByJob("ns", "job")
	if err != nil {
		t.Fatal(err)
	}
	if name, err := pods.GetPodName(); err != nil || name != "pod-a" {
		t.Fatal("pod name")
	}
	if _, err := (ResponseGetJobByPod{}).GetPodName(); err == nil {
		t.Fatal("empty items")
	}
	if logs, err := c.GetPodLogs("ns", "pod-a"); err != nil || logs != "hello" {
		t.Fatal("logs")
	}
	if err := c.DeletePod("ns", "pod-a"); err != nil {
		t.Fatal(err)
	}
	if New(c.runtime) == nil || newClient() == nil {
		t.Fatal("constructors")
	}
}

func TestPodFailures(t *testing.T) {
	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			w.WriteHeader(500)
			return
		}
		_, _ = w.Write([]byte("{"))
	}))
	c := podConnector(s)
	if _, err := c.GetPodByJob("ns", "job"); err == nil {
		t.Fatal("JSON")
	}
	if err := c.DeletePod("ns", "pod"); err == nil {
		t.Fatal("status")
	}
	s.Close()
	if _, err := c.GetPodLogs("ns", "pod"); err == nil {
		t.Fatal("network")
	}
}
