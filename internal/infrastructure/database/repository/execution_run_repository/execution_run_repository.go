package execution_run_repository

import (
	"encoding/json"

	"github.com/UFFeScience/akoflow/internal/domain"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/schema"
)

type IRepository interface {
	Create(run domain.ExecutionRun) error
	Complete(trace domain.ExecutionTrace) error
}

type Repository struct{}

func New() IRepository {
	db := (&repository.Database{}).Connect()
	defer db.Close()
	if err := schema.Apply(db); err != nil {
		panic(err)
	}
	return &Repository{}
}

func (r *Repository) Create(run domain.ExecutionRun) error {
	db := (&repository.Database{}).Connect()
	defer db.Close()
	_, err := db.Exec(`INSERT INTO execution_runs (
		id, schedule_plan_id, mode, seed, status, environment_snapshot_id
	) VALUES (?, ?, ?, ?, ?, ?)`, run.ID, run.SchedulePlanID, run.Mode, run.Seed,
		run.Status, run.EnvironmentSnapshotID)
	return err
}

func (r *Repository) Complete(trace domain.ExecutionTrace) error {
	db := (&repository.Database{}).Connect()
	defer db.Close()
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, task := range trace.Tasks {
		metadata, _ := json.Marshal(map[string]any{"mode": trace.Mode})
		if _, err = tx.Exec(`INSERT INTO task_executions (
			id, execution_run_id, plan_assignment_id, activity_id,
			planned_resource_id, allocated_resource_id, attempt, status, ready_at,
			data_ready_at, queued_at, started_at, finished_at, runtime_seconds,
			queue_seconds, transfer_seconds, interference_seconds, overhead_seconds,
			cost, failure_reason, metadata
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			task.ID, trace.RunID, task.PlanAssignmentID, task.ActivityID,
			task.PlannedResourceID, task.AllocatedResourceID, task.Attempt, task.Status,
			task.ReadyAt, task.DataReadyAt, task.QueuedAt, task.StartedAt, task.FinishedAt,
			task.RuntimeSeconds, task.QueueSeconds, task.TransferSeconds,
			task.InterferenceSeconds, task.OverheadSeconds, task.Cost, task.FailureReason,
			string(metadata)); err != nil {
			return err
		}
	}
	for _, transfer := range trace.Transfers {
		if _, err = tx.Exec(`INSERT INTO data_transfers (
			id, execution_run_id, producer_activity_id, consumer_activity_id,
			source_resource_id, target_resource_id, bytes, status, started_at,
			finished_at, duration_seconds, cost
		) VALUES (?, ?, ?, ?, ?, ?, ?, 'completed', ?, ?, ?, ?)`, transfer.ID,
			trace.RunID, transfer.ProducerActivityID, transfer.ConsumerActivityID,
			transfer.SourceResourceID, transfer.TargetResourceID, transfer.Bytes,
			transfer.StartedAt, transfer.FinishedAt, transfer.DurationSeconds,
			transfer.Cost); err != nil {
			return err
		}
	}
	_, err = tx.Exec(`UPDATE execution_runs SET status='completed',
		finished_at=CURRENT_TIMESTAMP, makespan_seconds=?, cost=? WHERE id=?`,
		trace.Executed.MakespanSeconds, trace.Executed.Cost, trace.RunID)
	if err != nil {
		return err
	}
	return tx.Commit()
}
