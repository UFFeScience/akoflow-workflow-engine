package execution

import (
	"context"
	"database/sql"
	"testing"

	"github.com/UFFeScience/akoflow/internal/domain"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/schema"
	_ "github.com/mattn/go-sqlite3"
)

func setup(t *testing.T) *Repository {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	if err := schema.Apply(db); err != nil {
		t.Fatal(err)
	}
	repository := &Repository{db: db}
	seedExecutionParents(t, repository)
	return repository
}

func seedExecutionParents(t *testing.T, repository *Repository) {
	t.Helper()
	statements := []string{
		`INSERT INTO runtimes(name) VALUES ('local')`,
		`INSERT INTO environments(id, name) VALUES ('environment', 'test')`,
		`INSERT INTO environment_versions(id, environment_id, version, status, network_model, interference_model, cost_model, storage_model, configuration_hash) VALUES ('env', 'environment', 1, 'published', '{}', '{}', '{}', '{}', 'hash')`,
		`INSERT INTO workflow_definitions(id, external_id, name) VALUES ('workflow', 'workflow', 'test')`,
		`INSERT INTO workflow_versions(id, workflow_id, version, definition_hash) VALUES ('workflow-version', 'workflow', 1, 'hash')`,
		`INSERT INTO activity_types(id, name) VALUES ('type', 'task')`,
		`INSERT INTO activity_definitions(
			id, workflow_version_id, activity_type_id, external_id, name, kind,
			capabilities, command_spec, resource_requirements, policy
		) VALUES ('activity', 'workflow-version', 'type', 'activity', 'activity', 'task', '{}', '{}', '{}', '{}')`,
		`INSERT INTO resources(id, environment_version_id, runtime_id, type, name, provider_id) VALUES ('resource', 'env', 'local', 'host', 'local', 'resource')`,
		`INSERT INTO schedule_plans(id, workflow_version_id, environment_version_id, source, algorithm) VALUES ('plan', 'workflow-version', 'env', 'plugin', 'test')`,
		`INSERT INTO schedule_plan_assignments(id, schedule_plan_id, activity_id, resource_id, order_on_resource) VALUES ('assignment', 'plan', 'activity', 'resource', 1)`,
	}
	for _, statement := range statements {
		if _, err := repository.db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRepositoryOwnsCompleteExecutionAggregate(t *testing.T) {
	repository := setup(t)
	ctx := context.Background()
	run := domain.ExecutionRun{ID: "run", SchedulePlanID: "plan", Mode: domain.ExecutionModeReal, Status: domain.ExecutionRunRunning}
	if err := repository.CreateRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	got, err := repository.FindRun(ctx, "run")
	if err != nil || got == nil || got.Mode != domain.ExecutionModeReal {
		t.Fatalf("run=%+v err=%v", got, err)
	}
	handle := domain.ActivityHandle{ID: "handle", RunID: "run", ActivityID: "activity", ResourceID: "resource", RuntimeID: "local", Status: domain.HandleRunning, Metadata: map[string]any{"pid": 1}}
	if err := repository.Save(ctx, handle); err != nil {
		t.Fatal(err)
	}
	if stored, err := repository.Find(ctx, "handle"); err != nil || stored == nil || stored.Metadata["pid"].(float64) != 1 {
		t.Fatalf("handle=%+v err=%v", stored, err)
	}
	task := domain.TaskExecution{ID: "task", ExecutionRunID: "run", PlanAssignmentID: "assignment", ActivityID: "activity", PlannedResourceID: "resource", Attempt: 1, Status: domain.TaskRunning}
	if err := repository.SaveTask(ctx, task); err != nil {
		t.Fatal(err)
	}
	task.Status, task.FinishedAt = domain.TaskCompleted, 4
	if err := repository.SaveTask(ctx, task); err != nil {
		t.Fatal(err)
	}
	trace := domain.ExecutionTrace{RunID: "run", Executed: domain.ExecutionMetrics{MakespanSeconds: 4, Cost: 2}}
	if err := repository.CompleteRun(ctx, trace); err != nil {
		t.Fatal(err)
	}
	if err := repository.FailRun(ctx, "missing", "failure"); err == nil {
		t.Fatal("missing run must fail")
	}
}

func TestRepositoryReturnsNilForMissingRecords(t *testing.T) {
	repository := setup(t)
	ctx := context.Background()
	if run, err := repository.FindRun(ctx, "missing"); err != nil || run != nil {
		t.Fatal("missing run must be nil")
	}
	if handle, err := repository.Find(ctx, "missing"); err != nil || handle != nil {
		t.Fatal("missing handle must be nil")
	}
}
