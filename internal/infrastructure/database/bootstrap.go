package database

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/UFFeScience/akoflow/internal/infrastructure/database/schema"
)

func Bootstrap(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("bootstrap requires a database")
	}
	if _, err := db.ExecContext(ctx, `PRAGMA foreign_keys = ON`); err != nil {
		return fmt.Errorf("enable foreign keys: %w", err)
	}
	empty, err := isEmpty(ctx, db)
	if err != nil {
		return err
	}
	if empty {
		return installSchema(ctx, db)
	}
	if err := migrateSchema(ctx, db); err != nil {
		return err
	}
	return Validate(ctx, db)
}

func migrateSchema(ctx context.Context, db *sql.DB) error {
	var version int
	if err := db.QueryRowContext(ctx, `SELECT COALESCE(MAX(version), 0) FROM schema_metadata`).Scan(&version); err != nil {
		return fmt.Errorf("%w: read schema version: %v", ErrIncompatibleSchema, err)
	}
	if version == schema.Version {
		return nil
	}
	if version == 8 && schema.Version == 9 {
		return migrateV8ToV9(ctx, db)
	}
	if version == 7 && (schema.Version == 8 || schema.Version == 9) {
		return migrateV7ToV8(ctx, db)
	}
	if version == 6 && schema.Version == 7 {
		return migrateV6ToV7(ctx, db)
	}
	if version == 5 && schema.Version == 6 {
		return migrateV5ToV6(ctx, db)
	}
	if version == 4 && schema.Version == 5 {
		return migrateV4ToV5(ctx, db)
	}
	if version == 3 && schema.Version == 4 {
		return migrateV3ToV4(ctx, db)
	}
	if version == 2 && schema.Version == 3 {
		return migrateV2ToV3(ctx, db)
	}
	if version != 1 || schema.Version != 2 {
		return fmt.Errorf("%w: no migration from version %d to %d", ErrIncompatibleSchema, version, schema.Version)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin schema migration: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS environment_connection_checks (
		id TEXT PRIMARY KEY,
		connection_id TEXT NOT NULL REFERENCES environment_connections(id) ON DELETE CASCADE,
		status TEXT NOT NULL CHECK(status IN ('online', 'offline')),
		message TEXT NOT NULL DEFAULT '', latency_ms REAL NOT NULL DEFAULT 0,
		checked_at DATETIME NOT NULL, metadata TEXT NOT NULL DEFAULT '{}'
	); CREATE INDEX IF NOT EXISTS environment_connection_checks_history_idx
		ON environment_connection_checks(connection_id, checked_at DESC);`); err != nil {
		return fmt.Errorf("migrate connection history: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE schema_metadata SET version=?, applied_at=?, checksum=?`,
		schema.Version, time.Now().UTC(), schemaChecksum()); err != nil {
		return fmt.Errorf("update schema metadata: %w", err)
	}
	return tx.Commit()
}

