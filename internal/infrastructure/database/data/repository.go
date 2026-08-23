package data

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"path"

	"github.com/UFFeScience/akoflow/internal/application/ports"
	"github.com/UFFeScience/akoflow/internal/domain"
)

type Repository struct{ db *sql.DB }

var _ ports.DataCatalog = (*Repository)(nil)

func New(db *sql.DB) *Repository { return &Repository{db: db} }

func (r *Repository) CatalogArtifacts(ctx context.Context, handle domain.ActivityHandle) error {
	if handle.Artifacts == nil {
		return nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var workflowVersionID, environmentVersionID string
	if err := tx.QueryRowContext(ctx, `SELECT workflow_version_id FROM activity_definitions WHERE id=?`,
		handle.ActivityID).Scan(&workflowVersionID); err != nil {
		return fmt.Errorf("resolve artifact workflow: %w", err)
	}
	if err := tx.QueryRowContext(ctx, `SELECT environment_version_id FROM resources WHERE id=?`,
		handle.ResourceID).Scan(&environmentVersionID); err != nil {
		return fmt.Errorf("resolve artifact environment: %w", err)
	}
	storageType := domain.StorageLocal
	locationStatus := domain.DataLocationEphemeral
	if configured, _ := handle.Metadata["artifactStorageType"].(string); configured == "pvc" || configured == "nfs" {
		storageType = domain.StorageType(configured)
		locationStatus = domain.DataLocationAvailable
	}
	storageID := stableID("storage", environmentVersionID, handle.ResourceID, string(storageType))
	configuredStorageID, _ := handle.Metadata["artifactStorageResourceId"].(string)
	if configuredStorageID != "" {
		storageID = configuredStorageID
	} else if _, err := tx.ExecContext(ctx, `INSERT INTO storage_resources (
		id, environment_version_id, name, type, endpoint, shared, configuration, metadata
	) VALUES (?, ?, ?, ?, ?, ?, '{}', '{"managedBy":"akoflow"}')
	ON CONFLICT(id) DO NOTHING`, storageID, environmentVersionID,
		"workspace-"+handle.ResourceID, storageType, handle.Artifacts.Root,
		storageType == domain.StoragePVC || storageType == domain.StorageNFS); err != nil {
		return fmt.Errorf("register workspace storage: %w", err)
	}
	for _, artifact := range handle.Artifacts.Files {
		if artifact.Change == domain.ArtifactDeleted {
			continue
		}
		objectID := stableID("object", workflowVersionID, handle.ActivityID, artifact.Path)
		if _, err := tx.ExecContext(ctx, `INSERT INTO data_objects (
			id, workflow_version_id, producer_activity_id, logical_name, relative_path, declared, metadata
		) VALUES (?, ?, ?, ?, ?, 0, '{"source":"runtime-discovery"}')
		ON CONFLICT(id) DO UPDATE SET relative_path=excluded.relative_path`, objectID,
			workflowVersionID, handle.ActivityID, path.Base(artifact.Path), artifact.Path); err != nil {
			return fmt.Errorf("register data object: %w", err)
		}
		instanceID := stableID("instance", handle.RunID, handle.ActivityID,
			fmt.Sprint(handle.Artifacts.Attempt), artifact.Path)
		metadata, _ := json.Marshal(map[string]any{"change": artifact.Change})
		if _, err := tx.ExecContext(ctx, `INSERT INTO data_object_instances (
			id, data_object_id, execution_run_id, producer_activity_id, attempt,
			relative_path, size_bytes, checksum, discovered, metadata
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 1, ?)
		ON CONFLICT(execution_run_id, producer_activity_id, attempt, relative_path) DO UPDATE SET
			size_bytes=excluded.size_bytes, checksum=excluded.checksum, metadata=excluded.metadata`,
			instanceID, objectID, handle.RunID, handle.ActivityID, handle.Artifacts.Attempt,
			artifact.Path, artifact.SizeBytes, artifact.Checksum, string(metadata)); err != nil {
			return fmt.Errorf("register data object instance: %w", err)
		}
		locationID := stableID("location", instanceID, handle.ResourceID)
		locationURI := (&url.URL{Scheme: "file", Path: path.Join(handle.Artifacts.Root, artifact.Path)}).String()
		if _, err := tx.ExecContext(ctx, `INSERT INTO data_locations (
			id, data_object_instance_id, storage_resource_id, resource_id,
			execution_run_id, uri, available_at, status, metadata
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(data_object_instance_id, uri) DO UPDATE SET
			status=excluded.status, available_at=excluded.available_at`, locationID, instanceID,
			storageID, handle.ResourceID, handle.RunID, locationURI, handle.FinishedAt,
			locationStatus, locationMetadata(locationStatus)); err != nil {
			return fmt.Errorf("register data location: %w", err)
		}
	}
	return tx.Commit()
}

func locationMetadata(status domain.DataLocationStatus) string {
	if status == domain.DataLocationAvailable {
		return `{"lifetime":"storage"}`
	}
	return `{"lifetime":"pod"}`
}

func (r *Repository) ListInstances(ctx context.Context, runID string) ([]domain.DataObjectInstance, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, data_object_id, execution_run_id,
		producer_activity_id, attempt, relative_path, size_bytes, checksum, media_type,
		discovered, metadata FROM data_object_instances WHERE execution_run_id=?
		ORDER BY producer_activity_id, relative_path`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []domain.DataObjectInstance
	for rows.Next() {
		var item domain.DataObjectInstance
		var discovered bool
		var metadata []byte
		if err := rows.Scan(&item.ID, &item.DataObjectID, &item.ExecutionRunID,
			&item.ProducerActivityID, &item.Attempt, &item.RelativePath, &item.SizeBytes,
			&item.Checksum, &item.MediaType, &discovered, &metadata); err != nil {
			return nil, err
		}
		item.Discovered = discovered
		_ = json.Unmarshal(metadata, &item.Metadata)
		result = append(result, item)
	}
	return result, rows.Err()
}

func (r *Repository) ListLocations(ctx context.Context, runID string) ([]domain.DataLocation, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, data_object_instance_id,
		COALESCE(storage_resource_id, ''), COALESCE(resource_id, ''), execution_run_id,
		uri, status, metadata FROM data_locations WHERE execution_run_id=? ORDER BY uri`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []domain.DataLocation
	for rows.Next() {
		var item domain.DataLocation
		var metadata []byte
		if err := rows.Scan(&item.ID, &item.DataObjectInstanceID, &item.StorageResourceID,
			&item.ResourceID, &item.ExecutionRunID, &item.URI, &item.Status, &metadata); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(metadata, &item.Metadata)
		result = append(result, item)
	}
	return result, rows.Err()
}

func (r *Repository) ListArtifactLocations(ctx context.Context) ([]domain.ArtifactLocation, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, variant_id, endpoint_id, uri, digest, scope, scope_id, available FROM artifact_locations ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var values []domain.ArtifactLocation
	for rows.Next() {
		var v domain.ArtifactLocation
		var available bool
		if err := rows.Scan(&v.ID, &v.VariantID, &v.EndpointID, &v.URI, &v.Digest, &v.Scope, &v.ScopeID, &available); err != nil {
			return nil, err
		}
		v.Available = available
		values = append(values, v)
	}
	return values, rows.Err()
}

func (r *Repository) ListArtifacts(ctx context.Context) ([]domain.ExecutableArtifact, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT artifact_id, version FROM artifact_versions ORDER BY artifact_id, version`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var values []domain.ExecutableArtifact
	for rows.Next() {
		var value domain.ExecutableArtifact
		if err := rows.Scan(&value.ID, &value.Version); err != nil {
			return nil, err
		}
		value.Name = value.ID
		values = append(values, value)
	}
	return values, rows.Err()
}

func (r *Repository) ListArtifactMaterializations(ctx context.Context, runID string) ([]domain.ArtifactMaterialization, error) {
	query := `SELECT id, run_id, activity_id, variant_id, digest, resource_id, environment_id, destination_path, status, verified_digest FROM artifact_materializations`
	args := []any{}
	if runID != "" {
		query += ` WHERE run_id=?`
		args = append(args, runID)
	}
	query += ` ORDER BY updated_at DESC`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var values []domain.ArtifactMaterialization
	for rows.Next() {
		var v domain.ArtifactMaterialization
		if err := rows.Scan(&v.ID, &v.RunID, &v.ActivityID, &v.VariantID, &v.Digest, &v.ResourceID, &v.EnvironmentID, &v.DestinationPath, &v.Status, &v.VerifiedDigest); err != nil {
			return nil, err
		}
		values = append(values, v)
	}
	return values, rows.Err()
}
func (r *Repository) SaveArtifactMaterialization(ctx context.Context, value domain.ArtifactMaterialization) error {
	const query = `INSERT INTO artifact_materializations (
		id,run_id,activity_id,variant_id,digest,resource_id,environment_id,destination_path,status,verified_digest
	) VALUES (?,?,?,?,?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET
		run_id=excluded.run_id,
		activity_id=excluded.activity_id,
		status=excluded.status,
		verified_digest=excluded.verified_digest,
		destination_path=excluded.destination_path,
		updated_at=CURRENT_TIMESTAMP`
	_, err := r.db.ExecContext(ctx, query,
		value.ID, value.RunID, value.ActivityID, value.VariantID, value.Digest,
		value.ResourceID, value.EnvironmentID, value.DestinationPath, value.Status, value.VerifiedDigest,
	)
	return err
}

func stableID(parts ...string) string {
	hash := sha256.New()
	for _, part := range parts {
		_, _ = hash.Write([]byte(part))
		_, _ = hash.Write([]byte{0})
	}
	return hex.EncodeToString(hash.Sum(nil)[:16])
}
