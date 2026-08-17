package planning

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/UFFeScience/akoflow/internal/application/ports"
	"github.com/UFFeScience/akoflow/internal/domain"
)

type Repository struct{ db *sql.DB }

var _ ports.PlanStore = (*Repository)(nil)

func New(db *sql.DB) *Repository { return &Repository{db: db} }

func (r *Repository) Save(ctx context.Context, plan domain.SchedulePlan) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`INSERT INTO schedule_plans (
		id, workflow_version_id, environment_version_id, network_topology_id, source, algorithm,
		algorithm_version, objective, status, deadline_seconds, budget,
		predicted_makespan_seconds, predicted_cost, predicted_feasible
	) VALUES (?, ?, ?, NULLIF(?, ''), ?, ?, ?, ?, 'valid', ?, ?, ?, ?, ?)`, plan.ID,
		plan.WorkflowVersionID, plan.EnvironmentVersionID, plan.NetworkTopologyID, plan.Source, plan.Algorithm,
		plan.AlgorithmVersion, plan.Objective, plan.DeadlineSeconds, plan.Budget,
		plan.Predicted.MakespanSeconds, plan.Predicted.Cost, plan.Predicted.Feasible); err != nil {
		return err
	}
	for _, assignment := range plan.Assignments {
		metadata, err := json.Marshal(assignment.Metadata)
		if err != nil {
			return err
		}
		if _, err = tx.Exec(`INSERT INTO schedule_plan_assignments (
			id, schedule_plan_id, activity_id, resource_id, core_id, slot_id,
			order_on_resource, priority, predicted_ready_at, predicted_start_at,
			predicted_finish_at, predicted_runtime_seconds, predicted_transfer_seconds,
			predicted_cost, metadata
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, assignment.ID,
			plan.ID, assignment.ActivityID, assignment.ResourceID, assignment.CoreID,
			assignment.SlotID, assignment.OrderOnResource, assignment.Priority,
			assignment.PredictedReadyAt, assignment.PredictedStartAt,
			assignment.PredictedFinishAt, assignment.PredictedRuntimeSeconds,
			assignment.PredictedTransferSeconds, assignment.PredictedCost, string(metadata)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *Repository) Find(ctx context.Context, id string) (*domain.SchedulePlan, error) {
	var plan domain.SchedulePlan
	var source string
	err := r.db.QueryRowContext(ctx, `SELECT id, workflow_version_id, environment_version_id,
		COALESCE(network_topology_id, ''), source, algorithm, algorithm_version, objective, deadline_seconds, budget,
		predicted_makespan_seconds, predicted_cost, predicted_feasible
		FROM schedule_plans WHERE id = ?`, id).Scan(&plan.ID, &plan.WorkflowVersionID,
		&plan.EnvironmentVersionID, &plan.NetworkTopologyID, &source, &plan.Algorithm, &plan.AlgorithmVersion,
		&plan.Objective, &plan.DeadlineSeconds, &plan.Budget,
		&plan.Predicted.MakespanSeconds, &plan.Predicted.Cost, &plan.Predicted.Feasible)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	plan.Source = domain.PlanningSource(source)
	rows, err := r.db.QueryContext(ctx, `SELECT id, schedule_plan_id, activity_id, resource_id,
		core_id, slot_id, order_on_resource, priority, predicted_ready_at,
		predicted_start_at, predicted_finish_at, predicted_runtime_seconds,
		predicted_transfer_seconds, predicted_cost, metadata
		FROM schedule_plan_assignments WHERE schedule_plan_id = ?
		ORDER BY resource_id, core_id, slot_id, order_on_resource`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var assignment domain.PlanAssignment
		var metadata string
		if err := rows.Scan(&assignment.ID, &assignment.PlanID, &assignment.ActivityID,
			&assignment.ResourceID, &assignment.CoreID, &assignment.SlotID,
			&assignment.OrderOnResource, &assignment.Priority, &assignment.PredictedReadyAt,
			&assignment.PredictedStartAt, &assignment.PredictedFinishAt,
			&assignment.PredictedRuntimeSeconds, &assignment.PredictedTransferSeconds,
			&assignment.PredictedCost, &metadata); err != nil {
			return nil, err
		}
		if metadata != "" {
			if err := json.Unmarshal([]byte(metadata), &assignment.Metadata); err != nil {
				return nil, err
			}
		}
		plan.Assignments = append(plan.Assignments, assignment)
	}
	return &plan, rows.Err()
}
