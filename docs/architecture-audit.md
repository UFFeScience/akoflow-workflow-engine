# Architecture and test audit

## Current boundaries

- `cmd/akoflow`, `cmd/server` and `cmd/worker` are composition roots.
- `internal/domain` owns the canonical workflow, environment, resource,
  planning and execution concepts.
- `internal/application/ports` owns provider-independent contracts.
- `internal/infrastructure` owns persistence and external-system adapters.
- `internal/execution` owns orchestration, lifecycle and execution engines.
- `internal/api` owns HTTP transport.
- `internal/cli` owns CLI commands, use cases, API adapters and assets.

## Remaining coupling to remove

1. Several application services still resolve repositories through the global
   application container. They must receive ports through constructors.
2. Kubernetes, Slurm, SSH and local-process adapters need injectable clients
   or command runners before deterministic failure-path tests are possible.
3. HTTP handlers that still create services directly need dependency structs,
   following the workflow-engine handler already migrated.
4. The old YAML workflow DTOs under `domain/workflow` mix transport parsing
   with domain behavior and should be separated into API request mappers.
5. Repository constructors still open a process-wide SQLite path. Connection
   factories must be injected so repository tests can use isolated databases.
6. The server process still starts worker, monitor, orchestrator, health check
   and garbage collector together. `cmd/worker` now exists, but server startup
   needs an explicit deployment-mode decision.

## Coverage baseline

The first complete profile after the structural migration reported **8.9%**
statement coverage. This is a real whole-tree measurement; packages without
tests are included as zero. Reaching 100% requires the dependency seams above,
not exclusions or generated coverage.

Every migrated package should cover success, dependency failure, malformed
input, empty input, cancellation, boundary values and repeated/idempotent calls
where applicable.
