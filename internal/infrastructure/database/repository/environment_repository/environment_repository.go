package environment_repository

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/UFFeScience/akoflow/internal/domain"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/schema"
)

type Definition struct {
	Environment domain.Environment               `json:"environment"`
	Version     domain.EnvironmentVersion        `json:"version"`
	Runtimes    []domain.EnvironmentRuntime      `json:"runtimes"`
	Resources   []domain.Resource                `json:"resources"`
	Links       []domain.NetworkLink             `json:"networkLinks"`
	Profiles    []domain.ActivityResourceProfile `json:"activityResourceProfiles,omitempty"`
	Connections []domain.EnvironmentConnection   `json:"connections,omitempty"`
}

type IRepository interface {
	Create(Definition) error
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
	environment := definition.Environment
	if environment.Status == "" {
		environment.Status = domain.EnvironmentDefined
	}
	if _, err = tx.Exec(`INSERT INTO environments (id, name, description, status)
		VALUES (?, ?, ?, ?)`, environment.ID, environment.Name, environment.Description, environment.Status); err != nil {
		return err
	}
	for _, connection := range definition.Connections {
		configuration, marshalErr := json.Marshal(connection.Configuration)
		if marshalErr != nil {
			return marshalErr
		}
		if _, err = tx.Exec(`INSERT INTO environment_connections (
			id, environment_id, name, type, endpoint, username, credential_ref, configuration
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, connection.ID, environment.ID,
			connection.Name, connection.Type, connection.Endpoint, connection.Username,
			connection.CredentialRef, string(configuration)); err != nil {
			return err
		}
	}
	version := definition.Version
	if _, err = tx.Exec(`INSERT INTO environment_versions (
		id, environment_id, version, status, network_model, interference_model,
		cost_model, storage_model, configuration_hash, published_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, version.ID, environment.ID,
		version.Version, version.Status, version.NetworkModel, version.InterferenceModel,
		version.CostModel, version.StorageModel, version.ConfigurationHash,
		version.PublishedAt); err != nil {
		return err
	}
	for _, runtime := range definition.Runtimes {
		configuration, _ := json.Marshal(runtime.Configuration)
		if _, err = tx.Exec(`INSERT INTO environment_runtimes (
			environment_version_id, runtime_id, role, configuration
		) VALUES (?, ?, ?, ?)`, version.ID, runtime.RuntimeID, runtime.Role,
			string(configuration)); err != nil {
			return err
		}
		capabilities, marshalErr := json.Marshal(runtime.Capabilities)
		if marshalErr != nil {
			return marshalErr
		}
		if _, err = tx.Exec(`INSERT INTO environment_runtime_capabilities (
			environment_version_id, runtime_id, capabilities
		) VALUES (?, ?, ?)`, version.ID, runtime.RuntimeID, string(capabilities)); err != nil {
			return err
		}
	}
	for _, resource := range definition.Resources {
		metadata, _ := json.Marshal(resource.Metadata)
		if _, err = tx.Exec(`INSERT INTO resources (
			id, environment_version_id, runtime_id, parent_resource_id, type, name,
			provider_id, tier, region, zone, architecture, cpu_cores, cpu_capacity,
			memory_bytes, storage_bytes, compute_speedup, price_per_second,
			boot_overhead_seconds, container_overhead_seconds, schedulable, metadata
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			resource.ID, version.ID, resource.RuntimeID, resource.ParentResourceID,
			resource.Type, resource.Name, resource.ProviderID, resource.Tier,
			resource.Region, resource.Zone, resource.Architecture, resource.CPUCores,
			resource.CPUCapacity, resource.MemoryBytes, resource.StorageBytes,
			resource.ComputeSpeedup, resource.PricePerSecond,
			resource.BootOverheadSeconds, resource.ContainerOverhead,
			resource.Schedulable, string(metadata)); err != nil {
			return err
		}
	}
	for _, link := range definition.Links {
		metadata, _ := json.Marshal(link.Metadata)
		if _, err = tx.Exec(`INSERT INTO network_links (
			id, environment_version_id, source_resource_id, target_resource_id,
			bandwidth_bits_per_second, latency_seconds, price_per_byte,
			bidirectional, sharing_policy, max_concurrent_transfers, metadata
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, link.ID, version.ID,
			link.SourceResourceID, link.TargetResourceID, link.BandwidthBitsPerSecond,
			link.LatencySeconds, link.PricePerByte, link.Bidirectional,
			link.SharingPolicy, link.MaxConcurrentTransfers, string(metadata)); err != nil {
			return err
		}
	}
	for _, profile := range definition.Profiles {
		metadata, _ := json.Marshal(profile.Metadata)
		if _, err = tx.Exec(`INSERT INTO activity_resource_profiles (
			id, activity_type_id, resource_id, runtime_seconds,
			runtime_stddev_seconds, cpu_utilization, peak_memory_bytes,
			disk_read_bytes, disk_write_bytes, energy_joules, source, sample_size,
			model_version, metadata
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, profile.ID,
			profile.ActivityTypeID, profile.ResourceID, profile.RuntimeSeconds,
			profile.RuntimeStdDevSeconds, profile.CPUUtilization,
			profile.PeakMemoryBytes, profile.DiskReadBytes, profile.DiskWriteBytes,
			profile.EnergyJoules, profile.Source, profile.SampleSize,
			profile.ModelVersion, string(metadata)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *Repository) UpdateStatus(ctx context.Context, id string, status domain.EnvironmentStatus) error {
	db := (&repository.Database{}).Connect()
	defer db.Close()
	result, err := db.ExecContext(ctx, `UPDATE environments SET status=? WHERE id=?`, status, id)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *Repository) UpsertConnection(ctx context.Context, connection domain.EnvironmentConnection) error {
	db := (&repository.Database{}).Connect()
	defer db.Close()
	configuration, err := json.Marshal(connection.Configuration)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, `INSERT INTO environment_connections (
		id, environment_id, name, type, endpoint, username, credential_ref, configuration
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET name=excluded.name, type=excluded.type,
		endpoint=excluded.endpoint, username=excluded.username,
		credential_ref=excluded.credential_ref, configuration=excluded.configuration`,
		connection.ID, connection.EnvironmentID, connection.Name, connection.Type,
		connection.Endpoint, connection.Username, connection.CredentialRef, string(configuration))
	return err
}

func (r *Repository) ListConnections(ctx context.Context, environmentID string) ([]domain.EnvironmentConnection, error) {
	db := (&repository.Database{}).Connect()
	defer db.Close()
	rows, err := db.QueryContext(ctx, `SELECT id, environment_id, name, type,
		endpoint, username, credential_ref, configuration, created_at
		FROM environment_connections WHERE environment_id=? ORDER BY name`, environmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	connections := make([]domain.EnvironmentConnection, 0)
	for rows.Next() {
		var connection domain.EnvironmentConnection
		var connectionType, configuration string
		if err := rows.Scan(&connection.ID, &connection.EnvironmentID, &connection.Name,
			&connectionType, &connection.Endpoint, &connection.Username,
			&connection.CredentialRef, &configuration, &connection.CreatedAt); err != nil {
			return nil, err
		}
		connection.Type = domain.ConnectionType(connectionType)
		if err := json.Unmarshal([]byte(configuration), &connection.Configuration); err != nil {
			return nil, err
		}
		connections = append(connections, connection)
	}
	return connections, rows.Err()
}
