package get_activity_dependencies_service

import (
	"reflect"
	"testing"

	"github.com/UFFeScience/akoflow/internal/application/ports"
	workflow_activity_entity "github.com/UFFeScience/akoflow/internal/domain/workflow/activity"
	workflow_entity "github.com/UFFeScience/akoflow/internal/domain/workflow/definition"
)

type workflowFake struct{ ports.WorkflowRepository }

func (workflowFake) Find(id int) (workflow_entity.Workflow, error) {
	return workflow_entity.Workflow{Id: id}, nil
}

type activityFake struct {
	ports.ActivityRepository
	activities ports.ActivitiesByWorkflow
	deps       []workflow_activity_entity.WorkflowActivityDependencyDatabase
}

func (f activityFake) GetActivitiesByWorkflowIds([]int) (ports.ActivitiesByWorkflow, error) {
	return f.activities, nil
}
func (f activityFake) GetWfaDependencies(int) ([]workflow_activity_entity.WorkflowActivityDependencyDatabase, error) {
	return f.deps, nil
}

func TestDependencyQueriesIncludeDirectAndTransitiveAncestors(t *testing.T) {
	activities := ports.ActivitiesByWorkflow{1: {{Id: 1, Name: "a"}, {Id: 2, Name: "b"}, {Id: 3, Name: "c"}}}
	deps := []workflow_activity_entity.WorkflowActivityDependencyDatabase{{ActivityId: 2, DependsOnId: 1}, {ActivityId: 3, DependsOnId: 2}}
	service := NewWithDependencies(workflowFake{}, activityFake{activities: activities, deps: deps})
	all := service.GetActivityDependencies(1)
	if got := []int{all[3][0].Id, all[3][1].Id}; !reflect.DeepEqual(got, []int{1, 2}) {
		t.Fatalf("transitive=%v", got)
	}
	direct := service.GetActivityDependenciesByActivity(1, 3)
	if len(direct[3]) != 1 || direct[3][0].Id != 2 {
		t.Fatalf("direct=%v", direct)
	}
	byWorkflow := service.GetActivityDependenciesByWorkflow(1)
	if len(byWorkflow[2]) != 1 || len(byWorkflow[3]) != 1 {
		t.Fatalf("byWorkflow=%v", byWorkflow)
	}
}
func TestDependencyHelpersHandleMissingAndSort(t *testing.T) {
	service := GetActivityDependenciesService{}
	if got := service.fillActivityDependencies(99, map[int]workflow_activity_entity.WorkflowActivities{}, nil); len(got) != 0 {
		t.Fatalf("missing=%v", got)
	}
	got := service.setDependenciesToArray(map[int]workflow_activity_entity.WorkflowActivities{3: {Id: 3}, 1: {Id: 1}, 2: {Id: 2}})
	if got[0].Id != 1 || got[1].Id != 2 || got[2].Id != 3 {
		t.Fatalf("sorted=%v", got)
	}
}
func TestNewInitializesDependencies(t *testing.T) {
	s := New()
	if s.workflowRepository == nil || s.activityRepository == nil {
		t.Fatalf("incomplete: %+v", s)
	}
}
