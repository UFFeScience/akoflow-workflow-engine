package hpc_runtime_service

import (
	"encoding/base64"
	"strings"
	"testing"

	runtime_entity "github.com/UFFeScience/akoflow/internal/domain/resource/runtime"
	workflow_activity_entity "github.com/UFFeScience/akoflow/internal/domain/workflow/activity"
	workflow_entity "github.com/UFFeScience/akoflow/internal/domain/workflow/definition"
)

func TestMakeSBatchFromRuntimeTemplate(t *testing.T) {
	template := "#JOB_NAME#|#OUTPUT#|#ERROR#|#TIME#|#PARTITION#|#NTASKS#|#NODES#|#GPUS#|#CPUS_PER_GPU#|#MEM#|#COMMAND#"
	metadata := map[string]string{
		"SLURM_SBATCHTEMPLATE": base64.StdEncoding.EncodeToString([]byte(template)),
		"SLURM_TIME":           "10:00", "SLURM_QUEUE": "debug", "SLURM_NTASKS": "2", "SLURM_NODES": "1",
		"SLURM_GPUS": "0", "SLURM_CPUS_PER_GPU": "1", "SLURM_MEM": "2G", "SLURM_MOUNT_PATH": "/runtime",
	}
	r := runtime_entity.NewRuntime("slurm", 1, metadata, "", "")
	m := NewMakeSBatchHPCRuntimeActivityService().SetRuntime(*r).SetSingularityCommand("run image")
	if m.GetSingularityCommand() != "run image" || m.GetTemplateSbatch() != template {
		t.Fatal("builder accessors")
	}
	wf := workflow_entity.Workflow{Id: 7, Spec: workflow_entity.WorkflowSpec{MountPath: "/shared"}}
	a := workflow_activity_entity.WorkflowActivities{Id: 9}
	command := m.Handle(wf, a)
	if !strings.Contains(command, "base64 -d | bash") {
		t.Fatalf("unexpected command: %s", command)
	}

	wf.Spec.MountPath = ""
	if !strings.Contains(m.Handle(wf, a), "base64 -d | bash") {
		t.Fatal("runtime mount fallback")
	}
}

func TestMakeSBatchInvalidTemplate(t *testing.T) {
	r := runtime_entity.NewRuntime("slurm", 1, map[string]string{"SLURM_SBATCHTEMPLATE": "%%%"}, "", "")
	m := NewMakeSBatchHPCRuntimeActivityService().SetRuntime(*r)
	if m.GetTemplateSbatch() != "" {
		t.Fatal("invalid base64 accepted")
	}
}
