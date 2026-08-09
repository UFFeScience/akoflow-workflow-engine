package kubernetes_runtime_service

import (
	"github.com/UFFeScience/akoflow/internal/domain/workflow/activity"
	"github.com/UFFeScience/akoflow/internal/domain/workflow/definition"
)

type IWorkerRunActivityService interface {
	SetWorkflow(workflow workflow_entity.Workflow) IWorkerRunActivityService
	SetWorkflowActivity(workflowActivity workflow_activity_entity.WorkflowActivities) IWorkerRunActivityService

	GetWorkflow() workflow_entity.Workflow
	GetWorkflowActivity() workflow_activity_entity.WorkflowActivities

	ApplyJob(activityID int) bool
	HandleResourceToRunJob(activityID int) bool
}

func ModeRunActivityService(mode string) IWorkerRunActivityService {
	modeMap := map[string]IWorkerRunActivityService{
		workflow_entity.MODE_STANDALONE:  NewWorkerRunActivityStandaloneService(),
		workflow_entity.MODE_DISTRIBUTED: NewWorkerRunActivityDistributedService(),
	}
	return modeMap[mode]
}
