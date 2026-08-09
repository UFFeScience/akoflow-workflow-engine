package schedule_plan_repository

import (
	"database/sql"
	"encoding/json"

	"github.com/UFFeScience/akoflow/internal/domain"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/schema"
)

type IRepository interface {
	Save(plan domain.SchedulePlan) error
	Find(id string) (*domain.SchedulePlan, error)
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

func (r *Repository) Save(plan domain.SchedulePlan) error {
	db := (&repository.Database{}).Connect()
	defer db.Close()
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`INSERT INTO schedule_plans (
		id, workflow_version_id, environment_version_id, source, algorithm,
		algorithm_version, objective, status, deadline_seconds, budget,
		predicted_makespan_seconds, predicted_cost, predicted_feasible
	) VALUES (?, ?, ?, ?, ?, ?, ?, 'valid', ?, ?, ?, ?, ?)`, plan.ID,
		plan.WorkflowVersionID, plan.EnvironmentVersionID, plan.Source, plan.Algorithm,
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

func (r *Repository) Find(id string) (*domain.SchedulePlan, error) {
	db := (&repository.Database{}).Connect()
	defer db.Close()
	var plan domain.SchedulePlan
	var source string
	err := db.QueryRow(`SELECT id, workflow_version_id, environment_version_id,
		source, algorithm, algorithm_version, objective, deadline_seconds, budget,
		predicted_makespan_seconds, predicted_cost, predicted_feasible
		FROM schedule_plans WHERE id = ?`, id).Scan(&plan.ID, &plan.WorkflowVersionID,
		&plan.EnvironmentVersionID, &source, &plan.Algorithm, &plan.AlgorithmVersion,
		&plan.Objective, &plan.DeadlineSeconds, &plan.Budget,
		&plan.Predicted.MakespanSeconds, &plan.Predicted.Cost, &plan.Predicted.Feasible)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	plan.Source = domain.PlanningSource(source)
	rows, err := db.Query(`SELECT id, schedule_plan_id, activity_id, resource_id,
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
