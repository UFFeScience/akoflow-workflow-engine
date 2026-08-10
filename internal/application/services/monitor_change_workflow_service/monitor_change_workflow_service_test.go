package monitor_change_workflow_service

import (
	"errors"
	"strings"
	"testing"

	workflow_activity_entity "github.com/UFFeScience/akoflow/internal/domain/workflow/activity"
	workflow_entity "github.com/UFFeScience/akoflow/internal/domain/workflow/definition"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/activity_repository"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/workflow_repository"
)

type pendingFake struct {
	workflows []workflow_entity.Workflow
	err       error
}

func (f pendingFake) GetPendingWorkflows() ([]workflow_entity.Workflow, error) {
	return f.workflows, f.err
}

type statusFake struct{}

func (statusFake) GetActivitiesByStatus(workflow workflow_entity.Workflow, status int) []workflow_activity_entity.WorkflowActivities {
	result := []workflow_activity_entity.WorkflowActivities{}
	for _, activity := range workflow.Spec.Activities {
		if activity.Status == status {
			result = append(result, activity)
		}
	}
	return result
}

type workflowRepositoryFake struct {
	workflow_repository.IWorkflowRepository
	updates map[int]int
	err     error
}

func (f *workflowRepositoryFake) UpdateStatus(id, status int) error {
	if f.err != nil {
		return f.err
	}
	f.updates[id] = status
	return nil
}

type runtimeFake struct {
	verified []int
}

func (f *runtimeFake) VerifyActivitiesWasFinished(workflow workflow_entity.Workflow) bool {
	f.verified = append(f.verified, workflow.Id)
	return true
}

func TestMonitorFinishesCompletedWorkflowsAndVerifiesRuntimes(t *testing.T) {
	completed := workflow_entity.Workflow{Id: 1, Spec: workflow_entity.WorkflowSpec{
		Runtime:    "k8s-a",
		Activities: []workflow_activity_entity.WorkflowActivities{{Id: 10, Status: activity_repository.StatusFinished, Runtime: "hpc-a"}},
	}}
	running := workflow_entity.Workflow{Id: 2, Spec: workflow_entity.WorkflowSpec{
		Activities: []workflow_activity_entity.WorkflowActivities{{Id: 20, Status: activity_repository.StatusRunning, Runtime: "local"}},
	}}
	repository := &workflowRepositoryFake{updates: map[int]int{}}
	runtimes := map[string]*runtimeFake{}
	service := NewWithDependencies(repository, pendingFake{workflows: []workflow_entity.Workflow{completed, running}}, statusFake{}, func(id string) RuntimeVerifier {
		if runtimes[id] == nil {
			runtimes[id] = &runtimeFake{}
		}
		return runtimes[id]
	})

	if err := service.MonitorChangeWorkflow(); err != nil {
		t.Fatalf("MonitorChangeWorkflow() error = %v", err)
	}
	if repository.updates[1] != workflow_repository.StatusFinished {
		t.Fatalf("completed workflow update = %v", repository.updates)
	}
	if _, updated := repository.updates[2]; updated {
		t.Fatalf("running workflow was finished: %v", repository.updates)
	}
	for _, runtimeID := range []string{"k8s-a", "hpc-a", "local"} {
		if runtimes[runtimeID] == nil || len(runtimes[runtimeID].verified) != 1 {
			t.Fatalf("runtime %s verification = %+v", runtimeID, runtimes[runtimeID])
		}
	}
}

func TestMonitorReturnsBoundaryErrors(t *testing.T) {
	t.Run("pending workflows", func(t *testing.T) {
		service := NewWithDependencies(&workflowRepositoryFake{}, pendingFake{err: errors.New("db down")}, statusFake{}, nil)
		err := service.MonitorChangeWorkflow()
		if err == nil || !strings.Contains(err.Error(), "load pending workflows") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("status update", func(t *testing.T) {
		workflow := workflow_entity.Workflow{Id: 3}
		service := NewWithDependencies(&workflowRepositoryFake{err: errors.New("write failed")}, pendingFake{workflows: []workflow_entity.Workflow{workflow}}, statusFake{}, func(string) RuntimeVerifier { return &runtimeFake{} })
		err := service.MonitorChangeWorkflow()
		if err == nil || !strings.Contains(err.Error(), "finish workflow 3") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("runtime missing", func(t *testing.T) {
		workflow := workflow_entity.Workflow{Id: 4, Spec: workflow_entity.WorkflowSpec{Activities: []workflow_activity_entity.WorkflowActivities{{Status: activity_repository.StatusRunning, Runtime: "gone"}}}}
		service := NewWithDependencies(&workflowRepositoryFake{updates: map[int]int{}}, pendingFake{workflows: []workflow_entity.Workflow{workflow}}, statusFake{}, func(string) RuntimeVerifier { return nil })
		err := service.MonitorChangeWorkflow()
		if err == nil || !strings.Contains(err.Error(), "runtime gone not found") {
			t.Fatalf("error = %v", err)
		}
	})
}

func TestNewUsesProductionDependencies(t *testing.T) {
	service := New()
	if service.workflowRepository == nil || service.getPendingWorkflowService == nil || service.getWorkflowByStatus == nil || service.runtimeResolver == nil {
		t.Fatalf("New() returned incomplete service: %+v", service)
	}
	if runtime := service.runtimeResolver("missing-runtime"); runtime != nil {
		t.Fatalf("missing runtime resolved to %T", runtime)
	}
}
