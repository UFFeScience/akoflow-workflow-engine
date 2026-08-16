package docker_runtime

import (
	workflow_activity_entity "github.com/UFFeScience/akoflow/internal/domain/workflow/activity"
	workflow_entity "github.com/UFFeScience/akoflow/internal/domain/workflow/definition"
	"testing"
)

func TestDockerRuntimeContract(t *testing.T) {
	for _, runtime := range []*DockerRuntime{New(), NewDockerRuntime()} {
		if runtime == nil {
			t.Fatal("nil")
		}
		if runtime.StartConnection() != nil || runtime.StopConnection() != nil || !runtime.ApplyJob(1, 2) || !runtime.DeleteJob(1, 2) || runtime.GetMetrics(1, 2) != "" || runtime.GetLogs(workflow_entity.Workflow{}, workflow_activity_entity.WorkflowActivities{}) != "" || runtime.GetStatus(1, 2) != "" || !runtime.VerifyActivitiesWasFinished(workflow_entity.Workflow{}) || !runtime.HealthCheck() {
			t.Fatal("contract failed")
		}
	}
}
