package list_workflows_api_service

import (
	"github.com/UFFeScience/akoflow/internal/api/mapper/mapper_engine_api"
	"github.com/UFFeScience/akoflow/internal/api/requests"
	"github.com/UFFeScience/akoflow/internal/application/utils"
	"github.com/UFFeScience/akoflow/internal/infrastructure/config"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/activity_repository"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/workflow_repository"
)

type ListWorkflowsApiService struct {
	workflowRepository workflow_repository.IWorkflowRepository
	activityRepository activity_repository.IActivityRepository
}

func New() *ListWorkflowsApiService {
	return &ListWorkflowsApiService{
		workflowRepository: config.App().Repository.WorkflowRepository,
		activityRepository: config.App().Repository.ActivityRepository,
	}
}

func (h *ListWorkflowsApiService) ListAllWorkflows() ([]types_api.ApiWorkflowType, error) {
	workflowsEngine, err := h.workflowRepository.ListAllWorkflows(nil)

	if err != nil {
		return nil, err
	}

	ids := utils.GetIds(workflowsEngine)
	mapWfActivities, err := h.activityRepository.GetActivitiesByWorkflowIds(ids)

	if err != nil {
		return nil, err
	}

	workflowsEngine = utils.HydrateWorkflows(workflowsEngine, mapWfActivities)

	workflowApi := mapper_engine_api.MapEngineWorkflowEntityToApiWorkflowEntityList(workflowsEngine)

	return workflowApi, nil

}
