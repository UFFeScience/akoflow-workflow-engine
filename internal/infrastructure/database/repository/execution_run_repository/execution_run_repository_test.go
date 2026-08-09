package execution_run_repository

import (
	"path/filepath"
	"testing"

	"github.com/UFFeScience/akoflow/internal/domain"
)

func setupRepository(t *testing.T) IRepository {
	t.Helper()
	t.Setenv("AKOFLOW_DATABASE_PATH", filepath.Join(t.TempDir(), "db.sqlite"))
	return New()
}

func TestExecutionRunCreateAndComplete(t *testing.T) {
	repository := setupRepository(t)
	run := domain.ExecutionRun{ID: "run", SchedulePlanID: "plan", Mode: domain.ExecutionModeSimulation, Seed: 1, Status: domain.ExecutionRunRunning, EnvironmentSnapshotID: "snapshot"}
	if err := repository.Create(run); err != nil {
		t.Fatal(err)
	}
	trace := domain.ExecutionTrace{RunID: "run", Mode: domain.ExecutionModeSimulation, Executed: domain.ExecutionMetrics{MakespanSeconds: 10, Cost: 2}, Tasks: []domain.TaskExecution{{ID: "task", PlanAssignmentID: "assignment", ActivityID: "activity", PlannedResourceID: "r1", AllocatedResourceID: "r2", Attempt: 1, Status: domain.TaskCompleted, ReadyAt: 1, DataReadyAt: 2, QueuedAt: 3, StartedAt: 4, FinishedAt: 5, RuntimeSeconds: 1, Cost: .5}}, Transfers: []domain.DataTransfer{{ID: "transfer", ProducerActivityID: "a", ConsumerActivityID: "b", SourceResourceID: "r1", TargetResourceID: "r2", Bytes: 10, StartedAt: 1, FinishedAt: 2, DurationSeconds: 1, Cost: .1}}}
	if err := repository.Complete(trace); err != nil {
		t.Fatal(err)
	}
	if err := repository.Create(run); err == nil {
		t.Fatal("duplicate run must fail")
	}
}

func TestCompleteRejectsDuplicateTask(t *testing.T) {
	repository := setupRepository(t)
	if err := repository.Create(domain.ExecutionRun{ID: "run", Mode: domain.ExecutionModeReal, Status: domain.ExecutionRunRunning}); err != nil {
		t.Fatal(err)
	}
	trace := domain.ExecutionTrace{RunID: "run", Tasks: []domain.TaskExecution{{ID: "same"}, {ID: "same"}}}
	if err := repository.Complete(trace); err == nil {
		t.Fatal("duplicate task execution must fail")
	}
}