func migrateV7ToV8(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `
		CREATE TABLE artifact_versions (
			id TEXT PRIMARY KEY, artifact_id TEXT NOT NULL, version TEXT NOT NULL,
			scope TEXT NOT NULL, scope_id TEXT NOT NULL DEFAULT '',
			UNIQUE(artifact_id, version, scope, scope_id));
		CREATE TABLE artifact_variants (
			id TEXT PRIMARY KEY, artifact_version_id TEXT NOT NULL REFERENCES artifact_versions(id),
			digest TEXT NOT NULL, format TEXT NOT NULL, architecture TEXT NOT NULL DEFAULT '',
			size_bytes INTEGER NOT NULL DEFAULT 0, UNIQUE(artifact_version_id, digest));
		CREATE TABLE transfer_endpoints (
			id TEXT PRIMARY KEY, kind TEXT NOT NULL, uri TEXT NOT NULL, resource_id TEXT,
			environment_id TEXT, configuration TEXT NOT NULL DEFAULT '{}');
		CREATE TABLE connector_bindings (
			id TEXT PRIMARY KEY, endpoint_id TEXT NOT NULL REFERENCES transfer_endpoints(id) ON DELETE CASCADE,
			connector TEXT NOT NULL, credential_ref TEXT NOT NULL DEFAULT '', enabled INTEGER NOT NULL DEFAULT 1);
		CREATE TABLE artifact_locations (
			id TEXT PRIMARY KEY, variant_id TEXT NOT NULL REFERENCES artifact_variants(id) ON DELETE CASCADE,
			endpoint_id TEXT NOT NULL REFERENCES transfer_endpoints(id), uri TEXT NOT NULL, digest TEXT NOT NULL,
			scope TEXT NOT NULL, scope_id TEXT NOT NULL DEFAULT '', available INTEGER NOT NULL DEFAULT 1);
		CREATE TABLE artifact_materializations (
			id TEXT PRIMARY KEY, run_id TEXT NOT NULL DEFAULT '', activity_id TEXT NOT NULL DEFAULT '',
			variant_id TEXT NOT NULL, digest TEXT NOT NULL, resource_id TEXT NOT NULL,
			environment_id TEXT NOT NULL DEFAULT '', destination_path TEXT NOT NULL, status TEXT NOT NULL,
			verified_digest TEXT NOT NULL DEFAULT '', updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP);
		CREATE TABLE transfer_runs_v2 (
			id TEXT PRIMARY KEY, plan_id TEXT NOT NULL, strategy TEXT NOT NULL, status TEXT NOT NULL,
			verified_blobs TEXT NOT NULL DEFAULT '[]', completed_chunks TEXT NOT NULL DEFAULT '[]',
			error TEXT NOT NULL DEFAULT '', created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP);
		CREATE TABLE storage_download_runs (
			id TEXT PRIMARY KEY,
			storage_resource_id TEXT NOT NULL REFERENCES storage_resources(id) ON DELETE CASCADE,
			path TEXT NOT NULL,
			status TEXT NOT NULL CHECK(status IN ('queued','ready','streaming','completed','failed')),
			strategy TEXT NOT NULL DEFAULT 'stream', url TEXT NOT NULL DEFAULT '',
			size_bytes INTEGER NOT NULL DEFAULT 0, transferred_bytes INTEGER NOT NULL DEFAULT 0,
			error TEXT NOT NULL DEFAULT '', created_at DATETIME NOT NULL, updated_at DATETIME NOT NULL);
		CREATE TABLE storage_index_runs (
			id TEXT PRIMARY KEY,
			storage_resource_id TEXT NOT NULL REFERENCES storage_resources(id) ON DELETE CASCADE,
			status TEXT NOT NULL, indexed_entries INTEGER NOT NULL DEFAULT 0,
			error TEXT NOT NULL DEFAULT '', created_at DATETIME NOT NULL, finished_at DATETIME);`); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE schema_metadata SET version=?, applied_at=?, checksum=?`, schema.Version, time.Now().UTC(), schemaChecksum()); err != nil {
		return err
	}
	return tx.Commit()
}

func migrateV8ToV9(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS storage_download_runs (
		id TEXT PRIMARY KEY,
		storage_resource_id TEXT NOT NULL REFERENCES storage_resources(id) ON DELETE CASCADE,
		path TEXT NOT NULL,
		status TEXT NOT NULL CHECK(status IN ('queued','ready','streaming','completed','failed')),
		strategy TEXT NOT NULL DEFAULT 'stream', url TEXT NOT NULL DEFAULT '',
		size_bytes INTEGER NOT NULL DEFAULT 0, transferred_bytes INTEGER NOT NULL DEFAULT 0,
		error TEXT NOT NULL DEFAULT '', created_at DATETIME NOT NULL, updated_at DATETIME NOT NULL);
		CREATE TABLE IF NOT EXISTS storage_index_runs (
		id TEXT PRIMARY KEY,
		storage_resource_id TEXT NOT NULL REFERENCES storage_resources(id) ON DELETE CASCADE,
		status TEXT NOT NULL, indexed_entries INTEGER NOT NULL DEFAULT 0,
		error TEXT NOT NULL DEFAULT '', created_at DATETIME NOT NULL, finished_at DATETIME);`); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE schema_metadata SET version=?, applied_at=?, checksum=?`,
		schema.Version, time.Now().UTC(), schemaChecksum()); err != nil {
		return err
	}
	return tx.Commit()
}

func migrateV6ToV7(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `CREATE TABLE console_sessions (
		id TEXT PRIMARY KEY, resource_id TEXT NOT NULL REFERENCES resources(id), runtime_id TEXT NOT NULL,
		connection_id TEXT NOT NULL, actor_id TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL CHECK(status IN ('starting','connected','closed','failed')),
		external_id TEXT NOT NULL DEFAULT '', failure TEXT NOT NULL DEFAULT '', created_at DATETIME NOT NULL,
		connected_at DATETIME, finished_at DATETIME);
		CREATE INDEX console_sessions_created_idx ON console_sessions(created_at DESC);`); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE schema_metadata SET version=?, applied_at=?, checksum=?`, schema.Version, time.Now().UTC(), schemaChecksum()); err != nil {
		return err
	}
	return tx.Commit()
}

func migrateV5ToV6(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `ALTER TABLE activity_handles ADD COLUMN log TEXT NOT NULL DEFAULT ''`); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE schema_metadata SET version=?, applied_at=?, checksum=?`, schema.Version, time.Now().UTC(), schemaChecksum()); err != nil {
		return err
	}
	return tx.Commit()
}

func migrateV4ToV5(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `CREATE TABLE console_session_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT, session_id TEXT NOT NULL,
		direction TEXT NOT NULL CHECK(direction IN ('input','output')), payload BLOB NOT NULL, occurred_at DATETIME NOT NULL);
		CREATE INDEX console_session_logs_session_idx ON console_session_logs(session_id, id);`); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE schema_metadata SET version=?, applied_at=?, checksum=?`, schema.Version, time.Now().UTC(), schemaChecksum()); err != nil {
		return err
	}
	return tx.Commit()
}

