# Akoflow Architecture

Akoflow is a control plane for planning and running scientific activities on heterogeneous environments. The same immutable workflow and schedule plan can be executed by a real runtime, a simulator or an interactive runtime.

## Boundaries

```text
API / CLI
    -> persistent command queue
        -> execution supervisor
            -> activity execution service
                -> runtime registry
                    -> local | kubernetes | slurm | simulation
```

The core records have distinct responsibilities:

- `WorkflowDefinition`: immutable activity graph.
- `EnvironmentVersion`: immutable execution platform description.
- `SchedulePlan`: planner prediction and resource assignment.
- `ExecutionRun`: one attempt to realize a plan in a selected mode.
- `ActivityHandle`: provider-independent identity of a started component.
- `TaskExecution`: measured or simulated observation.
- `ExecutionTrace`: comparison between predicted and observed results.

## Package Direction

```text
api -> application -> domain
runtime -> application ports -> domain
infrastructure -> application ports -> domain
cmd/server -> all packages (composition only)
```

Domain code must not import API, database, provider SDKs or runtime adapters.

## Repository Aggregates

- Workflow repository: definition, versions, activity types, activities and dependencies.
- Environment repository: environment, versions, connections and runtime capabilities.
- Resource inventory: resources, snapshots and network links.
- Planning repository: plans and assignments.
- Execution repository: runs, tasks, handles, lifecycle events and transfers.
- Queue repository: commands, leases, attempts and retry state.

Repositories receive a shared `*sql.DB`; they do not open a new database connection for every method.

## Execution Modes

`real`, `simulation` and `interactive` use the same `RuntimeAdapter` lifecycle:

```text
Start(ActivityExecutionContext) -> ActivityHandle
Inspect(ActivityHandle)         -> ActivityHandle
Stop(ActivityHandle)
```

An adapter declares all modes it supports. The registry resolves by `(mode, runtimeID)` and contains no provider-specific branching.

## Asynchronous Control

API writes return after a durable command is accepted. Queue leases permit recovery after process interruption. Handlers can run more than once and must therefore use stable aggregate IDs and idempotent persistence.
