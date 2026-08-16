package kubernetes_runtime_service

import (
	"errors"
	"github.com/UFFeScience/akoflow/internal/application/ports"
	runtime_entity "github.com/UFFeScience/akoflow/internal/domain/resource/runtime"
	workflow_activity_entity "github.com/UFFeScience/akoflow/internal/domain/workflow/activity"
	workflow_entity "github.com/UFFeScience/akoflow/internal/domain/workflow/definition"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/runtime_repository"
	connector_k8s "github.com/UFFeScience/akoflow/internal/infrastructure/kubernetes/connector"
	connector_pvc_k8s "github.com/UFFeScience/akoflow/internal/infrastructure/kubernetes/connector/connector_pvc_k8s"
	"testing"
)

type pvcRuntimeRepo struct {
	runtime_repository.IRuntimeRepository
	runtime *runtime_entity.Runtime
	err     error
}

func (r *pvcRuntimeRepo) GetByName(string) (*runtime_entity.Runtime, error) { return r.runtime, r.err }

type pvcStorageRepo struct {
	ports.StorageRepository
	err   error
	calls int
}

func (r *pvcStorageRepo) Update(ports.UpdateStorageParams) error { r.calls++; return r.err }

type pvcFake struct {
	connector_pvc_k8s.IConnectorPvc
	get               connector_pvc_k8s.ResponseGetPersistentVolumeClain
	create            connector_pvc_k8s.ResponseCreatePersistentVolumeClain
	getErr, createErr error
}

func (p *pvcFake) GetPersistentVolumeClain(string, string) (connector_pvc_k8s.ResponseGetPersistentVolumeClain, error) {
	return p.get, p.getErr
}
func (p *pvcFake) CreatePersistentVolumeClain(string, string, string, string) (connector_pvc_k8s.ResponseCreatePersistentVolumeClain, error) {
	return p.create, p.createErr
}

type pvcTopConnector struct {
	connector_k8s.IConnector
	pvc *pvcFake
}

func (c *pvcTopConnector) PersistentVolumeClain(*runtime_entity.Runtime) connector_pvc_k8s.IConnectorPvc {
	return c.pvc
}
func TestCreatePVCExistingAndCreate(t *testing.T) {
	rt := runtime_entity.NewRuntime("k8s", 1, nil, "", "")
	rr := &pvcRuntimeRepo{runtime: rt}
	sr := &pvcStorageRepo{}
	p := &pvcFake{}
	c := &CreatePVCService{connector: &pvcTopConnector{pvc: p}, storageRepository: sr, runtimeRepository: rr}
	wf := workflow_entity.Workflow{Id: 2, Spec: workflow_entity.WorkflowSpec{StoragePolicy: workflow_entity.WorkflowSpecStoragePolicy{StorageSize: "1Gi", StorageClassName: "fast"}}}
	a := workflow_activity_entity.WorkflowActivities{Id: 3, Runtime: "k8s"}
	p.get.Metadata.Name = "existing"
	if name, err := c.GetOrCreatePersistentVolumeClainByActivity(wf, a, "ns"); err != nil || name != "existing" {
		t.Fatal(name, err)
	}
	p.getErr = errors.New("missing")
	p.create.Metadata.Name = "created"
	if name, err := c.GetOrCreatePersistentVolumeClainByActivity(wf, a, "ns"); err != nil || name != "created" {
		t.Fatal(name, err)
	}
	wf.Spec.StoragePolicy.Type = workflow_entity.MODE_DISTRIBUTED
	p.getErr = nil
	p.get.Metadata.Name = "shared"
	if _, err := c.GetOrCreatePersistentVolumeClainByActivity(wf, a, "ns"); err != nil {
		t.Fatal(err)
	}
	p.getErr = errors.New("missing")
	if _, err := c.GetOrCreatePersistentVolumeClainByActivity(wf, a, "ns"); err != nil {
		t.Fatal(err)
	}
}
func TestCreatePVCFailures(t *testing.T) {
	rt := runtime_entity.NewRuntime("k8s", 1, nil, "", "")
	rr := &pvcRuntimeRepo{runtime: rt}
	sr := &pvcStorageRepo{}
	p := &pvcFake{getErr: errors.New("missing")}
	c := &CreatePVCService{connector: &pvcTopConnector{pvc: p}, storageRepository: sr, runtimeRepository: rr}
	wf := workflow_entity.Workflow{}
	a := workflow_activity_entity.WorkflowActivities{Id: 3, Runtime: "k8s"}
	rr.err = errors.New("db")
	if _, err := c.GetOrCreatePersistentVolumeClainByActivity(wf, a, "ns"); err == nil {
		t.Fatal("runtime")
	}
	rr.err = nil
	p.createErr = errors.New("api")
	if _, err := c.handleCreatePersistentVolumeClain(wf, a, "ns"); err == nil {
		t.Fatal("create")
	}
	p.createErr = nil
	if _, err := c.handleCreatePersistentVolumeClain(wf, a, "ns"); err != nil {
		t.Fatal("empty name preserves current contract")
	}
	p.create.Metadata.Name = "p"
	sr.err = errors.New("db")
	if _, err := c.handleCreatePersistentVolumeClain(wf, a, "ns"); err == nil {
		t.Fatal("storage")
	}
}
