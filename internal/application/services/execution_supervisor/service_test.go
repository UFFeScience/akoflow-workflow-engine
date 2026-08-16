package execution_supervisor

import (
	"context"
	"testing"
	"time"

	"github.com/UFFeScience/akoflow/internal/application/ports"
	"github.com/UFFeScience/akoflow/internal/domain"
)

type executionStoreFake struct {
	runs    []domain.ExecutionRun
	tasks   []domain.TaskExecution
	trace   domain.ExecutionTrace
	handles map[string]domain.ActivityHandle
	failed  string
}

func (f *executionStoreFake) CreateRun(_ context.Context, run domain.ExecutionRun) error {
	f.runs = append(f.runs, run)
	return nil
}
func (f *executionStoreFake) FindRun(context.Context, string) (*domain.ExecutionRun, error) {
	return nil, nil
}
func (f *executionStoreFake) SaveTask(_ context.Context, task domain.TaskExecution) error {
	f.tasks = append(f.tasks, task)
	return nil
}
func (f *executionStoreFake) CompleteRun(_ context.Context, trace domain.ExecutionTrace) error {
	f.trace = trace
	return nil
}
func (f *executionStoreFake) FailRun(_ context.Context, _ string, reason string) error {
	f.failed = reason
	return nil
}
func (f *executionStoreFake) Save(_ context.Context, h domain.ActivityHandle) error {
	if f.handles == nil {
		f.handles = map[string]domain.ActivityHandle{}
	}
	f.handles[h.ID] = h
	return nil
}
func (f *executionStoreFake) Find(_ context.Context, id string) (*domain.ActivityHandle, error) {
	h, ok := f.handles[id]
	if !ok {
		return nil, nil
	}
	return &h, nil
}

type activityControllerFake struct {
	started     []string
	inspections map[string]int
}

func (f *activityControllerFake) Start(_ context.Context, execution domain.ActivityExecutionContext) (domain.ActivityHandle, error) {
	f.started = append(f.started, execution.Activity.ID)
	return domain.ActivityHandle{ID: "h-" + execution.Activity.ID, RunID: execution.Run.ID, ActivityID: execution.Activity.ID, ResourceID: execution.Resource.ID, RuntimeID: execution.Resource.RuntimeID, Status: domain.HandleRunning, StartedAt: 1}, nil
}
func (f *activityControllerFake) Inspect(_ context.Context, id string, _ domain.ExecutionMode) (*domain.ActivityHandle, error) {
	if f.inspections == nil {
		f.inspections = map[string]int{}
	}
	f.inspections[id]++
	return &domain.ActivityHandle{ID: id, Status: domain.HandleCompleted, StartedAt: 1, FinishedAt: 2}, nil
}

type planExecutorFake struct{ called bool }

func (f *planExecutorFake) Execute(_ context.Context, request ports.ExecutionRequest) (domain.ExecutionTrace, error) {
	f.called = true
	return domain.ExecutionTrace{RunID: request.Run.ID, PlanID: request.Plan.ID, Mode: request.Run.Mode}, nil
}

func requestFixture(mode domain.ExecutionMode) ports.ExecutionRequest {
	simulation := &domain.ActivitySimulation{DurationSeconds: 1}
	return ports.ExecutionRequest{Run: domain.ExecutionRun{ID: "run", Mode: mode}, Plan: domain.SchedulePlan{ID: "plan", Assignments: []domain.PlanAssignment{{ID: "pa", ActivityID: "a", ResourceID: "r"}, {ID: "pb", ActivityID: "b", ResourceID: "r"}}}, Workflow: domain.WorkflowVersion{ID: "workflow", Activities: []domain.Activity{{ID: "a", Name: "a", Kind: domain.ActivityKindTask, Capabilities: []domain.ActivityCapability{domain.ActivityCapabilityReal}, Command: domain.ActivityCommand{Entrypoint: "true"}, Simulation: simulation}, {ID: "b", Name: "b", Kind: domain.ActivityKindTask, Capabilities: []domain.ActivityCapability{domain.ActivityCapabilityReal}, Command: domain.ActivityCommand{Entrypoint: "true"}, Simulation: simulation}}, Dependencies: []domain.ActivityDependency{{ActivityID: "b", DependsOnActivityID: "a"}}}, Resources: []domain.Resource{{ID: "r", RuntimeID: "local"}}}
}

func TestSupervisorExecutesDAGInDependencyOrder(t *testing.T) {
	store := &executionStoreFake{}
	activities := &activityControllerFake{}
	simulator := &planExecutorFake{}
	service, err := New(store, activities, simulator, Config{PollInterval: time.Microsecond, MaxParallel: 2})
	if err != nil {
		t.Fatal(err)
	}
	trace, err := service.Execute(context.Background(), requestFixture(domain.ExecutionModeReal))
	if err != nil {
		t.Fatal(err)
	}
	if len(activities.started) != 2 || activities.started[0] != "a" || activities.started[1] != "b" {
		t.Fatalf("order=%v", activities.started)
	}
	if trace.RunID != "run" || store.trace.RunID != "run" {
		t.Fatal("trace was not completed")
	}
}

func TestSupervisorDelegatesSimulation(t *testing.T) {
	store := &executionStoreFake{}
	simulator := &planExecutorFake{}
	service, _ := New(store, &activityControllerFake{}, simulator, Config{})
	request := requestFixture(domain.ExecutionModeSimulation)
	if _, err := service.Execute(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if !simulator.called {
		t.Fatal("simulator was not called")
	}
}

func TestSupervisorRejectsIncompletePlan(t *testing.T) {
	service, _ := New(&executionStoreFake{}, &activityControllerFake{}, &planExecutorFake{}, Config{})
	request := requestFixture(domain.ExecutionModeReal)
	request.Plan.Assignments = nil
	if _, err := service.Execute(context.Background(), request); err == nil {
		t.Fatal("incomplete plan must fail")
	}
}
