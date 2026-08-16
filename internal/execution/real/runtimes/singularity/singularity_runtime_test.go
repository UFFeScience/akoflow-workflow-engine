package singularity_runtime

import (
	workflow_activity_entity "github.com/UFFeScience/akoflow/internal/domain/workflow/activity"
	workflow_entity "github.com/UFFeScience/akoflow/internal/domain/workflow/definition"
	"testing"
)

func TestSingularityRuntimePassiveContract(t *testing.T) {
	runtime := NewSingularityRuntime()
	if runtime == nil || runtime.StartConnection() != nil || runtime.StopConnection() != nil || !runtime.DeleteJob(1, 2) || runtime.GetMetrics(1, 2) != "" || runtime.GetLogs(workflow_entity.Workflow{}, workflow_activity_entity.WorkflowActivities{}) != "" || runtime.GetStatus(1, 2) != "" || !runtime.HealthCheck() {
		t.Fatal("contract")
	}
}
