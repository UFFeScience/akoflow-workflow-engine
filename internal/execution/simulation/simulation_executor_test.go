package execution

import (
	"context"
	"testing"

	"github.com/UFFeScience/akoflow/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestSimulationExecutesFrozenPlanWithRealNetworkModel(t *testing.T) {
	workflow := domain.WorkflowVersion{
		ID: "wf-v1",
		Activities: []domain.Activity{
			{ID: "a", ActivityTypeID: "type-a", CPURequired: 1, MemoryRequiredBytes: 1},
			{ID: "b", ActivityTypeID: "type-b", CPURequired: 1, MemoryRequiredBytes: 1},
		},
		Dependencies:     []domain.ActivityDependency{{ActivityID: "b", DependsOnActivityID: "a"}},
		DataDependencies: []domain.ActivityDataDependency{{ProducerActivityID: "a", ConsumerActivityID: "b", SizeBytes: 1_000_000}},
	}
	resources := []domain.Resource{
		{ID: "edge", EnvironmentVersionID: "env-v1", RuntimeID: "edge-runtime", CPUCapacity: 1, MemoryBytes: 1, ComputeSpeedup: 1, Schedulable: true},
		{ID: "cloud", EnvironmentVersionID: "env-v1", RuntimeID: "cloud-runtime", CPUCapacity: 1, MemoryBytes: 1, ComputeSpeedup: 1, Schedulable: true},
	}
	plan := domain.SchedulePlan{
		ID: "plan", WorkflowVersionID: "wf-v1", EnvironmentVersionID: "env-v1",
		DeadlineSeconds: 10,
		Assignments: []domain.PlanAssignment{
			{ID: "pa", ActivityID: "a", ResourceID: "edge", OrderOnResource: 0},
			{ID: "pb", ActivityID: "b", ResourceID: "cloud", OrderOnResource: 0},
		},
	}
	profiles := []domain.ActivityResourceProfile{
		{ActivityTypeID: "type-a", ResourceID: "edge", RuntimeSeconds: 2},
		{ActivityTypeID: "type-b", ResourceID: "cloud", RuntimeSeconds: 3},
	}
	trace, err := NewSimulationExecutor().Execute(context.Background(), Request{
		Run:  domain.ExecutionRun{ID: "run", Mode: domain.ExecutionModeSimulation},
		Plan: plan, Workflow: workflow, Resources: resources, ActivityProfiles: profiles,
		NetworkLinks: []domain.NetworkLink{{
			SourceResourceID: "edge", TargetResourceID: "cloud", Bidirectional: true,
			BandwidthBitsPerSecond: 8_000_000, LatencySeconds: 0.1,
		}},
	})
	require.NoError(t, err)
	require.Len(t, trace.Transfers, 1)
	require.InDelta(t, 1.1, trace.Transfers[0].DurationSeconds, 1e-9)
	require.InDelta(t, 6.1, trace.Executed.MakespanSeconds, 1e-9)
	require.True(t, trace.Executed.Feasible)
}

func TestSimulationRejectsPlanThatDoesNotCoverWorkflow(t *testing.T) {
	workflow := domain.WorkflowVersion{ID: "wf", Activities: []domain.Activity{{ID: "a"}}}
	_, err := NewSimulationExecutor().Execute(context.Background(), Request{
		Run:  domain.ExecutionRun{Mode: domain.ExecutionModeSimulation},
		Plan: domain.SchedulePlan{WorkflowVersionID: "wf"}, Workflow: workflow,
	})
	require.ErrorContains(t, err, "covers 0 of 1 activities")
}
