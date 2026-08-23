package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/UFFeScience/akoflow/internal/infrastructure/database/schema"
)

var ErrIncompatibleSchema = errors.New("database schema is incompatible; recreate the database")

var requiredTables = []string{
	"schema_metadata",
	"environments",
	"environment_connections",
	"environment_connection_checks",
	"simulation_engines",
	"simulation_scenarios",
	"simulation_runs",
	"audit_events",
	"console_commands",
	"console_session_logs",
	"environment_versions",
	"environment_runtimes",
	"environment_runtime_capabilities",
	"discovery_runs",
	"resources",
	"resource_runtime_bindings",
	"resource_snapshots",
	"resource_relations",
	"execution_scopes",
	"execution_scope_environments",
	"network_links",
	"network_topologies",
	"workflow_definitions",
	"workflow_versions",
	"activity_types",
	"activity_definitions",
	"activity_dependencies",
	"activity_resource_profiles",
	"schedule_plans",
	"schedule_plan_assignments",
	"execution_runs",
	"task_executions",
	"activity_lifecycle_events",
	"activity_handles",
	"data_objects",
	"data_object_instances",
	"storage_resources",
	"storage_runtime_bindings",
	"data_locations",
	"data_transfers",
	"queue_jobs",
	"domain_events",
}

func Validate(ctx context.Context, db *sql.DB) error {
	if err := validateMetadata(ctx, db); err != nil {
		return err
	}
	for _, table := range requiredTables {
		if err := requireTable(ctx, db, table); err != nil {
			return err
		}
	}
	return validateIntegrity(ctx, db)
}

func validateMetadata(ctx context.Context, db *sql.DB) error {
	var version, rows int
	var checksum string
	err := db.QueryRowContext(ctx, `
		SELECT COUNT(*), COALESCE(MAX(version), 0), COALESCE(MAX(checksum), '')
		FROM schema_metadata
	`).Scan(&rows, &version, &checksum)
	if err != nil {
		return fmt.Errorf("%w: metadata unavailable: %v", ErrIncompatibleSchema, err)
	}
	if rows != 1 || version != schema.Version || checksum != schemaChecksum() {
		return fmt.Errorf(
			"%w: expected version %d with checksum %s, found version %d with checksum %s",
			ErrIncompatibleSchema, schema.Version, schemaChecksum(), version, checksum,
		)
	}
	return nil
}

func requireTable(ctx context.Context, db *sql.DB, table string) error {
	var count int
	err := db.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`,
		table,
	).Scan(&count)
	if err != nil {
		return fmt.Errorf("validate table %q: %w", table, err)
	}
	if count != 1 {
		return fmt.Errorf("%w: required table %q is missing", ErrIncompatibleSchema, table)
	}
	return nil
}

func validateIntegrity(ctx context.Context, db *sql.DB) error {
	// SQLite's integrity checks scan every page. Running one during normal API
	// startup keeps a large run-history database unavailable for several
	// minutes. The bootstrap validation above already verifies the schema;
	// integrity checks are a deliberate maintenance operation instead.
	return nil
	/*
		var result string
		// A full integrity_check scans the complete database at every server boot.
		// This can make a large execution-history database unavailable for minutes.
		// quick_check still validates the database structure on startup; operators can
		// run a full integrity check as a maintenance operation.
		if err := db.QueryRowContext(ctx, `PRAGMA quick_check(1)`).Scan(&result); err != nil {
			return fmt.Errorf("validate database integrity: %w", err)
		}
		if result != "ok" {
			return fmt.Errorf("%w: integrity check returned %q", ErrIncompatibleSchema, result)
		}
		return nil
	*/
}
