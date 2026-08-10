package get_pending_workflow_service

import (
	"errors"
	"testing"

	workflow_entity "github.com/UFFeScience/akoflow/internal/domain/workflow/definition"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/activity_repository"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/workflow_repository"
)

type workflowRepositoryFake struct {
	workflow_repository.IWorkflowRepository
	workflows []workflow_entity.Workflow
	err       error
	namespace string
}

func (f *workflowRepositoryFake) GetPendingWorkflows(namespace string) ([]workflow_entity.Workflow, error) {
	f.namespace = namespace
	return f.workflows, f.err
}

type activityRepositoryFake struct {
	activity_repository.IActivityRepository
	result activity_repository.ResultGetActivitiesByWorkflowIds
	err    error
	ids    []int
}

func (f *activityRepositoryFake) GetActivitiesByWorkflowIds(ids []int) (activity_repository.ResultGetActivitiesByWorkflowIds, error) {
	f.ids = append([]int(nil), ids...)
	return f.result, f.err
}

func TestGetPendingWorkflowsHydratesActivities(t *testing.T) {
	workflows := &workflowRepositoryFake{workflows: []workflow_entity.Workflow{{Id: 4}, {Id: 9}}}
	activities := &activityRepositoryFake{result: activity_repository.ResultGetActivitiesByWorkflowIds{
		4: {{Id: 40, WorkflowId: 4}},
		9: {{Id: 90, WorkflowId: 9}},
	}}
	service := NewWithDependencies("lab", workflows, activities)

	result, err := service.GetPendingWorkflows()

	if err != nil {
		t.Fatalf("GetPendingWorkflows() error = %v", err)
	}
	if workflows.namespace != "lab" || len(activities.ids) != 2 || activities.ids[0] != 4 || activities.ids[1] != 9 {
		t.Fatalf("namespace=%q ids=%v", workflows.namespace, activities.ids)
	}
	if len(result[0].Spec.Activities) != 1 || result[0].Spec.Activities[0].Id != 40 || len(result[1].Spec.Activities) != 1 || result[1].Spec.Activities[0].Id != 90 {
		t.Fatalf("hydrated workflows = %+v", result)
	}
}

func TestGetPendingWorkflowsPropagatesRepositoryFailures(t *testing.T) {
	t.Run("workflow repository", func(t *testing.T) {
		service := NewWithDependencies("lab", &workflowRepositoryFake{err: errors.New("database offline")}, &activityRepositoryFake{})
		if _, err := service.GetPendingWorkflows(); err == nil {
			t.Fatal("expected workflow repository error")
		}
	})

	t.Run("activity repository", func(t *testing.T) {
		service := NewWithDependencies("lab", &workflowRepositoryFake{workflows: []workflow_entity.Workflow{{Id: 4}}}, &activityRepositoryFake{err: errors.New("database offline")})
		if _, err := service.GetPendingWorkflows(); err == nil {
			t.Fatal("expected activity repository error")
		}
	})

	t.Run("dependencies not initialized", func(t *testing.T) {
		service := NewWithDependencies("lab", nil, nil)
		if _, err := service.GetPendingWorkflows(); err == nil {
			t.Fatal("expected initialization error")
		}
	})

	t.Run("activity dependency not initialized", func(t *testing.T) {
		service := NewWithDependencies("lab", &workflowRepositoryFake{}, nil)
		if _, err := service.GetPendingWorkflows(); err == nil {
			t.Fatal("expected activity repository initialization error")
		}
	})
}

func TestNewUsesApplicationDependencies(t *testing.T) {
	service := New()
	if service.namespace == "" || service.workflowRepository == nil || service.activityRepository == nil {
		t.Fatalf("New() returned incomplete service: %+v", service)
	}
}