func migrateV3ToV4(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `CREATE TABLE audit_events (
		id TEXT PRIMARY KEY, event_type TEXT NOT NULL, actor_id TEXT NOT NULL DEFAULT '', actor_type TEXT NOT NULL DEFAULT '',
		environment_id TEXT NOT NULL DEFAULT '', resource_id TEXT NOT NULL DEFAULT '', connection_id TEXT NOT NULL DEFAULT '',
		runtime_id TEXT NOT NULL DEFAULT '', session_id TEXT NOT NULL DEFAULT '', execution_id TEXT NOT NULL DEFAULT '',
		external_id TEXT NOT NULL DEFAULT '', outcome TEXT NOT NULL CHECK(outcome IN ('started','succeeded','failed')),
		summary TEXT NOT NULL, metadata TEXT NOT NULL DEFAULT '{}', occurred_at DATETIME NOT NULL);
		CREATE INDEX audit_events_time_idx ON audit_events(occurred_at DESC);
		CREATE INDEX audit_events_resource_idx ON audit_events(resource_id, occurred_at DESC);
		CREATE INDEX audit_events_environment_idx ON audit_events(environment_id, occurred_at DESC);
		CREATE INDEX audit_events_connection_idx ON audit_events(connection_id, occurred_at DESC);
		CREATE TABLE console_commands (
		id TEXT PRIMARY KEY, resource_id TEXT NOT NULL REFERENCES resources(id), runtime_id TEXT NOT NULL,
		connection_id TEXT NOT NULL, actor_id TEXT NOT NULL DEFAULT '', command_text TEXT NOT NULL,
		working_directory TEXT NOT NULL DEFAULT '', environment TEXT NOT NULL DEFAULT '{}', cpu_cores INTEGER NOT NULL DEFAULT 0,
		memory_bytes INTEGER NOT NULL DEFAULT 0, timeout_seconds INTEGER NOT NULL,
		status TEXT NOT NULL CHECK(status IN ('running','completed','failed')), stdout TEXT NOT NULL DEFAULT '',
		stderr TEXT NOT NULL DEFAULT '', exit_code INTEGER, external_id TEXT NOT NULL DEFAULT '', failure TEXT NOT NULL DEFAULT '',
		created_at DATETIME NOT NULL, started_at DATETIME NOT NULL, finished_at DATETIME);
		CREATE INDEX console_commands_created_idx ON console_commands(created_at DESC);
		CREATE INDEX console_commands_resource_idx ON console_commands(resource_id, created_at DESC);`); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE schema_metadata SET version=?, applied_at=?, checksum=?`, schema.Version, time.Now().UTC(), schemaChecksum()); err != nil {
		return err
	}
	return tx.Commit()
}

func migrateV2ToV3(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	statements := []string{
		`CREATE TABLE simulation_engines (id TEXT PRIMARY KEY, name TEXT NOT NULL, driver TEXT NOT NULL UNIQUE, enabled INTEGER NOT NULL DEFAULT 1)`,
		strings.Join([]string{
			`CREATE TABLE simulation_scenarios (id TEXT PRIMARY KEY, name TEXT NOT NULL,`,
			`environment_version_id TEXT NOT NULL REFERENCES environment_versions(id),`,
			`environment_snapshot_id TEXT NOT NULL DEFAULT '',`,
			`engine_id TEXT NOT NULL REFERENCES simulation_engines(id),`,
			`seed INTEGER NOT NULL DEFAULT 0, network_overrides TEXT NOT NULL DEFAULT '{}',`,
			`interference_model TEXT NOT NULL DEFAULT '{}', cost_model TEXT NOT NULL DEFAULT '{}',`,
			`data_scale REAL NOT NULL DEFAULT 1, created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP)`,
		}, " "),
		strings.Join([]string{
			`CREATE TABLE simulation_runs (id TEXT PRIMARY KEY,`,
			`scenario_id TEXT NOT NULL REFERENCES simulation_scenarios(id),`,
			`execution_run_id TEXT NOT NULL UNIQUE REFERENCES execution_runs(id),`,
			`created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP)`,
		}, " "),
	}
	for _, statement := range statements {
		if _, err = tx.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	if _, err = tx.ExecContext(ctx, `UPDATE schema_metadata SET version=?, applied_at=?, checksum=?`, schema.Version, time.Now().UTC(), schemaChecksum()); err != nil {
		return err
	}
	return tx.Commit()
}

func installSchema(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin schema bootstrap: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, schema.SQL); err != nil {
		return fmt.Errorf("apply canonical schema: %w", err)
	}
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO schema_metadata(version, applied_at, checksum) VALUES (?, ?, ?)`,
		schema.Version,
		time.Now().UTC(),
		schemaChecksum(),
	); err != nil {
		return fmt.Errorf("record schema metadata: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit schema bootstrap: %w", err)
	}
	return Validate(ctx, db)
}

func isEmpty(ctx context.Context, db *sql.DB) (bool, error) {
	var count int
	err := db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM sqlite_master
		WHERE type = 'table' AND name NOT LIKE 'sqlite_%'
	`).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("inspect database schema: %w", err)
	}
	return count == 0, nil
}

func schemaChecksum() string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(schema.SQL)))
}
