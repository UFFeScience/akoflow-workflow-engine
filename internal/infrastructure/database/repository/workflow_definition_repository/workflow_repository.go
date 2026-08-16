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
	if _, err = tx.Exec(`INSERT INTO workflow_definitions (id, external_id, name, namespace)
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
		if err = activity.Validate(); err != nil {
			return err
		}
		metadata, _ := json.Marshal(activity.Metadata)
		capabilities, _ := json.Marshal(activity.Capabilities)
		command, _ := json.Marshal(activity.Command)
		resources, _ := json.Marshal(activity.Resources)
		service, _ := json.Marshal(activity.Service)
		simulation, _ := json.Marshal(activity.Simulation)
		policy, _ := json.Marshal(activity.Policy)
		if _, err = tx.Exec(`INSERT INTO activity_definitions (
			id, workflow_version_id, activity_type_id, external_id, name, kind,
			capabilities, command_spec, resource_requirements, service_spec,
			simulation_spec, policy, priority, metadata
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, activity.ID, version.ID,
			activity.ActivityTypeID, activity.ExternalID, activity.Name, activity.Kind,
			string(capabilities), string(command), string(resources), nullableJSON(activity.Service, service),
			nullableJSON(activity.Simulation, simulation), string(policy), activity.Priority,
			string(metadata)); err != nil {
			return err
		}
	}
	for _, dependency := range version.Dependencies {
		if _, err = tx.Exec(`INSERT INTO activity_dependencies (
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
		external_id, name, kind, capabilities, command_spec, resource_requirements,
		service_spec, simulation_spec, policy, priority, metadata
		FROM activity_definitions WHERE workflow_version_id=?`, id)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var activity domain.Activity
		var capabilities, command, resources, policy, metadata string
		var service, simulation sql.NullString
		if err := rows.Scan(&activity.ID, &activity.WorkflowVersionID,
			&activity.ActivityTypeID, &activity.ExternalID, &activity.Name,
			&activity.Kind, &capabilities, &command, &resources, &service,
			&simulation, &policy, &activity.Priority, &metadata); err != nil {
			rows.Close()
			return nil, err
		}
		_ = json.Unmarshal([]byte(metadata), &activity.Metadata)
		_ = json.Unmarshal([]byte(capabilities), &activity.Capabilities)
		_ = json.Unmarshal([]byte(command), &activity.Command)
		_ = json.Unmarshal([]byte(resources), &activity.Resources)
		_ = json.Unmarshal([]byte(policy), &activity.Policy)
		if service.Valid {
			activity.Service = &domain.ActivityService{}
			_ = json.Unmarshal([]byte(service.String), activity.Service)
		}
		if simulation.Valid {
			activity.Simulation = &domain.ActivitySimulation{}
			_ = json.Unmarshal([]byte(simulation.String), activity.Simulation)
		}
		workflow.Activities = append(workflow.Activities, activity)
	}
	rows.Close()
	deps, err := db.Query(`SELECT d.activity_id, d.depends_on_activity_id,
		d.dependency_type FROM activity_dependencies d
		JOIN activity_definitions a ON a.id=d.activity_id
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

func nullableJSON(value any, encoded []byte) any {
	if value == nil {
		return nil
	}
	return string(encoded)
}
