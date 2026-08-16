package list_workflows_api_service

import (
	"errors"
	"testing"

	"github.com/UFFeScience/akoflow/internal/application/ports"
	workflow_entity "github.com/UFFeScience/akoflow/internal/domain/workflow/definition"
)

type workflowFake struct {
	ports.WorkflowRepository
	values []workflow_entity.Workflow
	err    error
}

func (f workflowFake) ListAllWorkflows(*ports.WorkflowListOptions) ([]workflow_entity.Workflow, error) {
	return f.values, f.err
}

type activityFake struct {
	ports.ActivityRepository
	values ports.ActivitiesByWorkflow
	err    error
	ids    []int
}

func (f *activityFake) GetActivitiesByWorkflowIds(ids []int) (ports.ActivitiesByWorkflow, error) {
	f.ids = ids
	return f.values, f.err
}

func TestListHydratesAndMapsWorkflows(t *testing.T) {
	activities := &activityFake{values: ports.ActivitiesByWorkflow{1: {{Id: 10, Name: "task"}}}}
	service := NewWithDependencies(workflowFake{values: []workflow_entity.Workflow{{Id: 1, Name: "wf"}}}, activities)
	result, err := service.ListAllWorkflows()
	if err != nil || len(result) != 1 || len(activities.ids) != 1 || activities.ids[0] != 1 {
		t.Fatalf("result=%v ids=%v err=%v", result, activities.ids, err)
	}
}
func TestListPropagatesRepositoryErrors(t *testing.T) {
	if _, err := NewWithDependencies(workflowFake{err: errors.New("wf")}, &activityFake{}).ListAllWorkflows(); err == nil {
		t.Fatal("expected workflow error")
	}
	if _, err := NewWithDependencies(workflowFake{values: []workflow_entity.Workflow{{Id: 1}}}, &activityFake{err: errors.New("activity")}).ListAllWorkflows(); err == nil {
		t.Fatal("expected activity error")
	}
}
func TestNewInitializesDependencies(t *testing.T) {
	s := New()
	if s.workflowRepository == nil || s.activityRepository == nil {
		t.Fatalf("incomplete: %+v", s)
	}
}
