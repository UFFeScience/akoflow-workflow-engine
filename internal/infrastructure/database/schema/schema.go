package schema

import (
	"database/sql"
	"fmt"
)

var statements = []string{
	`CREATE TABLE IF NOT EXISTS runtimes (
		name TEXT PRIMARY KEY,
		status INTEGER NOT NULL DEFAULT 0,
		metadata TEXT NOT NULL DEFAULT '{}',
		max_nodes INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		deleted_at DATETIME
	)`,
	`CREATE TABLE IF NOT EXISTS workflows (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		namespace TEXT NOT NULL DEFAULT '',
		name TEXT NOT NULL,
		raw_workflow TEXT NOT NULL,
		status INTEGER NOT NULL DEFAULT 0,
		runtime TEXT NOT NULL DEFAULT '',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		deleted_at DATETIME
	)`,
	`CREATE TABLE IF NOT EXISTS activities (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		workflow_id INTEGER NOT NULL REFERENCES workflows(id),
		namespace TEXT NOT NULL DEFAULT '',
		name TEXT NOT NULL,
		image TEXT NOT NULL DEFAULT '',
		runtime TEXT NOT NULL DEFAULT '',
		resource_k8s_base64 TEXT NOT NULL DEFAULT '',
		status INTEGER NOT NULL DEFAULT 0,
		proc_id TEXT,
		resource_selector TEXT,
		mount_path TEXT,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		started_at DATETIME,
		finished_at DATETIME
	)`,
	`CREATE TABLE IF NOT EXISTS activities_dependencies (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		workflow_id INTEGER NOT NULL REFERENCES workflows(id),
		activity_id INTEGER NOT NULL REFERENCES activities(id),
		depend_on_activity INTEGER NOT NULL REFERENCES activities(id)
	)`,
	`CREATE TABLE IF NOT EXISTS pre_activities (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		activity_id INTEGER NOT NULL REFERENCES activities(id),
		workflow_id INTEGER NOT NULL REFERENCES workflows(id),
		namespace TEXT NOT NULL DEFAULT '',
		name TEXT NOT NULL,
		resource_k8s_base64 TEXT,
		status INTEGER NOT NULL DEFAULT 0,
		log TEXT
	)`,
	`CREATE TABLE IF NOT EXISTS activities_schedules (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		workflow_id INTEGER NOT NULL REFERENCES workflows(id),
		activity_id INTEGER NOT NULL REFERENCES activities(id),
		resource_id TEXT NOT NULL,
		schedule_name TEXT NOT NULL,
		cpu_required REAL NOT NULL DEFAULT 0,
		memory_required REAL NOT NULL DEFAULT 0,
		metadata TEXT NOT NULL DEFAULT '{}',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`,
	`CREATE TABLE IF NOT EXISTS storages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		workflow_id INTEGER NOT NULL REFERENCES workflows(id),
		activity_id INTEGER NOT NULL REFERENCES activities(id),
		pvc_name TEXT,
		namespace TEXT NOT NULL DEFAULT '',
		status INTEGER NOT NULL DEFAULT 0,
		storage_mount_path TEXT NOT NULL DEFAULT '',
		storage_class TEXT NOT NULL DEFAULT '',
		storage_size TEXT NOT NULL DEFAULT '',
		initial_file_list TEXT NOT NULL DEFAULT '',
		end_file_list TEXT NOT NULL DEFAULT '',
		initial_disk_spec TEXT NOT NULL DEFAULT '',
		end_disk_spec TEXT NOT NULL DEFAULT '',
		keep_storage_after_finish INTEGER NOT NULL DEFAULT 0,
		detached DATETIME,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`,
	`CREATE TABLE IF NOT EXISTS logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		activity_id INTEGER NOT NULL REFERENCES activities(id),
		logs TEXT NOT NULL DEFAULT '',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`,
	`CREATE TABLE IF NOT EXISTS metrics (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		activity_id INTEGER NOT NULL REFERENCES activities(id),
		cpu TEXT NOT NULL DEFAULT '',
		memory TEXT NOT NULL DEFAULT '',
		window TEXT NOT NULL DEFAULT '',
		timestamp TEXT NOT NULL DEFAULT '',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`,
	`CREATE TABLE IF NOT EXISTS workflow_executions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		namespace TEXT NOT NULL DEFAULT '',
		name TEXT NOT NULL DEFAULT '',
		workflow_id INTEGER NOT NULL REFERENCES workflows(id),
		status TEXT NOT NULL DEFAULT '',
		runtime TEXT NOT NULL DEFAULT '',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		deleted_at DATETIME
	)`,
	`CREATE TABLE IF NOT EXISTS schedules (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		type TEXT NOT NULL,
		code TEXT NOT NULL,
		name TEXT NOT NULL UNIQUE,
		plugin_so_path TEXT NOT NULL DEFAULT '',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`,
	`CREATE TABLE IF NOT EXISTS environments (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL DEFAULT 'defined',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`,
	`CREATE TABLE IF NOT EXISTS environment_connections (
		id TEXT PRIMARY KEY,
		environment_id TEXT NOT NULL REFERENCES environments(id),
		name TEXT NOT NULL,
		type TEXT NOT NULL,
		endpoint TEXT NOT NULL DEFAULT '',
		username TEXT NOT NULL DEFAULT '',
		credential_ref TEXT NOT NULL DEFAULT '',
		configuration TEXT NOT NULL DEFAULT '{}',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(environment_id, name)
	)`,
	`CREATE TABLE IF NOT EXISTS environment_versions (
		id TEXT PRIMARY KEY,
		environment_id TEXT NOT NULL REFERENCES environments(id),
		version INTEGER NOT NULL,
		status TEXT NOT NULL,
		network_model TEXT NOT NULL,
		interference_model TEXT NOT NULL,
		cost_model TEXT NOT NULL,
		storage_model TEXT NOT NULL,
		configuration_hash TEXT NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		published_at DATETIME,
		UNIQUE(environment_id, version)
	)`,
	`CREATE TABLE IF NOT EXISTS environment_runtimes (
		environment_version_id TEXT NOT NULL REFERENCES environment_versions(id),
		runtime_id TEXT NOT NULL REFERENCES runtimes(name),
		role TEXT NOT NULL DEFAULT '',
		configuration TEXT NOT NULL DEFAULT '{}',
		PRIMARY KEY(environment_version_id, runtime_id)
	)`,
	`CREATE TABLE IF NOT EXISTS environment_runtime_capabilities (
		environment_version_id TEXT NOT NULL,
		runtime_id TEXT NOT NULL,
		capabilities TEXT NOT NULL DEFAULT '{}',
		PRIMARY KEY(environment_version_id, runtime_id),
		FOREIGN KEY(environment_version_id, runtime_id)
			REFERENCES environment_runtimes(environment_version_id, runtime_id)
	)`,
	`CREATE TABLE IF NOT EXISTS discovery_runs (
		id TEXT PRIMARY KEY,
		environment_version_id TEXT NOT NULL REFERENCES environment_versions(id),
		provider TEXT NOT NULL,
		status TEXT NOT NULL,
		resources_found INTEGER NOT NULL DEFAULT 0,
		error TEXT NOT NULL DEFAULT '',
		started_at DATETIME NOT NULL,
		finished_at DATETIME
	)`,
	`CREATE TABLE IF NOT EXISTS resources (
		id TEXT PRIMARY KEY,
		environment_version_id TEXT NOT NULL REFERENCES environment_versions(id),
		runtime_id TEXT NOT NULL REFERENCES runtimes(name),
		parent_resource_id TEXT REFERENCES resources(id),
		type TEXT NOT NULL,
		name TEXT NOT NULL,
		provider_id TEXT NOT NULL,
		tier TEXT NOT NULL DEFAULT '',
		region TEXT NOT NULL DEFAULT '',
		zone TEXT NOT NULL DEFAULT '',
		architecture TEXT NOT NULL DEFAULT '',
		cpu_cores INTEGER NOT NULL DEFAULT 0,
		cpu_capacity REAL NOT NULL DEFAULT 0,
		memory_bytes INTEGER NOT NULL DEFAULT 0,
		storage_bytes INTEGER NOT NULL DEFAULT 0,
		compute_speedup REAL NOT NULL DEFAULT 1,
		price_per_second REAL NOT NULL DEFAULT 0,
		boot_overhead_seconds REAL NOT NULL DEFAULT 0,
		container_overhead_seconds REAL NOT NULL DEFAULT 0,
		schedulable INTEGER NOT NULL DEFAULT 1,
		metadata TEXT NOT NULL DEFAULT '{}',
		UNIQUE(environment_version_id, provider_id)
	)`,
	`CREATE TABLE IF NOT EXISTS resource_snapshots (
		id TEXT PRIMARY KEY,
		resource_id TEXT NOT NULL REFERENCES resources(id),
		captured_at DATETIME NOT NULL,
		available INTEGER NOT NULL,
		cpu_used REAL NOT NULL DEFAULT 0,
		memory_used_bytes INTEGER NOT NULL DEFAULT 0,
		network_in_bps REAL NOT NULL DEFAULT 0,
		network_out_bps REAL NOT NULL DEFAULT 0,
		disk_read_bps REAL NOT NULL DEFAULT 0,
		disk_write_bps REAL NOT NULL DEFAULT 0,
		queue_length INTEGER NOT NULL DEFAULT 0,
		metadata TEXT NOT NULL DEFAULT '{}'
	)`,
	`CREATE TABLE IF NOT EXISTS network_links (
		id TEXT PRIMARY KEY,
		environment_version_id TEXT NOT NULL REFERENCES environment_versions(id),
		source_resource_id TEXT NOT NULL REFERENCES resources(id),
		target_resource_id TEXT NOT NULL REFERENCES resources(id),
		bandwidth_bits_per_second REAL NOT NULL CHECK(bandwidth_bits_per_second > 0),
		latency_seconds REAL NOT NULL DEFAULT 0 CHECK(latency_seconds >= 0),
		price_per_byte REAL NOT NULL DEFAULT 0 CHECK(price_per_byte >= 0),
		bidirectional INTEGER NOT NULL DEFAULT 1,
		sharing_policy TEXT NOT NULL DEFAULT 'independent',
		max_concurrent_transfers INTEGER NOT NULL DEFAULT 0,
		metadata TEXT NOT NULL DEFAULT '{}',
		UNIQUE(environment_version_id, source_resource_id, target_resource_id)
	)`,
	`CREATE TABLE IF NOT EXISTS normalized_workflows (
		id TEXT PRIMARY KEY,
		external_id TEXT NOT NULL UNIQUE,
		name TEXT NOT NULL,
		namespace TEXT NOT NULL DEFAULT '',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`,
	`CREATE TABLE IF NOT EXISTS workflow_versions (
		id TEXT PRIMARY KEY,
		workflow_id TEXT NOT NULL REFERENCES normalized_workflows(id),
		version INTEGER NOT NULL,
		definition_hash TEXT NOT NULL,
		raw_definition TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL DEFAULT 'draft',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(workflow_id, version)
	)`,
	`CREATE TABLE IF NOT EXISTS activity_types (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		application TEXT NOT NULL DEFAULT '',
		default_image TEXT NOT NULL DEFAULT '',
		cpu_intensity REAL NOT NULL DEFAULT 0,
		memory_intensity REAL NOT NULL DEFAULT 0,
		io_intensity REAL NOT NULL DEFAULT 0,
		network_intensity REAL NOT NULL DEFAULT 0,
		metadata TEXT NOT NULL DEFAULT '{}'
	)`,
	`CREATE TABLE IF NOT EXISTS normalized_activities (
		id TEXT PRIMARY KEY,
		workflow_version_id TEXT NOT NULL REFERENCES workflow_versions(id),
		activity_type_id TEXT NOT NULL REFERENCES activity_types(id),
		external_id TEXT NOT NULL,
		name TEXT NOT NULL,
		command TEXT NOT NULL,
		image TEXT NOT NULL DEFAULT '',
		priority INTEGER NOT NULL DEFAULT 0,
		cpu_required REAL NOT NULL DEFAULT 0,
		memory_required_bytes INTEGER NOT NULL DEFAULT 0,
		storage_required_bytes INTEGER NOT NULL DEFAULT 0,
		metadata TEXT NOT NULL DEFAULT '{}',
		UNIQUE(workflow_version_id, external_id)
	)`,
	`CREATE TABLE IF NOT EXISTS normalized_activity_dependencies (
		activity_id TEXT NOT NULL REFERENCES normalized_activities(id),
		depends_on_activity_id TEXT NOT NULL REFERENCES normalized_activities(id),
		dependency_type TEXT NOT NULL DEFAULT 'control',
		PRIMARY KEY(activity_id, depends_on_activity_id),
		CHECK(activity_id <> depends_on_activity_id)
	)`,
	`CREATE TABLE IF NOT EXISTS activity_resource_profiles (
		id TEXT PRIMARY KEY,
		activity_type_id TEXT NOT NULL REFERENCES activity_types(id),
		resource_id TEXT NOT NULL REFERENCES resources(id),
		runtime_seconds REAL NOT NULL CHECK(runtime_seconds >= 0),
		runtime_stddev_seconds REAL NOT NULL DEFAULT 0,
		cpu_utilization REAL NOT NULL DEFAULT 0,
		peak_memory_bytes INTEGER NOT NULL DEFAULT 0,
		disk_read_bytes INTEGER NOT NULL DEFAULT 0,
		disk_write_bytes INTEGER NOT NULL DEFAULT 0,
		energy_joules REAL NOT NULL DEFAULT 0,
		source TEXT NOT NULL DEFAULT 'configured',
		sample_size INTEGER NOT NULL DEFAULT 0,
		model_version TEXT NOT NULL DEFAULT '',
		metadata TEXT NOT NULL DEFAULT '{}',
		UNIQUE(activity_type_id, resource_id, model_version)
	)`,
	`CREATE TABLE IF NOT EXISTS schedule_plans (
		id TEXT PRIMARY KEY,
		workflow_version_id TEXT NOT NULL REFERENCES workflow_versions(id),
		environment_version_id TEXT NOT NULL REFERENCES environment_versions(id),
		source TEXT NOT NULL CHECK(source IN ('plugin', 'imported')),
		algorithm TEXT NOT NULL,
		algorithm_version TEXT NOT NULL DEFAULT '',
		objective TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL DEFAULT 'draft',
		deadline_seconds REAL NOT NULL DEFAULT 0,
		budget REAL NOT NULL DEFAULT 0,
		predicted_makespan_seconds REAL NOT NULL DEFAULT 0,
		predicted_cost REAL NOT NULL DEFAULT 0,
		predicted_feasible INTEGER NOT NULL DEFAULT 0,
		configuration TEXT NOT NULL DEFAULT '{}',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`,
	`CREATE TABLE IF NOT EXISTS schedule_plan_assignments (
		id TEXT PRIMARY KEY,
		schedule_plan_id TEXT NOT NULL REFERENCES schedule_plans(id),
		activity_id TEXT NOT NULL REFERENCES normalized_activities(id),
		resource_id TEXT NOT NULL REFERENCES resources(id),
		core_id TEXT NOT NULL DEFAULT '',
		slot_id TEXT NOT NULL DEFAULT '',
		order_on_resource INTEGER NOT NULL,
		priority INTEGER NOT NULL DEFAULT 0,
		predicted_ready_at REAL NOT NULL DEFAULT 0,
		predicted_start_at REAL NOT NULL DEFAULT 0,
		predicted_finish_at REAL NOT NULL DEFAULT 0,
		predicted_runtime_seconds REAL NOT NULL DEFAULT 0,
		predicted_transfer_seconds REAL NOT NULL DEFAULT 0,
		predicted_cost REAL NOT NULL DEFAULT 0,
		metadata TEXT NOT NULL DEFAULT '{}',
		UNIQUE(schedule_plan_id, activity_id),
		UNIQUE(schedule_plan_id, resource_id, core_id, order_on_resource)
	)`,
	`CREATE TABLE IF NOT EXISTS execution_runs (
		id TEXT PRIMARY KEY,
		schedule_plan_id TEXT NOT NULL REFERENCES schedule_plans(id),
		mode TEXT NOT NULL CHECK(mode IN ('real', 'simulation')),
		seed INTEGER NOT NULL DEFAULT 0,
		status TEXT NOT NULL DEFAULT 'created',
		environment_snapshot_id TEXT NOT NULL DEFAULT '',
		started_at DATETIME,
		finished_at DATETIME,
		makespan_seconds REAL NOT NULL DEFAULT 0,
		cost REAL NOT NULL DEFAULT 0,
		failure_reason TEXT NOT NULL DEFAULT '',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`,
	`CREATE TABLE IF NOT EXISTS task_executions (
		id TEXT PRIMARY KEY,
		execution_run_id TEXT NOT NULL REFERENCES execution_runs(id),
		plan_assignment_id TEXT NOT NULL REFERENCES schedule_plan_assignments(id),
		activity_id TEXT NOT NULL REFERENCES normalized_activities(id),
		planned_resource_id TEXT NOT NULL REFERENCES resources(id),
		allocated_resource_id TEXT REFERENCES resources(id),
		attempt INTEGER NOT NULL DEFAULT 1,
		status TEXT NOT NULL DEFAULT 'blocked',
		ready_at REAL NOT NULL DEFAULT 0,
		data_ready_at REAL NOT NULL DEFAULT 0,
		queued_at REAL NOT NULL DEFAULT 0,
		started_at REAL NOT NULL DEFAULT 0,
		finished_at REAL NOT NULL DEFAULT 0,
		runtime_seconds REAL NOT NULL DEFAULT 0,
		queue_seconds REAL NOT NULL DEFAULT 0,
		transfer_seconds REAL NOT NULL DEFAULT 0,
		interference_seconds REAL NOT NULL DEFAULT 0,
		overhead_seconds REAL NOT NULL DEFAULT 0,
		cost REAL NOT NULL DEFAULT 0,
		failure_reason TEXT NOT NULL DEFAULT '',
		metadata TEXT NOT NULL DEFAULT '{}',
		UNIQUE(execution_run_id, activity_id, attempt)
	)`,
	`CREATE TABLE IF NOT EXISTS activity_lifecycle_events (
		id TEXT PRIMARY KEY,
		task_execution_id TEXT NOT NULL REFERENCES task_executions(id),
		phase TEXT NOT NULL,
		status TEXT NOT NULL,
		started_at REAL NOT NULL DEFAULT 0,
		finished_at REAL NOT NULL DEFAULT 0,
		duration_seconds REAL NOT NULL DEFAULT 0,
		source TEXT NOT NULL CHECK(source IN ('measured', 'simulated')),
		error TEXT NOT NULL DEFAULT '',
		metadata TEXT NOT NULL DEFAULT '{}'
	)`,
	`CREATE TABLE IF NOT EXISTS data_objects (
		id TEXT PRIMARY KEY,
		workflow_version_id TEXT NOT NULL REFERENCES workflow_versions(id),
		producer_activity_id TEXT REFERENCES normalized_activities(id),
		logical_name TEXT NOT NULL,
		relative_path TEXT NOT NULL DEFAULT '',
		size_bytes INTEGER NOT NULL DEFAULT 0,
		checksum TEXT NOT NULL DEFAULT '',
		metadata TEXT NOT NULL DEFAULT '{}'
	)`,
	`CREATE TABLE IF NOT EXISTS data_locations (
		data_object_id TEXT NOT NULL REFERENCES data_objects(id),
		resource_id TEXT NOT NULL REFERENCES resources(id),
		execution_run_id TEXT REFERENCES execution_runs(id),
		available_at REAL NOT NULL DEFAULT 0,
		status TEXT NOT NULL DEFAULT 'available',
		PRIMARY KEY(data_object_id, resource_id, execution_run_id)
	)`,
	`CREATE TABLE IF NOT EXISTS data_transfers (
		id TEXT PRIMARY KEY,
		execution_run_id TEXT NOT NULL REFERENCES execution_runs(id),
		data_object_id TEXT REFERENCES data_objects(id),
		producer_activity_id TEXT REFERENCES normalized_activities(id),
		consumer_activity_id TEXT REFERENCES normalized_activities(id),
		source_resource_id TEXT NOT NULL REFERENCES resources(id),
		target_resource_id TEXT NOT NULL REFERENCES resources(id),
		bytes INTEGER NOT NULL DEFAULT 0,
		status TEXT NOT NULL,
		started_at REAL NOT NULL DEFAULT 0,
		finished_at REAL NOT NULL DEFAULT 0,
		duration_seconds REAL NOT NULL DEFAULT 0,
		cost REAL NOT NULL DEFAULT 0,
		metadata TEXT NOT NULL DEFAULT '{}'
	)`,
	`CREATE TABLE IF NOT EXISTS queue_jobs (
		id TEXT PRIMARY KEY,
		category TEXT NOT NULL,
		event_type TEXT NOT NULL,
		aggregate_type TEXT NOT NULL DEFAULT '',
		aggregate_id TEXT NOT NULL DEFAULT '',
		payload BLOB NOT NULL DEFAULT X'',
		status TEXT NOT NULL DEFAULT 'pending'
			CHECK(status IN ('pending', 'leased', 'completed', 'failed', 'cancelled')),
		priority INTEGER NOT NULL DEFAULT 0,
		available_at DATETIME NOT NULL,
		lease_owner TEXT NOT NULL DEFAULT '',
		lease_expires_at DATETIME,
		attempts INTEGER NOT NULL DEFAULT 0,
		max_attempts INTEGER NOT NULL DEFAULT 5 CHECK(max_attempts > 0),
		idempotency_key TEXT,
		last_error TEXT NOT NULL DEFAULT '',
		created_at DATETIME NOT NULL,
		started_at DATETIME,
		completed_at DATETIME
	)`,
	`CREATE UNIQUE INDEX IF NOT EXISTS queue_jobs_idempotency_idx
		ON queue_jobs(idempotency_key) WHERE idempotency_key IS NOT NULL AND idempotency_key <> ''`,
	`CREATE INDEX IF NOT EXISTS queue_jobs_available_idx
		ON queue_jobs(status, category, available_at, priority DESC)`,
}

func Apply(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err = tx.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		return err
	}
	for i, statement := range statements {
		if _, err = tx.Exec(statement); err != nil {
			return fmt.Errorf("apply schema statement %d: %w", i+1, err)
		}
	}
	if err = ensureColumn(tx, "environments", "status", "TEXT NOT NULL DEFAULT 'defined'"); err != nil {
		return fmt.Errorf("migrate environments status: %w", err)
	}
	return tx.Commit()
}

func ensureColumn(tx *sql.Tx, table, column, definition string) error {
	rows, err := tx.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		return err
	}
	found := false
	for rows.Next() {
		var cid int
		var name, columnType string
		var notNull, primaryKey int
		var defaultValue sql.NullString
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			rows.Close()
			return err
		}
		if name == column {
			found = true
		}
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if found {
		return nil
	}
	_, err = tx.Exec(`ALTER TABLE ` + table + ` ADD COLUMN ` + column + ` ` + definition)
	return err
}
