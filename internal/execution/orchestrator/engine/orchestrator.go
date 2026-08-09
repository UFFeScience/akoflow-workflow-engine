package orchestrator

import (
	"time"

	"github.com/UFFeScience/akoflow/internal/application/services/get_pending_workflow_service"
	"github.com/UFFeScience/akoflow/internal/application/services/orchestrate_workflow_service"
	"github.com/UFFeScience/akoflow/internal/infrastructure/config"
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
