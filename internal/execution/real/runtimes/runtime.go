package runtimes

import (
	"fmt"
	"strings"

	"github.com/UFFeScience/akoflow/internal/domain/workflow/activity"
	"github.com/UFFeScience/akoflow/internal/domain/workflow/definition"
	"github.com/UFFeScience/akoflow/internal/execution/real/runtimes/docker"
	"github.com/UFFeScience/akoflow/internal/execution/real/runtimes/local"
	"github.com/UFFeScience/akoflow/internal/execution/real/runtimes/singularity"
	"github.com/UFFeScience/akoflow/internal/infrastructure/config"
	"github.com/UFFeScience/akoflow/internal/infrastructure/kubernetes/runtime"
	"github.com/UFFeScience/akoflow/internal/infrastructure/slurm/runtime"
)

const RUNTIME_K8S = "k8s"
const RUNTIME_DOCKER = "docker"
const RUNTIME_SINGULARITY = "singularity"
const RUNTIME_HPC = "hpc"
const RUNTIME_LOCAL = "local"

type IRuntime interface {
	StartConnection() error
	StopConnection() error

	ApplyJob(workflowID int, activityID int) bool
	DeleteJob(workflowID int, activityID int) bool

	GetMetrics(workflowID int, activityID int) string
	GetLogs(workflow workflow_entity.Workflow, workflowActivity workflow_activity_entity.WorkflowActivities) string

	GetStatus(workflowID int, activityID int) string

	VerifyActivitiesWasFinished(workflow workflow_entity.Workflow) bool

	HealthCheck() bool
}

func normalizeRuntime(runtime string) string {
	// if runtime start with k8s or k8s://, set it to k8s

	if strings.HasPrefix(runtime, "k8s") {
		return RUNTIME_K8S
	}

	if strings.HasPrefix(runtime, "hpc") {
		return RUNTIME_HPC
	}

	return runtime

}

func GetRuntimeInstance(runtimeName string) IRuntime {

	runtime := normalizeRuntime(runtimeName)

	modeMap := map[string]IRuntime{
		RUNTIME_DOCKER: docker_runtime.NewDockerRuntime(),

		RUNTIME_K8S: kubernetes_runtime.NewKubernetesRuntime().
			SetRuntimeType(RUNTIME_K8S).
			SetRuntimeName(runtimeName),

		RUNTIME_SINGULARITY: singularity_runtime.NewSingularityRuntime(),

		RUNTIME_HPC: hpc_runtime.NewHpcRuntime().
			SetRuntimeType(RUNTIME_HPC).
			SetRuntimeName(runtimeName),

		RUNTIME_LOCAL: local_runtime.NewLocalRuntime().
			SetRuntimeType(RUNTIME_LOCAL).
			SetRuntimeName(runtimeName),
	}
	if modeMap[runtime] == nil {
		config.App().Logger.Error(fmt.Sprintf("Runtime not found: %s", runtimeName))
	}
	return modeMap[runtime]
}
