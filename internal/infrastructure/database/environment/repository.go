package environment

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/UFFeScience/akoflow/internal/application/ports"
	"github.com/UFFeScience/akoflow/internal/domain"
)

type Definition = domain.EnvironmentDefinition
type Repository struct{ db *sql.DB }

var _ ports.EnvironmentCatalog = (*Repository)(nil)
var _ ports.StorageCatalog = (*Repository)(nil)

func New(db *sql.DB) *Repository { return &Repository{db: db} }

func (r *Repository) Create(ctx context.Context, definition Definition) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := insertEnvironment(tx, definition); err != nil {
		return err
	}
	if err := insertConnections(tx, definition); err != nil {
		return err
	}
	if err := insertEnvironmentVersion(tx, definition); err != nil {
		return err
	}
	if err := insertRuntimes(tx, definition); err != nil {
		return err
	}
	if err := insertResources(tx, definition); err != nil {
		return err
	}
	if err := insertStorages(tx, definition); err != nil {
		return err
	}
	if err := insertProfiles(tx, definition); err != nil {
		return err
	}
	return tx.Commit()
}

func insertStorages(tx *sql.Tx, definition Definition) error {
	for _, storage := range definition.Storages {
		configuration, err := json.Marshal(storage.Configuration)
		if err != nil {
			return err
		}
		metadata, err := json.Marshal(storage.Metadata)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(`INSERT INTO storage_resources (
			id, environment_version_id, name, type, endpoint, capacity_bytes,
			shared, read_only, configuration, credential_reference, metadata
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, storage.ID, definition.Version.ID,
			storage.Name, storage.Type, storage.Endpoint, storage.CapacityBytes,
			storage.Shared, storage.ReadOnly, string(configuration),
			storage.CredentialReference, string(metadata)); err != nil {
			return err
		}
		for _, binding := range storage.RuntimeBindings {
			bindingConfiguration, err := json.Marshal(binding.Configuration)
			if err != nil {
				return err
			}
			containerPath := binding.ContainerPath
			if containerPath == "" {
				containerPath = "/akoflow/data"
			}
			if _, err := tx.Exec(`INSERT INTO storage_runtime_bindings (
				storage_resource_id, environment_version_id, runtime_id, is_default,
				host_path, container_path, read_only, configuration
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, storage.ID, definition.Version.ID,
				binding.RuntimeID, binding.Default, binding.HostPath, containerPath,
				binding.ReadOnly, string(bindingConfiguration)); err != nil {
				return err
			}
		}
	}
	return nil
}

func insertEnvironment(tx *sql.Tx, definition Definition) error {
	environment := definition.Environment
	if environment.Status == "" {
		environment.Status = domain.EnvironmentDefined
	}
	_, err := tx.Exec(`INSERT INTO environments (id, name, description, status)
		VALUES (?, ?, ?, ?)`, environment.ID, environment.Name, environment.Description, environment.Status)
	return err
}

