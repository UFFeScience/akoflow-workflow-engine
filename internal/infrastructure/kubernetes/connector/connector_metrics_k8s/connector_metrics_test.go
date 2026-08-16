package connector_metrics_k8s

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	runtime_entity "github.com/UFFeScience/akoflow/internal/domain/resource/runtime"
)

func metricsConnector(s *httptest.Server) *ConnectorMetricsK8s {
	r := runtime_entity.NewRuntime("k8s", 1, map[string]string{"K8S_API_SERVER_HOST": strings.TrimPrefix(s.URL, "https://"), "K8S_API_SERVER_TOKEN": "t"}, "", "")
	return &ConnectorMetricsK8s{client: s.Client(), runtime: r}
}

func TestMetricsSuccess(t *testing.T) {
	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/pods/") {
			_, _ = w.Write([]byte(`{"window":"30s","containers":[{"usage":{"cpu":"2m","memory":"4Mi"}}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"items":[]}`))
	}))
	defer s.Close()
	c := metricsConnector(s)
	pod, err := c.GetPodMetrics("ns", "pod")
	if err != nil {
		t.Fatal(err)
	}
	m, err := pod.GetMetrics()
	if err != nil || m.Cpu != "2m" {
		t.Fatal("metric conversion")
	}
	if _, err = c.GetNodeMetrics(); err != nil {
		t.Fatal(err)
	}
	if _, err = (ResponseGetPodMetrics{}).GetMetrics(); err == nil {
		t.Fatal("missing metric accepted")
	}
	if New(c.runtime) == nil || newClient() == nil {
		t.Fatal("constructors")
	}
}

func TestMetricsFailures(t *testing.T) {
	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(404) }))
	c := metricsConnector(s)
	if _, err := c.GetPodMetrics("ns", "pod"); err == nil {
		t.Fatal("status accepted")
	}
	if _, err := c.GetNodeMetrics(); err == nil {
		t.Fatal("status accepted")
	}
	s.Close()
	if _, err := c.GetPodMetrics("ns", "pod"); err == nil {
		t.Fatal("network accepted")
	}
	bad := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("{")) }))
	defer bad.Close()
	if _, err := metricsConnector(bad).GetNodeMetrics(); err == nil {
		t.Fatal("JSON accepted")
	}
}
