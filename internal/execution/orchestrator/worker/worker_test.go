package worker

import (
	"errors"
	"reflect"
	"testing"

	"github.com/UFFeScience/akoflow/internal/application/services/run_activity_in_cluster_service"
	"github.com/UFFeScience/akoflow/internal/domain"
	workflow_activity_entity "github.com/UFFeScience/akoflow/internal/domain/workflow/activity"
	workflow_entity "github.com/UFFeScience/akoflow/internal/domain/workflow/definition"
	"github.com/UFFeScience/akoflow/internal/execution/lifecycle/channel"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/activity_repository"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/resource_repository"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/workflow_repository"
)

type runnerFake struct {
	ids      []int
	failures map[int]error
}

func (f *runnerFake) Run(activityID int) error {
	f.ids = append(f.ids, activityID)
	return f.failures[activityID]
}

func TestWorkerConsumesActivitiesUntilStopSignal(t *testing.T) {
	events := make(chan channel.DataChannel, 4)
	runner := &runnerFake{failures: map[int]error{2: errors.New("runtime failed")}}
	worker := NewWithDependencies(events, runner)

	events <- channel.DataChannel{Id: 1}
	events <- channel.DataChannel{Id: 2}
	events <- channel.DataChannel{Id: 3}
	events <- channel.DataChannel{Id: FLAG_ID_WORKER_STOP_LISTENING}
	worker.StartWorker()

	if !reflect.DeepEqual(runner.ids, []int{1, 2, 3}) {
		t.Fatalf("processed IDs = %v", runner.ids)
	}
}

func TestWorkerStopsWhenChannelCloses(t *testing.T) {
	events := make(chan channel.DataChannel)
	close(events)
	runner := &runnerFake{failures: map[int]error{}}

	NewWithDependencies(events, runner).StartWorker()

	if len(runner.ids) != 0 {
		t.Fatalf("processed IDs = %v", runner.ids)
	}
}

func TestWorkerDoesNotDispatchStopSignal(t *testing.T) {
	events := make(chan channel.DataChannel, 1)
	events <- channel.DataChannel{Id: FLAG_ID_WORKER_STOP_LISTENING}
	runner := &runnerFake{failures: map[int]error{}}

	NewWithDependencies(events, runner).StartWorker()

	if len(runner.ids) != 0 {
		t.Fatalf("stop signal was dispatched: %v", runner.ids)
	}
}

type endToEndActivityRepository struct {
	activity_repository.IActivityRepository
}

func (endToEndActivityRepository) Find(id int) (workflow_activity_entity.WorkflowActivities, error) {
	return workflow_activity_entity.WorkflowActivities{Id: id, WorkflowId: 99}, nil
}

func (endToEndActivityRepository) GetActivityScheduleByActivityId(id int) (workflow_activity_entity.ActivitySchedule, error) {
	return workflow_activity_entity.ActivitySchedule{WorkflowID: 99, ActivityID: id, ResourceID: "edge-3"}, nil
}

type endToEndWorkflowRepository struct {
	workflow_repository.IWorkflowRepository
}

func (endToEndWorkflowRepository) Find(id int) (workflow_entity.Workflow, error) {
	return workflow_entity.Workflow{Id: id}, nil
}

type endToEndResourceRepository struct {
	resource_repository.IRepository
}

func (endToEndResourceRepository) FindByID(id string) (*domain.Resource, error) {
	return &domain.Resource{ID: id, RuntimeID: "local://edge-3"}, nil
}

type endToEndRuntime struct {
	workflowID int
	activityID int
	calls      int
}

func (r *endToEndRuntime) ApplyJob(workflowID, activityID int) bool {
	r.workflowID, r.activityID = workflowID, activityID
	r.calls++
	return true
}

func TestActivityEventReachesAssignedRuntimeEndToEnd(t *testing.T) {
	runtime := &endToEndRuntime{}
	runner := run_activity_in_cluster_service.NewWithDependencies(
		endToEndWorkflowRepository{},
		endToEndActivityRepository{},
		endToEndResourceRepository{},
		func(runtimeID string) run_activity_in_cluster_service.RuntimeExecutor {
			if runtimeID != "local://edge-3" {
				t.Fatalf("runtime ID = %q", runtimeID)
			}
			return runtime
		},
	)
	events := make(chan channel.DataChannel, 2)
	events <- channel.DataChannel{Id: 73, Namespace: "science"}
	events <- channel.DataChannel{Id: FLAG_ID_WORKER_STOP_LISTENING}

	NewWithDependencies(events, runner).StartWorker()

	if runtime.calls != 1 || runtime.workflowID != 99 || runtime.activityID != 73 {
		t.Fatalf("runtime calls=%d workflow=%d activity=%d", runtime.calls, runtime.workflowID, runtime.activityID)
	}
}