func insertConnections(tx *sql.Tx, definition Definition) error {
	environmentID := definition.Environment.ID
	for _, connection := range definition.Connections {
		configuration, err := json.Marshal(connection.Configuration)
		if err != nil {
			return err
		}
		_, err = tx.Exec(`INSERT INTO environment_connections (
			id, environment_id, name, type, endpoint, username, credential_ref, configuration
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, connection.ID, environmentID,
			connection.Name, connection.Type, connection.Endpoint, connection.Username,
			connection.CredentialRef, string(configuration))
		if err != nil {
			return err
		}
	}
	return nil
}

func insertEnvironmentVersion(tx *sql.Tx, definition Definition) error {
	version := definition.Version
	_, err := tx.Exec(`INSERT INTO environment_versions (
		id, environment_id, version, status, network_model, interference_model,
		cost_model, configuration_hash, published_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, version.ID, definition.Environment.ID,
		version.Version, version.Status, version.NetworkModel, version.InterferenceModel,
		version.CostModel, version.ConfigurationHash,
		version.PublishedAt)
	return err
}

func insertRuntimes(tx *sql.Tx, definition Definition) error {
	versionID := definition.Version.ID
	for _, runtime := range definition.Runtimes {
		if _, err := tx.Exec(`INSERT OR IGNORE INTO runtimes (name) VALUES (?)`, runtime.RuntimeID); err != nil {
			return err
		}
		configuration, err := json.Marshal(runtime.Configuration)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(`INSERT INTO environment_runtimes (
			environment_version_id, runtime_id, role, configuration
		) VALUES (?, ?, ?, ?)`, versionID, runtime.RuntimeID, runtime.Role,
			string(configuration)); err != nil {
			return err
		}
		capabilities, err := json.Marshal(runtime.Capabilities)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(`INSERT INTO environment_runtime_capabilities (
			environment_version_id, runtime_id, capabilities
		) VALUES (?, ?, ?)`, versionID, runtime.RuntimeID, string(capabilities)); err != nil {
			return err
		}
	}
	return nil
}

func insertResources(tx *sql.Tx, definition Definition) error {
	for _, resource := range definition.Resources {
		metadata, err := json.Marshal(resource.Metadata)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(`INSERT INTO resources (
			id, environment_version_id, runtime_id, execution_target, parent_resource_id, type, name,
			provider_id, tier, region, zone, architecture, cpu_cores, cpu_capacity,
			memory_bytes, storage_bytes, compute_speedup, price_per_second,
			boot_overhead_seconds, container_overhead_seconds, schedulable, metadata
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			resource.ID, definition.Version.ID, resource.RuntimeID, normalizedExecutionTarget(resource.ExecutionTarget), resource.ParentResourceID,
			resource.Type, resource.Name, resource.ProviderID, resource.Tier,
			resource.Region, resource.Zone, resource.Architecture, resource.CPUCores,
			resource.CPUCapacity, resource.MemoryBytes, resource.StorageBytes,
			resource.ComputeSpeedup, resource.PricePerSecond,
			resource.BootOverheadSeconds, resource.ContainerOverhead,
			resource.Schedulable, string(metadata)); err != nil {
			return err
		}
	}
	return nil
}

func normalizedExecutionTarget(target domain.ResourceExecutionTarget) domain.ResourceExecutionTarget {
	if target == "" {
		return domain.ExecutionTargetBatch
	}
	return target
}

func insertProfiles(tx *sql.Tx, definition Definition) error {
	for _, profile := range definition.Profiles {
		metadata, err := json.Marshal(profile.Metadata)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(`INSERT INTO activity_resource_profiles (
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
	return nil
}

func (r *Repository) UpdateStatus(ctx context.Context, id string, status domain.EnvironmentStatus) error {
	result, err := r.db.ExecContext(ctx, `UPDATE environments SET status=? WHERE id=?`, status, id)
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
	configuration, err := json.Marshal(connection.Configuration)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO environment_connections (
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
	rows, err := r.db.QueryContext(ctx, `SELECT id, environment_id, name, type,
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

func (r *Repository) ListRuntimeStorages(ctx context.Context, versionID, runtimeID string) ([]domain.StorageResource, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT s.id, s.environment_version_id, s.name,
		s.type, s.endpoint, s.capacity_bytes, s.shared, s.read_only, s.configuration,
		s.credential_reference, s.metadata, b.is_default, b.host_path,
		b.container_path, b.read_only, b.configuration
		FROM storage_resources s
		JOIN storage_runtime_bindings b ON b.storage_resource_id=s.id
		WHERE b.environment_version_id=? AND b.runtime_id=?
		ORDER BY b.is_default DESC, s.name`, versionID, runtimeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	storages := make([]domain.StorageResource, 0)
	for rows.Next() {
		var item domain.StorageResource
		var storageType, storageConfiguration, storageMetadata, bindingConfiguration string
		var binding domain.StorageRuntimeBinding
		if err := rows.Scan(&item.ID, &item.EnvironmentVersionID, &item.Name,
			&storageType, &item.Endpoint, &item.CapacityBytes, &item.Shared,
			&item.ReadOnly, &storageConfiguration, &item.CredentialReference,
			&storageMetadata, &binding.Default, &binding.HostPath,
			&binding.ContainerPath, &binding.ReadOnly, &bindingConfiguration); err != nil {
			return nil, err
		}
		item.Type = domain.StorageType(storageType)
		binding.RuntimeID = runtimeID
		if err := json.Unmarshal([]byte(storageConfiguration), &item.Configuration); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(storageMetadata), &item.Metadata); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(bindingConfiguration), &binding.Configuration); err != nil {
			return nil, err
		}
		item.RuntimeBindings = []domain.StorageRuntimeBinding{binding}
		storages = append(storages, item)
	}
	return storages, rows.Err()
}

func (r *Repository) FindDefaultRuntimeStorage(ctx context.Context, versionID, runtimeID string) (*domain.StorageResource, error) {
	storages, err := r.ListRuntimeStorages(ctx, versionID, runtimeID)
	if err != nil {
		return nil, err
	}
	for i := range storages {
		if storages[i].RuntimeBindings[0].Default {
			return &storages[i], nil
		}
	}
	return nil, sql.ErrNoRows
}
