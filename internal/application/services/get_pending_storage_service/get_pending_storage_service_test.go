package get_pending_storage_service

import (
	"errors"
	"testing"

	"github.com/UFFeScience/akoflow/internal/application/ports"
	workflow_activity_entity "github.com/UFFeScience/akoflow/internal/domain/workflow/activity"
)

type storageFake struct {
	ports.StorageRepository
	values []ports.Storage
	err    error
}

func (f storageFake) GetCreatedStorages(string) ([]ports.Storage, error) { return f.values, f.err }

type activityFake struct {
	ports.ActivityRepository
	values ports.ActivitiesByWorkflow
	err    error
}

func (f activityFake) GetActivitiesByWorkflowIds([]int) (ports.ActivitiesByWorkflow, error) {
	return f.values, f.err
}

type statusFake struct{}

func (statusFake) GetActivitiesByStatuses(values []workflow_activity_entity.WorkflowActivities, status int) []workflow_activity_entity.WorkflowActivities {
	result := []workflow_activity_entity.WorkflowActivities{}
	for _, v := range values {
		if v.Status == status {
			result = append(result, v)
		}
	}
	return result
}

type dependenciesFake struct {
	values workflow_activity_entity.MapActivityDependencies
}

func (f dependenciesFake) GetActivityDependenciesByWorkflow(int) workflow_activity_entity.MapActivityDependencies {
	return f.values
}

func TestPendingStoragesReturnsFinishedProducerAfterConsumersStarted(t *testing.T) {
	activities := ports.ActivitiesByWorkflow{1: {{Id: 1, Status: ports.ActivityStatusFinished}, {Id: 2, Status: ports.ActivityStatusRunning}}}
	storages := []ports.Storage{{ID: 10, WorkflowID: 1, ActivityID: 1, Status: ports.StorageStatusCreated, KeepStorageAfterFinish: 1}, {ID: 20, WorkflowID: 1, ActivityID: 2, Status: ports.StorageStatusCreated}}
	deps := workflow_activity_entity.MapActivityDependencies{2: {{Id: 1}}}
	service := NewWithDependencies("lab", activityFake{values: activities}, storageFake{values: storages}, statusFake{}, dependenciesFake{values: deps})
	result, err := service.GetPendingStorages()
	if err != nil || len(result) != 1 || result[0].Id != 1 || !result[0].KeepDisk {
		t.Fatalf("result=%v err=%v", result, err)
	}
}
func TestPendingStoragesFiltersCompletedAndUnreferencedStorage(t *testing.T) {
	activities := ports.ActivitiesByWorkflow{1: {{Id: 1, Status: ports.ActivityStatusFinished}}}
	service := NewWithDependencies("lab", activityFake{values: activities}, storageFake{values: []ports.Storage{{ID: 1, WorkflowID: 1, ActivityID: 1, Status: ports.StorageStatusCompleted}}}, statusFake{}, dependenciesFake{})
	result, err := service.GetPendingStorages()
	if err != nil || len(result) != 0 {
		t.Fatalf("result=%v err=%v", result, err)
	}
}
func TestPendingStoragesPropagatesRepositoryErrors(t *testing.T) {
	cases := []GetPendingStorageService{NewWithDependencies("lab", activityFake{}, storageFake{err: errors.New("storage")}, statusFake{}, dependenciesFake{}), NewWithDependencies("lab", activityFake{err: errors.New("activity")}, storageFake{values: []ports.Storage{{WorkflowID: 1}}}, statusFake{}, dependenciesFake{})}
	for i := range cases {
		if _, err := cases[i].GetPendingStorages(); err == nil {
			t.Fatalf("case %d expected error", i)
		}
	}
}
func TestNewInitializesDependencies(t *testing.T) {
	s := New()
	if s.activityRepository == nil || s.storageRepository == nil || s.getWorkflowByStatusService == nil || s.getActivityDependencies == nil {
		t.Fatalf("incomplete: %+v", s)
	}
}
