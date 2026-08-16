package hpc_runtime

import (
	workflow_activity_entity "github.com/UFFeScience/akoflow/internal/domain/workflow/activity"
	workflow_entity "github.com/UFFeScience/akoflow/internal/domain/workflow/definition"
	"testing"
)

func TestHPCRuntimeSimpleContract(t *testing.T) {
	h := &HpcRuntime{}
	if h.SetRuntimeName("hpc") != h || h.SetRuntimeType("slurm") != h {
		t.Fatal("setters")
	}
	if h.StartConnection() != nil || h.StopConnection() != nil {
		t.Fatal("connection")
	}
	if !h.DeleteJob(1, 2) || h.GetMetrics(1, 2) != "" || h.GetLogs(workflow_entity.Workflow{}, workflow_activity_entity.WorkflowActivities{}) != "" || h.GetStatus(1, 2) != "" {
		t.Fatal("contract")
	}
}
