package workflow_definition_repository

import (
	"database/sql"
	"encoding/json"

	"github.com/UFFeScience/akoflow/internal/domain"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/schema"
)

type Definition struct {
	ID         string                 `json:"id"`
	ExternalID string                 `json:"externalId"`
	Name       string                 `json:"name"`
	Namespace  string                 `json:"namespace"`
	Version    domain.WorkflowVersion `json:"version"`
	Types      []domain.ActivityType  `json:"activityTypes"`
}

type IRepository interface {
	Create(definition Definition) error
	FindVersion(id string) (*domain.WorkflowVersion, error)
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

func (r *Repository) Create(definition Definition) error {
	db := (&repository.Database{}).Connect()
	defer db.Close()
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`INSERT INTO normalized_workflows (id, external_id, name, namespace)
		VALUES (?, ?, ?, ?)`, definition.ID, definition.ExternalID, definition.Name,
		definition.Namespace); err != nil {
		return err
	}
	version := definition.Version
	if _, err = tx.Exec(`INSERT INTO workflow_versions (
		id, workflow_id, version, definition_hash, status
	) VALUES (?, ?, ?, ?, 'published')`, version.ID, definition.ID, version.Version,
		version.DefinitionHash); err != nil {
		return err
	}
	for _, activityType := range definition.Types {
		metadata, _ := json.Marshal(activityType.Metadata)
		if _, err = tx.Exec(`INSERT OR IGNORE INTO activity_types (
			id, name, application, default_image, cpu_intensity, memory_intensity,
			io_intensity, network_intensity, metadata
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, activityType.ID, activityType.Name,
			activityType.Application, activityType.DefaultImage, activityType.CPUIntensity,
			activityType.MemoryIntensity, activityType.IOIntensity,
			activityType.NetworkIntensity, string(metadata)); err != nil {
			return err
		}
	}
	for _, activity := range version.Activities {
		metadata, _ := json.Marshal(activity.Metadata)
		if _, err = tx.Exec(`INSERT INTO normalized_activities (
			id, workflow_version_id, activity_type_id, external_id, name, command,
			image, priority, cpu_required, memory_required_bytes,
			storage_required_bytes, metadata
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, activity.ID, version.ID,
			activity.ActivityTypeID, activity.ExternalID, activity.Name, activity.Command,
			activity.Image, activity.Priority, activity.CPURequired,
			activity.MemoryRequiredBytes, activity.StorageRequiredBytes, string(metadata)); err != nil {
			return err
		}
	}
	for _, dependency := range version.Dependencies {
		if _, err = tx.Exec(`INSERT INTO normalized_activity_dependencies (
			activity_id, depends_on_activity_id, dependency_type
		) VALUES (?, ?, ?)`, dependency.ActivityID, dependency.DependsOnActivityID,
			dependency.Type); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *Repository) FindVersion(id string) (*domain.WorkflowVersion, error) {
	db := (&repository.Database{}).Connect()
	defer db.Close()
	var workflow domain.WorkflowVersion
	err := db.QueryRow(`SELECT id, workflow_id, version, definition_hash
		FROM workflow_versions WHERE id=?`, id).Scan(&workflow.ID, &workflow.WorkflowID,
		&workflow.Version, &workflow.DefinitionHash)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	rows, err := db.Query(`SELECT id, workflow_version_id, activity_type_id,
		external_id, name, command, image, priority, cpu_required,
		memory_required_bytes, storage_required_bytes, metadata
		FROM normalized_activities WHERE workflow_version_id=?`, id)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var activity domain.Activity
		var metadata string
		if err := rows.Scan(&activity.ID, &activity.WorkflowVersionID,
			&activity.ActivityTypeID, &activity.ExternalID, &activity.Name,
			&activity.Command, &activity.Image, &activity.Priority,
			&activity.CPURequired, &activity.MemoryRequiredBytes,
			&activity.StorageRequiredBytes, &metadata); err != nil {
			rows.Close()
			return nil, err
		}
		_ = json.Unmarshal([]byte(metadata), &activity.Metadata)
		workflow.Activities = append(workflow.Activities, activity)
	}
	rows.Close()
	deps, err := db.Query(`SELECT d.activity_id, d.depends_on_activity_id,
		d.dependency_type FROM normalized_activity_dependencies d
		JOIN normalized_activities a ON a.id=d.activity_id
		WHERE a.workflow_version_id=?`, id)
	if err != nil {
		return nil, err
	}
	defer deps.Close()
	for deps.Next() {
		var dependency domain.ActivityDependency
		if err := deps.Scan(&dependency.ActivityID, &dependency.DependsOnActivityID,
			&dependency.Type); err != nil {
			return nil, err
		}
		workflow.Dependencies = append(workflow.Dependencies, dependency)
	}
	return &workflow, deps.Err()
}
