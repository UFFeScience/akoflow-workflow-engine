package connector_healthz

import (
	runtime_entity "github.com/UFFeScience/akoflow/internal/domain/resource/runtime"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthzResponses(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer token" {
			t.Error("token")
		}
		w.WriteHeader(200)
	}))
	host := strings.TrimPrefix(server.URL, "https://")
	rt := runtime_entity.NewRuntime("k8s", 1, map[string]string{"K8S_API_SERVER_HOST": host, "K8S_API_SERVER_TOKEN": "token"}, "", "")
	if !New(rt).Healthz().Success {
		t.Fatal("healthy")
	}
	server.Close()
	if New(rt).Healthz().Success {
		t.Fatal("network failure")
	}
	if newClient() == nil {
		t.Fatal("client")
	}
}
func TestHealthzStatusFailure(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(503) }))
	defer server.Close()
	rt := runtime_entity.NewRuntime("k8s", 1, map[string]string{"K8S_API_SERVER_HOST": strings.TrimPrefix(server.URL, "https://")}, "", "")
	if New(rt).Healthz().Success {
		t.Fatal("status")
	}
}
