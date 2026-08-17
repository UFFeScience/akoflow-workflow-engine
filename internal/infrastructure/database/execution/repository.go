package execution

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/UFFeScience/akoflow/internal/application/ports"
	"github.com/UFFeScience/akoflow/internal/domain"
	domainevents "github.com/UFFeScience/akoflow/internal/domain/events"
	dbevents "github.com/UFFeScience/akoflow/internal/infrastructure/database/events"
)

type Repository struct{ db *sql.DB }

var _ ports.ExecutionStore = (*Repository)(nil)

func New(db *sql.DB) *Repository { return &Repository{db: db} }

func (r *Repository) CreateRun(ctx context.Context, run domain.ExecutionRun) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `INSERT INTO execution_runs (
		id, schedule_plan_id, mode, seed, status, environment_snapshot_id, started_at
	) VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	ON CONFLICT(id) DO NOTHING`, run.ID, run.SchedulePlanID, run.Mode, run.Seed,
		run.Status, run.EnvironmentSnapshotID)
	if err != nil {
		return err
	}
	if changed, _ := result.RowsAffected(); changed == 1 {
		if err := dbevents.Append(ctx, tx, domainevents.Event{
			Type: domainevents.ExecutionStarted, AggregateType: "execution_run", AggregateID: run.ID,
			Payload: map[string]any{"mode": run.Mode, "schedulePlanId": run.SchedulePlanID},
		}); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *Repository) FindRun(ctx context.Context, id string) (*domain.ExecutionRun, error) {
	var run domain.ExecutionRun
	err := r.db.QueryRowContext(ctx, `SELECT id, schedule_plan_id, mode, seed,
		status, environment_snapshot_id FROM execution_runs WHERE id=?`, id).
		Scan(&run.ID, &run.SchedulePlanID, &run.Mode, &run.Seed, &run.Status,
			&run.EnvironmentSnapshotID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &run, nil
}

func (r *Repository) ListTasks(ctx context.Context, runID string) ([]domain.TaskExecution, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, execution_run_id, plan_assignment_id,
		activity_id, planned_resource_id, COALESCE(allocated_resource_id, ''), attempt,
		status, ready_at, data_ready_at, queued_at, started_at, finished_at,
		runtime_seconds, queue_seconds, transfer_seconds, interference_seconds,
		overhead_seconds, cost, failure_reason FROM task_executions
		WHERE execution_run_id=? ORDER BY started_at, activity_id`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tasks []domain.TaskExecution
	for rows.Next() {
		var task domain.TaskExecution
		if err := rows.Scan(&task.ID, &task.ExecutionRunID, &task.PlanAssignmentID,
			&task.ActivityID, &task.PlannedResourceID, &task.AllocatedResourceID,
			&task.Attempt, &task.Status, &task.ReadyAt, &task.DataReadyAt,
			&task.QueuedAt, &task.StartedAt, &task.FinishedAt, &task.RuntimeSeconds,
			&task.QueueSeconds, &task.TransferSeconds, &task.InterferenceSeconds,
			&task.OverheadSeconds, &task.Cost, &task.FailureReason); err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, rows.Err()
}

func (r *Repository) ListEvents(ctx context.Context, runID string) ([]domainevents.Event, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT payload FROM domain_events
		WHERE (aggregate_type='execution_run' AND aggregate_id=?)
		OR (aggregate_type='activity_execution' AND json_extract(payload, '$.payload.executionRunId')=?)
		ORDER BY occurred_at, rowid`, runID, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var events []domainevents.Event
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		var event domainevents.Event
		if err := json.Unmarshal(payload, &event); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func (r *Repository) Save(ctx context.Context, handle domain.ActivityHandle) error {
	endpoints, err := json.Marshal(handle.Endpoints)
	if err != nil {
		return fmt.Errorf("marshal handle endpoints: %w", err)
	}
	metadata, err := json.Marshal(handle.Metadata)
	if err != nil {
		return fmt.Errorf("marshal handle metadata: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO activity_handles (
		id, execution_run_id, activity_id, resource_id, runtime_id, external_id,
		status, endpoints, started_at, finished_at, exit_code, failure, metadata
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET external_id=excluded.external_id,
		status=excluded.status, endpoints=excluded.endpoints,
		started_at=excluded.started_at, finished_at=excluded.finished_at,
		exit_code=excluded.exit_code, failure=excluded.failure,
		metadata=excluded.metadata, updated_at=CURRENT_TIMESTAMP`,
		handle.ID, handle.RunID, handle.ActivityID, handle.ResourceID, handle.RuntimeID,
		handle.ExternalID, handle.Status, string(endpoints), handle.StartedAt,
		handle.FinishedAt, handle.ExitCode, handle.Failure, string(metadata))
	return err
}

func (r *Repository) Find(ctx context.Context, id string) (*domain.ActivityHandle, error) {
	var handle domain.ActivityHandle
	var endpoints, metadata string
	err := r.db.QueryRowContext(ctx, `SELECT id, execution_run_id, activity_id,
		resource_id, runtime_id, external_id, status, endpoints, started_at,
		finished_at, exit_code, failure, metadata FROM activity_handles WHERE id=?`, id).
		Scan(&handle.ID, &handle.RunID, &handle.ActivityID, &handle.ResourceID,
			&handle.RuntimeID, &handle.ExternalID, &handle.Status, &endpoints,
			&handle.StartedAt, &handle.FinishedAt, &handle.ExitCode, &handle.Failure,
			&metadata)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(endpoints), &handle.Endpoints); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(metadata), &handle.Metadata); err != nil {
		return nil, err
	}
	return &handle, nil
}

func (r *Repository) SaveTask(ctx context.Context, task domain.TaskExecution) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	previous, err := taskStatus(ctx, tx, task.ExecutionRunID, task.ActivityID, task.Attempt)
	if err != nil {
		return err
	}
	if err := saveTask(ctx, tx, task); err != nil {
		return err
	}
	if previous != string(task.Status) {
		if err := appendTaskEvent(ctx, tx, task); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *Repository) CompleteRun(ctx context.Context, trace domain.ExecutionTrace) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, task := range trace.Tasks {
		previous, err := taskStatus(ctx, tx, task.ExecutionRunID, task.ActivityID, task.Attempt)
		if err != nil {
			return err
		}
		if err := saveTask(ctx, tx, task); err != nil {
			return err
		}
		if previous != string(task.Status) {
			if err := appendTaskEvent(ctx, tx, task); err != nil {
				return err
			}
		}
	}
	for _, transfer := range trace.Transfers {
		if _, err := tx.ExecContext(ctx, `INSERT INTO data_transfers (
			id, execution_run_id, producer_activity_id, consumer_activity_id,
			source_resource_id, target_resource_id, bytes, status, started_at,
			finished_at, duration_seconds, cost
		) VALUES (?, ?, ?, ?, ?, ?, ?, 'completed', ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET status='completed', started_at=excluded.started_at,
			finished_at=excluded.finished_at, duration_seconds=excluded.duration_seconds,
			cost=excluded.cost`, transfer.ID,
			trace.RunID, transfer.ProducerActivityID, transfer.ConsumerActivityID,
			transfer.SourceResourceID, transfer.TargetResourceID, transfer.Bytes,
			transfer.StartedAt, transfer.FinishedAt, transfer.DurationSeconds,
			transfer.Cost); err != nil {
			return err
		}
	}
	result, err := tx.ExecContext(ctx, `UPDATE execution_runs SET status='completed',
		finished_at=CURRENT_TIMESTAMP, makespan_seconds=?, cost=?, failure_reason=''
		WHERE id=?`, trace.Executed.MakespanSeconds, trace.Executed.Cost, trace.RunID)
	if err != nil {
		return err
	}
	if changed, _ := result.RowsAffected(); changed != 1 {
		return fmt.Errorf("execution run %q not found", trace.RunID)
	}
	if err := dbevents.Append(ctx, tx, domainevents.Event{
		Type: domainevents.ExecutionCompleted, AggregateType: "execution_run", AggregateID: trace.RunID,
		Payload: map[string]any{"makespanSeconds": trace.Executed.MakespanSeconds, "cost": trace.Executed.Cost},
	}); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *Repository) FailRun(ctx context.Context, id, reason string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `UPDATE execution_runs SET status='failed',
		finished_at=CURRENT_TIMESTAMP, failure_reason=? WHERE id=?`, reason, id)
	if err != nil {
		return err
	}
	if changed, _ := result.RowsAffected(); changed != 1 {
		return fmt.Errorf("execution run %q not found", id)
	}
	if err := dbevents.Append(ctx, tx, domainevents.Event{
		Type: domainevents.ExecutionFailed, AggregateType: "execution_run", AggregateID: id,
		Payload: map[string]any{"reason": reason},
	}); err != nil {
		return err
	}
	return tx.Commit()
}

func taskStatus(ctx context.Context, tx *sql.Tx, runID, activityID string, attempt int) (string, error) {
	var status string
	err := tx.QueryRowContext(ctx, `SELECT status FROM task_executions
		WHERE execution_run_id=? AND activity_id=? AND attempt=?`, runID, activityID, attempt).Scan(&status)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return status, err
}

func appendTaskEvent(ctx context.Context, tx *sql.Tx, task domain.TaskExecution) error {
	eventType := ""
	switch task.Status {
	case domain.TaskRunning:
		eventType = domainevents.ActivityStarted
	case domain.TaskCompleted:
		eventType = domainevents.ActivityCompleted
	case domain.TaskFailed, domain.TaskCancelled:
		eventType = domainevents.ActivityFailed
	default:
		return nil
	}
	lifecycleID := fmt.Sprintf("%s:%s:%d:%s", task.ExecutionRunID, task.ActivityID, task.Attempt, task.Status)
	if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO activity_lifecycle_events (
		id, task_execution_id, phase, status, started_at, finished_at,
		duration_seconds, source, error, metadata
	) VALUES (?, ?, 'execution', ?, ?, ?, ?, 'measured', ?, '{}')`, lifecycleID,
		task.ID, task.Status, task.StartedAt, task.FinishedAt, task.RuntimeSeconds,
		task.FailureReason); err != nil {
		return err
	}
	return dbevents.Append(ctx, tx, domainevents.Event{
		Type: eventType, AggregateType: "activity_execution", AggregateID: task.ID,
		OccurredAt: time.Now().UTC(), Payload: map[string]any{
			"executionRunId": task.ExecutionRunID, "activityId": task.ActivityID,
			"status": task.Status, "runtimeSeconds": task.RuntimeSeconds,
			"failureReason": task.FailureReason,
		},
	})
}

func saveTask(ctx context.Context, tx *sql.Tx, task domain.TaskExecution) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO task_executions (
		id, execution_run_id, plan_assignment_id, activity_id, planned_resource_id,
		allocated_resource_id, attempt, status, ready_at, data_ready_at, queued_at,
		started_at, finished_at, runtime_seconds, queue_seconds, transfer_seconds,
		interference_seconds, overhead_seconds, cost, failure_reason
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(execution_run_id, activity_id, attempt) DO UPDATE SET
		allocated_resource_id=excluded.allocated_resource_id, status=excluded.status,
		started_at=excluded.started_at, finished_at=excluded.finished_at,
		runtime_seconds=excluded.runtime_seconds, queue_seconds=excluded.queue_seconds,
		transfer_seconds=excluded.transfer_seconds,
		interference_seconds=excluded.interference_seconds,
		overhead_seconds=excluded.overhead_seconds, cost=excluded.cost,
		failure_reason=excluded.failure_reason`,
		task.ID, task.ExecutionRunID, task.PlanAssignmentID, task.ActivityID,
		task.PlannedResourceID, nullableString(task.AllocatedResourceID), task.Attempt,
		task.Status, task.ReadyAt, task.DataReadyAt, task.QueuedAt, task.StartedAt,
		task.FinishedAt, task.RuntimeSeconds, task.QueueSeconds, task.TransferSeconds,
		task.InterferenceSeconds, task.OverheadSeconds, task.Cost, task.FailureReason)
	return err
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}
