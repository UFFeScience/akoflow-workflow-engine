package requests

import (
	"testing"

	"github.com/UFFeScience/akoflow/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestStructuredExecutableWorkflowConvertsToDomain(t *testing.T) {
	definition, err := (Workflow{Name: "portable-flow", Spec: WorkflowSpec{
		Namespace: "science",
		Activities: []WorkflowActivity{{
			Name: "compute",
			Command: domain.ActivityCommand{
				Entrypoint: "sh",
				Arguments:  []string{"-c", "echo ready"},
				Executable: &domain.ExecutableReference{
					Source: domain.ExecutableSource{
						Type:           domain.ExecutableSourceRemoteFile,
						Path:           "/apps/tool.sif",
						EnvironmentRef: "plafrim",
					},
					Delivery: domain.ExecutableDelivery{Strategy: domain.DeliveryUseInPlace},
				},
			},
		}},
	}}).Domain()
	require.NoError(t, err)
	activity := definition.Version.Activities[0]
	require.Equal(t, "/apps/tool.sif", activity.Command.Executable.Source.Path)
	require.Equal(t, domain.DeliveryUseInPlace, activity.Command.Executable.Delivery.Strategy)
}

func TestLegacyShapedWorkflowConvertsToNormalizedDomain(t *testing.T) {
	definition, err := (Workflow{Name: "science-flow", Spec: WorkflowSpec{
		Namespace: "science", Image: "busybox:1.36",
		Activities: []WorkflowActivity{
			{Name: "prepare", Run: "prepare", CPULimit: "100m", MemoryLimit: "16Mi"},
			{Name: "process", Run: "process", DependsOn: []string{"prepare"}},
		},
	}}).Domain()
	require.NoError(t, err)
	require.Equal(t, "science-flow-v1", definition.Version.ID)
	require.Equal(t, 0.1, definition.Version.Activities[0].Resources.CPU)
	require.Equal(t, int64(16*1024*1024), definition.Version.Activities[0].Resources.MemoryBytes)
	require.Equal(t, "process", definition.Version.Dependencies[0].ActivityID)
	require.Equal(t, "prepare", definition.Version.Dependencies[0].DependsOnActivityID)
}

func TestWorkflowRejectsUnknownDependency(t *testing.T) {
	_, err := (Workflow{Name: "flow", Spec: WorkflowSpec{
		Namespace: "science", Image: "busybox",
		Activities: []WorkflowActivity{{Name: "process", Run: "run", DependsOn: []string{"missing"}}},
	}}).Domain()
	require.ErrorContains(t, err, "unknown activity")
}
