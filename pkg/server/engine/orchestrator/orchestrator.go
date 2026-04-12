package orchestrator

import (
	"time"

	"github.com/ovvesley/akoflow/pkg/server/config"
	"github.com/ovvesley/akoflow/pkg/server/services/get_pending_workflow_service"
	"github.com/ovvesley/akoflow/pkg/server/services/orchestrate_workflow_service"
)

const TimeToUpdateSeconds = 1

func StartOrchestrator() {

	for {
		handleOrchestrator()
		time.Sleep(TimeToUpdateSeconds * time.Second)
		config.App().Logger.Info("Orchestrator is running")
	}

}

func handleOrchestrator() {
	getPendingWorkflowService := get_pending_workflow_service.New()
	workflows, err := getPendingWorkflowService.GetPendingWorkflows()
	if err != nil {
		config.App().Logger.Error("failed to load pending workflows:", err)
		return
	}

	dispatchToWorkerActivityService := orchestrate_workflow_service.New()
	dispatchToWorkerActivityService.Orchestrate(workflows)

}
