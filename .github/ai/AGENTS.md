# Akoflow Agent Guide

This repository uses this file as the shared architecture reference for AI agents.

## Core Shape

- Go project with two binaries: client and server.
- The server is layered: HTTP handlers, services, runtimes, connectors, repositories and entities.
- `config.App()` is the dependency composition root.

## Primary Flows

- Client submits a workflow to the server.
- Server persists workflow and activities.
- Orchestrator schedules pending work.
- Worker dispatches activity execution.
- Runtime wrapper delegates to runtime service.
- Runtime service uses connectors and repositories to execute and monitor the job.

## Runtime Order For Debugging

1. `pkg/server/services/run_activity_in_cluster_service`
2. `pkg/server/runtimes/runtime.go`
3. `pkg/server/runtimes/<runtime>` wrapper
4. `pkg/server/runtimes/<runtime>/<runtime>_service`
5. connector package
6. repository layer

## Important Invariants

- Handlers should not talk directly to repositories.
- Runtime wrappers should stay thin.
- Runtime services contain the real execution logic.
- File persistence depends on the correct mount path inside the runtime.
- Singularity and HPC are the most sensitive to mount-path mismatches.

## Key References

- `AI_ARCHITECTURE.md`
- `AI_RUNTIME_EXECUTION_FLOW.md`
