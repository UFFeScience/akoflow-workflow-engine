package get_pending_workflow_service

import (
	"errors"
	"github.com/UFFeScience/akoflow/internal/application/utils"
	"github.com/UFFeScience/akoflow/internal/domain/workflow/definition"
	"github.com/UFFeScience/akoflow/internal/infrastructure/config"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/activity_repository"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/workflow_repository"
)

type GetPendingWorkflowService struct {
	namespace          string
	workflowRepository workflow_repository.IWorkflowRepository
	activityRepository activity_repository.IActivityRepository
}

func New() GetPendingWorkflowService {
	return GetPendingWorkflowService{
		namespace:          config.App().DefaultNamespace,
		workflowRepository: config.App().Repository.WorkflowRepository,
		activityRepository: config.App().Repository.ActivityRepository,
	}
}

func (g *GetPendingWorkflowService) GetPendingWorkflows() ([]workflow_entity.Workflow, error) {
	workflows, err := g.retriveWorkflowsOnDatabase()
	if err != nil {
		return nil, err
	}

	return workflows, nil
}

func (g *GetPendingWorkflowService) retriveWorkflowsOnDatabase() ([]workflow_entity.Workflow, error) {
	if g.workflowRepository == nil {
		return nil, errors.New("workflow repository is not initialized")
	}

	if g.activityRepository == nil {
		return nil, errors.New("activity repository is not initialized")
	}

	workflows, err := g.workflowRepository.GetPendingWorkflows(g.namespace)
	if err != nil {
		return nil, err
	}

	ids := utils.GetIds(workflows)
	mapWfActivities, err := g.activityRepository.GetActivitiesByWorkflowIds(ids)

	if err != nil {
		return nil, err
	}

	workflows = utils.HydrateWorkflows(workflows, mapWfActivities)
	return workflows, nil
}
