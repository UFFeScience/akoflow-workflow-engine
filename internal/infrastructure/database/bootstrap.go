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
