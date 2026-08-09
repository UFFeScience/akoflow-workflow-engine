package connector_k8s

import (
	"crypto/tls"
	"net/http"
	"testing"

	runtime_entity "github.com/UFFeScience/akoflow/internal/domain/resource/runtime"
)

func TestConnectorBuildsAllKubernetesClients(t *testing.T) {
	client := NewClient()
	transport, ok := client.Transport.(*http.Transport)
	if !ok || transport.TLSClientConfig == nil || !transport.TLSClientConfig.InsecureSkipVerify {
		t.Fatal("expected configured TLS client")
	}
	_ = tls.VersionTLS13
	runtime := &runtime_entity.Runtime{Name: "k8s"}
	connector := New()
	clients := []any{
		connector.Namespace(runtime), connector.Pod(runtime), connector.Job(runtime), connector.Deployment(runtime),
		connector.Metrics(runtime), connector.PersistentVolumeClain(runtime), connector.ClusterRole(runtime),
		connector.ClusterRoleBinding(runtime), connector.Role(runtime), connector.RoleBinding(runtime),
		connector.Service(runtime), connector.ServiceAccount(runtime), connector.StorageClass(runtime),
		connector.Healthz(runtime), connector.Nodes(runtime),
	}
	for index, value := range clients {
		if value == nil {
			t.Fatalf("client %d is nil", index)
		}
	}
}
