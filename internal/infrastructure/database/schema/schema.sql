CREATE TABLE system_instance (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		organization TEXT NOT NULL DEFAULT '',
		location TEXT NOT NULL DEFAULT '',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
CREATE TABLE environments (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL DEFAULT 'defined',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
CREATE TABLE environment_connections (
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
	);
CREATE TABLE environment_versions (
		id TEXT PRIMARY KEY,
		environment_id TEXT NOT NULL REFERENCES environments(id),
		version INTEGER NOT NULL,
		status TEXT NOT NULL,
		network_model TEXT NOT NULL,
		interference_model TEXT NOT NULL,
		cost_model TEXT NOT NULL,
		configuration_hash TEXT NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		published_at DATETIME,
		UNIQUE(environment_id, version)
	);
CREATE TABLE environment_runtimes (
		id TEXT PRIMARY KEY,
		environment_version_id TEXT NOT NULL REFERENCES environment_versions(id),
		name TEXT NOT NULL,
		driver TEXT NOT NULL CHECK(driver IN ('slurm', 'kubernetes', 'ssh', 'local', 'serverless', 'simgrid')),
		mode TEXT NOT NULL CHECK(mode IN ('execution', 'simulation')),
		role TEXT NOT NULL DEFAULT '',
		configuration TEXT NOT NULL DEFAULT '{}',
		UNIQUE(environment_version_id, name)
	);
CREATE TABLE environment_runtime_capabilities (
		runtime_id TEXT PRIMARY KEY REFERENCES environment_runtimes(id) ON DELETE CASCADE,
		capabilities TEXT NOT NULL DEFAULT '{}',
		FOREIGN KEY(runtime_id) REFERENCES environment_runtimes(id)
	);
CREATE TABLE discovery_runs (
		id TEXT PRIMARY KEY,
		environment_version_id TEXT NOT NULL REFERENCES environment_versions(id),
		provider TEXT NOT NULL,
		status TEXT NOT NULL,
		resources_found INTEGER NOT NULL DEFAULT 0,
		error TEXT NOT NULL DEFAULT '',
		started_at DATETIME NOT NULL,
		finished_at DATETIME
	);
CREATE TABLE resources (
		id TEXT PRIMARY KEY,
		environment_version_id TEXT NOT NULL REFERENCES environment_versions(id),
		execution_target TEXT NOT NULL DEFAULT 'batch'
			CHECK(execution_target IN ('batch', 'direct')),
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
	);
CREATE TABLE resource_runtime_bindings (
		resource_id TEXT NOT NULL REFERENCES resources(id) ON DELETE CASCADE,
		runtime_id TEXT NOT NULL REFERENCES environment_runtimes(id) ON DELETE CASCADE,
		enabled INTEGER NOT NULL DEFAULT 1,
		configuration TEXT NOT NULL DEFAULT '{}',
		PRIMARY KEY(resource_id, runtime_id)
	);
CREATE TABLE resource_snapshots (
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
	);
CREATE TABLE resource_relations (
		environment_version_id TEXT NOT NULL REFERENCES environment_versions(id),
		source_resource_id TEXT NOT NULL REFERENCES resources(id) ON DELETE CASCADE,
		target_resource_id TEXT NOT NULL REFERENCES resources(id) ON DELETE CASCADE,
		relation_type TEXT NOT NULL CHECK(relation_type IN ('contains', 'member_of', 'accessible_via')),
		metadata TEXT NOT NULL DEFAULT '{}',
		PRIMARY KEY(source_resource_id, target_resource_id, relation_type),
		CHECK(source_resource_id <> target_resource_id)
	);
CREATE TABLE execution_scopes (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		network_topology_id TEXT,
		metadata TEXT NOT NULL DEFAULT '{}'
	);
CREATE TABLE execution_scope_environments (
		execution_scope_id TEXT NOT NULL REFERENCES execution_scopes(id) ON DELETE CASCADE,
		environment_version_id TEXT NOT NULL REFERENCES environment_versions(id),
		PRIMARY KEY(execution_scope_id, environment_version_id)
	);
CREATE TABLE network_topologies (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		version INTEGER NOT NULL CHECK(version > 0),
		execution_scope_id TEXT NOT NULL REFERENCES execution_scopes(id),
		metadata TEXT NOT NULL DEFAULT '{}',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
CREATE TABLE network_links (
		id TEXT PRIMARY KEY,
		topology_id TEXT NOT NULL REFERENCES network_topologies(id) ON DELETE CASCADE,
		source_resource_id TEXT NOT NULL REFERENCES resources(id),
		target_resource_id TEXT NOT NULL REFERENCES resources(id),
		bandwidth_bits_per_second REAL NOT NULL CHECK(bandwidth_bits_per_second > 0),
		latency_seconds REAL NOT NULL DEFAULT 0 CHECK(latency_seconds >= 0),
		price_per_byte REAL NOT NULL DEFAULT 0 CHECK(price_per_byte >= 0),
		bidirectional INTEGER NOT NULL DEFAULT 1,
		sharing_policy TEXT NOT NULL DEFAULT 'independent',
		max_concurrent_transfers INTEGER NOT NULL DEFAULT 0,
		metadata TEXT NOT NULL DEFAULT '{}',
		UNIQUE(topology_id, source_resource_id, target_resource_id)
	);
CREATE TABLE workflow_definitions (
		id TEXT PRIMARY KEY,
		external_id TEXT NOT NULL UNIQUE,
		name TEXT NOT NULL,
		namespace TEXT NOT NULL DEFAULT '',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
CREATE TABLE workflow_versions (
		id TEXT PRIMARY KEY,
		workflow_id TEXT NOT NULL REFERENCES workflow_definitions(id),
		version INTEGER NOT NULL,
		definition_hash TEXT NOT NULL,
		raw_definition TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL DEFAULT 'draft',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(workflow_id, version)
	);
CREATE TABLE activity_types (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		application TEXT NOT NULL DEFAULT '',
		default_image TEXT NOT NULL DEFAULT '',
		cpu_intensity REAL NOT NULL DEFAULT 0,
		memory_intensity REAL NOT NULL DEFAULT 0,
		io_intensity REAL NOT NULL DEFAULT 0,
		network_intensity REAL NOT NULL DEFAULT 0,
		metadata TEXT NOT NULL DEFAULT '{}'
	);
CREATE TABLE activity_definitions (
		id TEXT PRIMARY KEY,
		workflow_version_id TEXT NOT NULL REFERENCES workflow_versions(id),
		activity_type_id TEXT NOT NULL REFERENCES activity_types(id),
		external_id TEXT NOT NULL,
		name TEXT NOT NULL,
		kind TEXT NOT NULL CHECK(kind IN ('task', 'service', 'interactive')),
		capabilities TEXT NOT NULL,
		command_spec TEXT NOT NULL,
		resource_requirements TEXT NOT NULL,
		service_spec TEXT,
		simulation_spec TEXT,
		policy TEXT NOT NULL,
		priority INTEGER NOT NULL DEFAULT 0,
		metadata TEXT NOT NULL DEFAULT '{}',
		UNIQUE(workflow_version_id, external_id)
	);
CREATE TABLE activity_dependencies (
		activity_id TEXT NOT NULL REFERENCES activity_definitions(id),
		depends_on_activity_id TEXT NOT NULL REFERENCES activity_definitions(id),
		dependency_type TEXT NOT NULL DEFAULT 'control',
		PRIMARY KEY(activity_id, depends_on_activity_id),
		CHECK(activity_id <> depends_on_activity_id)
	);
CREATE TABLE activity_resource_profiles (
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
	);
CREATE TABLE schedule_plans (
		id TEXT PRIMARY KEY,
		workflow_version_id TEXT NOT NULL REFERENCES workflow_versions(id),
		execution_scope_id TEXT NOT NULL REFERENCES execution_scopes(id),
		network_topology_id TEXT REFERENCES network_topologies(id),
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
	);
CREATE TABLE schedule_plan_assignments (
		id TEXT PRIMARY KEY,
		schedule_plan_id TEXT NOT NULL REFERENCES schedule_plans(id),
		activity_id TEXT NOT NULL REFERENCES activity_definitions(id),
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
	);
CREATE TABLE execution_runs (
		id TEXT PRIMARY KEY,
		schedule_plan_id TEXT NOT NULL REFERENCES schedule_plans(id),
		mode TEXT NOT NULL CHECK(mode IN ('real', 'simulation', 'interactive')),
		seed INTEGER NOT NULL DEFAULT 0,
		status TEXT NOT NULL DEFAULT 'created',
		environment_snapshot_id TEXT NOT NULL DEFAULT '',
		started_at DATETIME,
		finished_at DATETIME,
		makespan_seconds REAL NOT NULL DEFAULT 0,
		cost REAL NOT NULL DEFAULT 0,
		failure_reason TEXT NOT NULL DEFAULT '',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
CREATE TABLE task_executions (
		id TEXT PRIMARY KEY,
		execution_run_id TEXT NOT NULL REFERENCES execution_runs(id),
		plan_assignment_id TEXT NOT NULL REFERENCES schedule_plan_assignments(id),
		activity_id TEXT NOT NULL REFERENCES activity_definitions(id),
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
	);
CREATE TABLE activity_lifecycle_events (
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
	);
CREATE TABLE activity_handles (
		id TEXT PRIMARY KEY,
		execution_run_id TEXT NOT NULL REFERENCES execution_runs(id),
		activity_id TEXT NOT NULL REFERENCES activity_definitions(id),
		resource_id TEXT NOT NULL REFERENCES resources(id),
		runtime_id TEXT NOT NULL,
		external_id TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL CHECK(status IN ('starting', 'running', 'completed', 'failed', 'stopped')),
		endpoints TEXT NOT NULL DEFAULT '[]',
		started_at REAL NOT NULL DEFAULT 0,
		finished_at REAL NOT NULL DEFAULT 0,
		exit_code INTEGER,
		failure TEXT NOT NULL DEFAULT '',
		artifacts TEXT NOT NULL DEFAULT 'null',
		metadata TEXT NOT NULL DEFAULT '{}',
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
CREATE TABLE data_objects (
		id TEXT PRIMARY KEY,
		workflow_version_id TEXT NOT NULL REFERENCES workflow_versions(id),
		producer_activity_id TEXT REFERENCES activity_definitions(id),
		logical_name TEXT NOT NULL,
		relative_path TEXT NOT NULL DEFAULT '',
		declared INTEGER NOT NULL DEFAULT 0,
		metadata TEXT NOT NULL DEFAULT '{}'
	);
CREATE TABLE data_object_instances (
		id TEXT PRIMARY KEY,
		data_object_id TEXT NOT NULL REFERENCES data_objects(id),
		execution_run_id TEXT NOT NULL REFERENCES execution_runs(id),
		producer_activity_id TEXT NOT NULL REFERENCES activity_definitions(id),
		attempt INTEGER NOT NULL DEFAULT 1,
		relative_path TEXT NOT NULL,
		size_bytes INTEGER NOT NULL DEFAULT 0 CHECK(size_bytes >= 0),
		checksum TEXT NOT NULL DEFAULT '',
		media_type TEXT NOT NULL DEFAULT '',
		discovered INTEGER NOT NULL DEFAULT 1,
		metadata TEXT NOT NULL DEFAULT '{}',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(execution_run_id, producer_activity_id, attempt, relative_path)
	);
CREATE TABLE storage_resources (
		id TEXT PRIMARY KEY,
		environment_version_id TEXT NOT NULL REFERENCES environment_versions(id),
		name TEXT NOT NULL,
		type TEXT NOT NULL CHECK(type IN ('local', 'pvc', 'nfs', 's3', 'lustre')),
		endpoint TEXT NOT NULL DEFAULT '',
		capacity_bytes INTEGER NOT NULL DEFAULT 0 CHECK(capacity_bytes >= 0),
		shared INTEGER NOT NULL DEFAULT 0,
		read_only INTEGER NOT NULL DEFAULT 0,
		configuration TEXT NOT NULL DEFAULT '{}',
		credential_reference TEXT NOT NULL DEFAULT '',
		metadata TEXT NOT NULL DEFAULT '{}',
		UNIQUE(environment_version_id, name)
	);
CREATE TABLE storage_runtime_bindings (
		storage_resource_id TEXT NOT NULL REFERENCES storage_resources(id) ON DELETE CASCADE,
		environment_version_id TEXT NOT NULL,
		runtime_id TEXT NOT NULL,
		is_default INTEGER NOT NULL DEFAULT 0,
		host_path TEXT NOT NULL DEFAULT '',
		container_path TEXT NOT NULL DEFAULT '/akoflow/data',
		read_only INTEGER NOT NULL DEFAULT 0,
		configuration TEXT NOT NULL DEFAULT '{}',
		PRIMARY KEY(storage_resource_id, runtime_id),
		FOREIGN KEY(runtime_id) REFERENCES environment_runtimes(id)
	);
CREATE UNIQUE INDEX one_default_storage_per_runtime
	ON storage_runtime_bindings(environment_version_id, runtime_id)
	WHERE is_default = 1;
CREATE TABLE data_locations (
		id TEXT PRIMARY KEY,
		data_object_instance_id TEXT NOT NULL REFERENCES data_object_instances(id),
		storage_resource_id TEXT REFERENCES storage_resources(id),
		resource_id TEXT REFERENCES resources(id),
		execution_run_id TEXT NOT NULL REFERENCES execution_runs(id),
		uri TEXT NOT NULL,
		available_at REAL NOT NULL DEFAULT 0,
		verified_at REAL NOT NULL DEFAULT 0,
		status TEXT NOT NULL DEFAULT 'ephemeral'
			CHECK(status IN ('ephemeral', 'staging', 'available', 'failed', 'deleted')),
		metadata TEXT NOT NULL DEFAULT '{}',
		UNIQUE(data_object_instance_id, uri)
	);
CREATE TABLE data_transfers (
		id TEXT PRIMARY KEY,
		execution_run_id TEXT NOT NULL REFERENCES execution_runs(id),
		data_object_id TEXT REFERENCES data_objects(id),
		data_object_instance_id TEXT REFERENCES data_object_instances(id),
		producer_activity_id TEXT REFERENCES activity_definitions(id),
		consumer_activity_id TEXT REFERENCES activity_definitions(id),
		source_resource_id TEXT NOT NULL REFERENCES resources(id),
		target_resource_id TEXT NOT NULL REFERENCES resources(id),
		bytes INTEGER NOT NULL DEFAULT 0,
		status TEXT NOT NULL,
		started_at REAL NOT NULL DEFAULT 0,
		finished_at REAL NOT NULL DEFAULT 0,
		duration_seconds REAL NOT NULL DEFAULT 0,
		cost REAL NOT NULL DEFAULT 0,
		metadata TEXT NOT NULL DEFAULT '{}'
	);
CREATE TABLE queue_jobs (
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
	);
CREATE UNIQUE INDEX queue_jobs_idempotency_idx
		ON queue_jobs(idempotency_key) WHERE idempotency_key IS NOT NULL AND idempotency_key <> '';
CREATE INDEX queue_jobs_available_idx
		ON queue_jobs(status, category, available_at, priority DESC);
CREATE TABLE domain_events (
		id TEXT PRIMARY KEY,
		event_type TEXT NOT NULL,
		aggregate_type TEXT NOT NULL,
		aggregate_id TEXT NOT NULL,
		payload BLOB NOT NULL DEFAULT X'',
		occurred_at DATETIME NOT NULL,
		metadata TEXT NOT NULL DEFAULT '{}'
	);
CREATE INDEX domain_events_aggregate_idx
		ON domain_events(aggregate_type, aggregate_id, occurred_at);
CREATE INDEX domain_events_type_idx
		ON domain_events(event_type, occurred_at);
CREATE TABLE schema_metadata (
	version INTEGER NOT NULL,
	applied_at DATETIME NOT NULL,
	checksum TEXT NOT NULL
);
