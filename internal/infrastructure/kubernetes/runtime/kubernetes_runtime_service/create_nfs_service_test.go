package kubernetes_runtime_service

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	runtime_entity "github.com/UFFeScience/akoflow/internal/domain/resource/runtime"
	workflow_activity_entity "github.com/UFFeScience/akoflow/internal/domain/workflow/activity"
	workflow_entity "github.com/UFFeScience/akoflow/internal/domain/workflow/definition"
	"github.com/UFFeScience/akoflow/internal/infrastructure/config"
	connector_k8s "github.com/UFFeScience/akoflow/internal/infrastructure/kubernetes/connector"
)

func TestCreateNFSManifestBuilders(t *testing.T) {
	wf := workflow_entity.Workflow{Id: 42, Name: "wf", Spec: workflow_entity.WorkflowSpec{MountPath: "/data", StoragePolicy: workflow_entity.WorkflowSpecStoragePolicy{StorageClassName: "fast", StorageSize: "10Gi"}}}
	a := workflow_activity_entity.WorkflowActivities{Id: 7, Name: "task"}
	c := CreateNfsService{}
	if c.SetWorkflow(wf).SetActivity(a).SetNamespace("lab") != &c {
		t.Fatal("fluent")
	}
	if c.GetWorkflow().Id != 42 || c.GetWorkflowActivity().Id != 7 || c.GetWorkflowIdString() != "42" || c.GetNamespace() != "lab" {
		t.Fatal("state")
	}
	if c.createServiceAccount().Metadata.Namespace != "lab" {
		t.Fatal("service account")
	}
	if c.createService().Metadata.Name == "" {
		t.Fatal("service")
	}
	if c.createPersistentVolumeClaim().Metadata.Name == "" {
		t.Fatal("pvc")
	}
	if c.createDeployment().Metadata.Name == "" {
		t.Fatal("deployment")
	}
	if c.createClusterRole().Metadata.Name == "" {
		t.Fatal("cluster role")
	}
	if c.createClusterRoleBinding().Metadata.Name == "" {
		t.Fatal("cluster binding")
	}
	if c.createRole().Metadata.Namespace != "lab" {
		t.Fatal("role")
	}
	if c.createRoleBinding().Metadata.Namespace != "lab" {
		t.Fatal("role binding")
	}
	if !strings.Contains(c.createStorageClass().Provisioner, "42") {
		t.Fatal("storage class")
	}
	if makeNfsProvisionerName(42) != "nfs-provisioner-42" {
		t.Fatal("provisioner")
	}
}

func TestCreateNFSNamespaceFallback(t *testing.T) {
	c := CreateNfsService{namespace: "old"}
	c.SetNamespace("  ")
	if c.GetNamespace() == "old" {
		t.Fatal("fallback not applied")
	}
}

func TestCreateNFSResources(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.WriteHeader(200)
		} else {
			w.WriteHeader(201)
		}
		_, _ = w.Write([]byte(`{"name":"created"}`))
	}))
	defer server.Close()
	rt := runtime_entity.NewRuntime("k8s", 1, map[string]string{"K8S_API_SERVER_HOST": strings.TrimPrefix(server.URL, "https://"), "K8S_API_SERVER_TOKEN": "t"}, "", "")
	rr := &pvcRuntimeRepo{runtime: rt}
	wf := workflow_entity.Workflow{Id: 42, Spec: workflow_entity.WorkflowSpec{StoragePolicy: workflow_entity.WorkflowSpecStoragePolicy{StorageSize: "1Gi", StorageClassName: "fast"}}}
	a := workflow_activity_entity.WorkflowActivities{Id: 3, Runtime: "k8s"}
	c := CreateNfsService{workflow: wf, workflowActivity: a, namespace: "lab", connector: connector_k8s.New(), runtimeRepository: rr}
	if !c.Create() {
		t.Fatal("create")
	}
	old := config.App()
	defer config.SetAppContainer(old)
	old.Connector.K8sConnector = connector_k8s.New()
	config.SetAppContainer(old)
	if !c.NfsServerIsCreated() {
		t.Fatal("exists")
	}
	rr.err = errors.New("db")
	if c.Create() || c.NfsServerIsCreated() {
		t.Fatal("runtime error")
	}
}
func TestCreateNFSProviderFailures(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500); _, _ = w.Write([]byte("fail")) }))
	defer server.Close()
	rt := runtime_entity.NewRuntime("k8s", 1, map[string]string{"K8S_API_SERVER_HOST": strings.TrimPrefix(server.URL, "https://")}, "", "")
	rr := &pvcRuntimeRepo{runtime: rt}
	c := CreateNfsService{workflow: workflow_entity.Workflow{Id: 1}, workflowActivity: workflow_activity_entity.WorkflowActivities{Runtime: "k8s"}, namespace: "ns", connector: connector_k8s.New(), runtimeRepository: rr}
	if !c.Create() {
		t.Fatal("provider errors retain best-effort contract")
	}
}
