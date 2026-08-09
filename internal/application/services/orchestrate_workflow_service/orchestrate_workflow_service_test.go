package orchestrate_workflow_service

import (
	"errors"
	"reflect"
	"testing"

	workflow_activity_entity "github.com/UFFeScience/akoflow/internal/domain/workflow/activity"
	workflow_entity "github.com/UFFeScience/akoflow/internal/domain/workflow/definition"
	"github.com/UFFeScience/akoflow/internal/execution/lifecycle/channel"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/activity_repository"
)

type statusSelector struct{}

func (statusSelector) GetActivitiesByStatus(workflow workflow_entity.Workflow, status int) []workflow_activity_entity.WorkflowActivities {
	result := make([]workflow_activity_entity.WorkflowActivities, 0)
	for _, activity := range workflow.Spec.Activities {
		if activity.Status == status {
			result = append(result, activity)
		}
	}
	return result
}

func workflowWithActivities(activities ...workflow_activity_entity.WorkflowActivities) workflow_entity.Workflow {
	return workflow_entity.Workflow{Id: 10, Spec: workflow_entity.WorkflowSpec{Activities: activities}}
}

func TestOrchestrateDispatchesRootAndUnblockedActivities(t *testing.T) {
	manager := &channel.Manager{WorfklowChannel: make(chan channel.DataChannel, 4)}
	scheduledIDs := []int{}
	service := NewWithDependencies("science", manager, statusSelector{}, func(_ workflow_entity.Workflow, activities []workflow_activity_entity.WorkflowActivities) ([]workflow_activity_entity.WorkflowActivities, error) {
		for _, activity := range activities {
			scheduledIDs = append(scheduledIDs, activity.Id)
		}
		return activities, nil
	})
	workflow := workflowWithActivities(
		workflow_activity_entity.WorkflowActivities{Id: 1, Name: "root", Status: activity_repository.StatusCreated},
		workflow_activity_entity.WorkflowActivities{Id: 2, Name: "done", Status: activity_repository.StatusFinished},
		workflow_activity_entity.WorkflowActivities{Id: 3, Name: "unblocked", DependsOn: []string{"done"}, Status: activity_repository.StatusCreated},
		workflow_activity_entity.WorkflowActivities{Id: 4, Name: "blocked", DependsOn: []string{"missing"}, Status: activity_repository.StatusCreated},
	)

	result := service.Orchestrate([]workflow_entity.Workflow{workflow})

	if !reflect.DeepEqual(scheduledIDs, []int{1, 3}) {
		t.Fatalf("scheduled IDs = %v", scheduledIDs)
	}
	if got := result[10]; len(got) != 2 || got[0].Id != 1 || got[1].Id != 3 {
		t.Fatalf("orchestrated activities = %+v", got)
	}
	for _, wantID := range []int{1, 3} {
		event := <-manager.WorfklowChannel
		if event.Id != wantID || event.Namespace != "science" {
			t.Fatalf("event = %+v, want ID %d", event, wantID)
		}
	}
}

func TestOrchestrateSkipsEntireWorkflowWhileAnActivityIsSyncing(t *testing.T) {
	manager := &channel.Manager{WorfklowChannel: make(chan channel.DataChannel, 1)}
	schedulerCalls := 0
	service := NewWithDependencies("science", manager, statusSelector{}, func(_ workflow_entity.Workflow, activities []workflow_activity_entity.WorkflowActivities) ([]workflow_activity_entity.WorkflowActivities, error) {
		schedulerCalls++
		return activities, nil
	})
	workflow := workflowWithActivities(
		workflow_activity_entity.WorkflowActivities{Id: 1, Status: activity_repository.StatusCreated},
		workflow_activity_entity.WorkflowActivities{Id: 2, Status: activity_repository.StatusSyncing},
	)

	result := service.Orchestrate([]workflow_entity.Workflow{workflow})

	if schedulerCalls != 0 || len(result[10]) != 0 || len(manager.WorfklowChannel) != 0 {
		t.Fatalf("schedulerCalls=%d result=%v queued=%d", schedulerCalls, result, len(manager.WorfklowChannel))
	}
}

func TestOrchestrateUsesUnscheduledActivitiesWhenSchedulerFails(t *testing.T) {
	manager := &channel.Manager{WorfklowChannel: make(chan channel.DataChannel, 1)}
	service := NewWithDependencies("science", manager, statusSelector{}, func(_ workflow_entity.Workflow, _ []workflow_activity_entity.WorkflowActivities) ([]workflow_activity_entity.WorkflowActivities, error) {
		return nil, errors.New("plugin unavailable")
	})
	workflow := workflowWithActivities(workflow_activity_entity.WorkflowActivities{Id: 8, Name: "root", Status: activity_repository.StatusCreated})

	result := service.Orchestrate([]workflow_entity.Workflow{workflow})

	if len(result[10]) != 1 || result[10][0].Id != 8 {
		t.Fatalf("result = %+v", result)
	}
	if event := <-manager.WorfklowChannel; event.Id != 8 {
		t.Fatalf("event = %+v", event)
	}
}

func TestDependencyRequiresEveryPredecessor(t *testing.T) {
	service := NewWithDependencies("science", &channel.Manager{WorfklowChannel: make(chan channel.DataChannel, 1)}, statusSelector{}, func(_ workflow_entity.Workflow, activities []workflow_activity_entity.WorkflowActivities) ([]workflow_activity_entity.WorkflowActivities, error) {
		return activities, nil
	})
	pending := workflow_activity_entity.WorkflowActivities{Id: 3, DependsOn: []string{"a", "b"}}

	if service.isDependentOnFinished(pending, []workflow_activity_entity.WorkflowActivities{{Name: "a"}}) {
		t.Fatal("activity was released with an unfinished predecessor")
	}
	if !service.isDependentOnFinished(pending, []workflow_activity_entity.WorkflowActivities{{Name: "a"}, {Name: "b"}}) {
		t.Fatal("activity was not released after every predecessor finished")
	}
}
