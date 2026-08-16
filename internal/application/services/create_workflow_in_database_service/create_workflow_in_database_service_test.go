package create_workflow_in_database_service

import (
	"errors"
	"testing"

	"github.com/UFFeScience/akoflow/internal/application/ports"
	workflow_activity_entity "github.com/UFFeScience/akoflow/internal/domain/workflow/activity"
	workflow_entity "github.com/UFFeScience/akoflow/internal/domain/workflow/definition"
)

type workflowRepoFake struct {
	ports.WorkflowRepository
	id                 int
	createErr, findErr error
	found              workflow_entity.Workflow
}

func (f *workflowRepoFake) Create(string, workflow_entity.Workflow) (int, error) {
	return f.id, f.createErr
}
func (f *workflowRepoFake) Find(int) (workflow_entity.Workflow, error) { return f.found, f.findErr }

type activityRepoFake struct {
	ports.ActivityRepository
	err        error
	calls      int
	activities []workflow_activity_entity.WorkflowActivities
}

func (f *activityRepoFake) Create(_ string, _ workflow_entity.Workflow, a []workflow_activity_entity.WorkflowActivities) error {
	f.calls++
	f.activities = a
	return f.err
}

type storageCreatorFake struct {
	err       error
	id, calls int
}

func (f *storageCreatorFake) CreateByWorkflow(id int) error { f.id = id; f.calls++; return f.err }

func TestCreatePersistsWorkflowActivitiesAndStorage(t *testing.T) {
	wf := workflow_entity.Workflow{Spec: workflow_entity.WorkflowSpec{Activities: []workflow_activity_entity.WorkflowActivities{{Id: 1}}}}
	workflows := &workflowRepoFake{id: 7, found: workflow_entity.Workflow{Id: 7}}
	activities := &activityRepoFake{}
	storage := &storageCreatorFake{}
	service := NewWithDependencies("lab", workflows, activities, storage)
	id, err := service.Create(wf)
	if err != nil || id != 7 || activities.calls != 1 || len(activities.activities) != 1 || storage.id != 7 {
		t.Fatalf("id=%d activities=%+v storage=%+v err=%v", id, activities, storage, err)
	}
}

func TestCreateStopsAtFailedBoundary(t *testing.T) {
	tests := []struct {
		name                      string
		workflows                 *workflowRepoFake
		activityErr, storageErr   error
		wantActivity, wantStorage int
	}{
		{"create workflow", &workflowRepoFake{createErr: errors.New("create")}, nil, nil, 0, 0},
		{"find workflow", &workflowRepoFake{id: 7, findErr: errors.New("find")}, nil, nil, 0, 0},
		{"create activities", &workflowRepoFake{id: 7, found: workflow_entity.Workflow{Id: 7}}, errors.New("activities"), nil, 1, 0},
		{"create storage", &workflowRepoFake{id: 7, found: workflow_entity.Workflow{Id: 7}}, nil, errors.New("storage"), 1, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &activityRepoFake{err: tt.activityErr}
			s := &storageCreatorFake{err: tt.storageErr}
			_, err := NewWithDependencies("lab", tt.workflows, a, s).Create(workflow_entity.Workflow{})
			if err == nil || a.calls != tt.wantActivity || s.calls != tt.wantStorage {
				t.Fatalf("activity=%d storage=%d err=%v", a.calls, s.calls, err)
			}
		})
	}
}

func TestNewInitializesProductionDependencies(t *testing.T) {
	if s := New(); s.workflowRepository == nil || s.activityRepository == nil || s.createStorageInDatabaseService == nil {
		t.Fatalf("incomplete: %+v", s)
	}
}
