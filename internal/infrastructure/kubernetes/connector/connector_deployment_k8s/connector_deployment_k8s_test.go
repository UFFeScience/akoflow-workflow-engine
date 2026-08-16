package connector_deployment_k8s

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	runtime_entity "github.com/UFFeScience/akoflow/internal/domain/resource/runtime"
	nfs_server_entity "github.com/UFFeScience/akoflow/internal/infrastructure/kubernetes/entity/nfs"
)

func deploymentConnector(server *httptest.Server) *ConnectorDeploymentK8s {
	host := strings.TrimPrefix(server.URL, "https://")
	runtime := runtime_entity.NewRuntime("k8s", 1, map[string]string{"K8S_API_SERVER_HOST": host, "K8S_API_SERVER_TOKEN": "token"}, "", "")
	return &ConnectorDeploymentK8s{client: server.Client(), runtime: runtime}
}
func TestDeploymentConnectorCRUDSuccess(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer token" {
			t.Error("authorization")
		}
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusCreated)
		} else {
			w.WriteHeader(http.StatusOK)
		}
		w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()
	c := deploymentConnector(server)
	deployment := nfs_server_entity.Deployment{Metadata: nfs_server_entity.Metadata{Namespace: "lab", Name: "nfs"}}
	if !c.ListDeployments("lab").Success || !c.GetDeployment("lab", "nfs").Success || !c.CreateDeployment(deployment).Success || !c.UpdateDeployment(deployment).Success || !c.DeleteDeployment("lab", "nfs").Success {
		t.Fatal("CRUD failed")
	}
	if New(c.runtime) == nil || newClient() == nil {
		t.Fatal("constructors")
	}
}
func TestDeploymentConnectorHTTPAndDecodeErrors(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("failure"))
	}))
	c := deploymentConnector(server)
	deployment := nfs_server_entity.Deployment{Metadata: nfs_server_entity.Metadata{Namespace: "lab", Name: "nfs"}}
	if c.ListDeployments("lab").Success || c.GetDeployment("lab", "nfs").Success || c.CreateDeployment(deployment).Success || c.UpdateDeployment(deployment).Success || c.DeleteDeployment("lab", "nfs").Success {
		t.Fatal("HTTP errors accepted")
	}
	server.Close()
	if c.ListDeployments("lab").Success {
		t.Fatal("network error accepted")
	}
	bad := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK); w.Write([]byte("{")) }))
	defer bad.Close()
	if deploymentConnector(bad).ListDeployments("lab").Success {
		t.Fatal("invalid JSON accepted")
	}
}
func TestDeploymentConnectorRequestErrors(t *testing.T) {
	runtime := runtime_entity.NewRuntime("k8s", 1, map[string]string{"K8S_API_SERVER_HOST": "bad host"}, "", "")
	c := &ConnectorDeploymentK8s{client: http.DefaultClient, runtime: runtime}
	if c.ListDeployments("lab").Success || c.GetDeployment("lab", "x").Success || c.DeleteDeployment("lab", "x").Success {
		t.Fatal("invalid host accepted")
	}
}
