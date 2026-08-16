package create_storage_in_database_service

import (
	"errors"
	"testing"

	"github.com/UFFeScience/akoflow/internal/application/ports"
	"github.com/UFFeScience/akoflow/internal/application/services/get_map_activities_keep_disk_service"
	workflow_entity "github.com/UFFeScience/akoflow/internal/domain/workflow/definition"
)

type workflowFake struct {
	ports.WorkflowRepository
	workflow workflow_entity.Workflow
	err      error
}

func (f workflowFake) Find(int) (workflow_entity.Workflow, error) { return f.workflow, f.err }

type storageFake struct {
	ports.StorageRepository
	params ports.CreateStorageParams
	err    error
	calls  int
}

func (f *storageFake) Create(p ports.CreateStorageParams) error {
	f.params = p
	f.calls++
	return f.err
}

type keepDiskFake struct {
	values get_map_activities_keep_disk_service.MapActivitiesKeepDisk
	err    error
}

func (f keepDiskFake) GetMapActivitiesKeepDisk(int) (get_map_activities_keep_disk_service.MapActivitiesKeepDisk, error) {
	return f.values, f.err
}

func TestCreateByWorkflowBuildsStorageDefinition(t *testing.T) {
	wf := workflow_entity.Workflow{Id: 9, Spec: workflow_entity.WorkflowSpec{MountPath: "/data", StoragePolicy: workflow_entity.WorkflowSpecStoragePolicy{StorageClassName: "fast", StorageSize: "20Gi"}}}
	storage := &storageFake{}
	service := NewWithDependencies("lab", workflowFake{workflow: wf}, storage, keepDiskFake{values: map[int]bool{1: true}})
	if err := service.CreateByWorkflow(9); err != nil {
		t.Fatal(err)
	}
	p := storage.params
	if p.WorkflowID != 9 || p.Namespace != "lab" || p.StorageClass != "fast" || !p.ActivitiesKeepingDisk[1] {
		t.Fatalf("params=%+v", p)
	}
}
func TestCreateByWorkflowPropagatesErrors(t *testing.T) {
	cases := []CreateStorageInDatabaseService{NewWithDependencies("lab", workflowFake{err: errors.New("find")}, &storageFake{}, keepDiskFake{}), NewWithDependencies("lab", workflowFake{workflow: workflow_entity.Workflow{Id: 1}}, &storageFake{}, keepDiskFake{err: errors.New("map")}), NewWithDependencies("lab", workflowFake{workflow: workflow_entity.Workflow{Id: 1}}, &storageFake{err: errors.New("create")}, keepDiskFake{values: map[int]bool{}})}
	for i := range cases {
		if err := cases[i].CreateByWorkflow(1); err == nil {
			t.Fatalf("case %d expected error", i)
		}
	}
}
func TestNewInitializesDependencies(t *testing.T) {
	s := New()
	if s.workflowRepository == nil || s.storageRepository == nil || s.getMapActivitiesKeepDiskService == nil {
		t.Fatalf("incomplete: %+v", s)
	}
}
