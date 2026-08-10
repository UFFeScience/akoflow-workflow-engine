package list_workflows_api_service

import (
	"github.com/UFFeScience/akoflow/internal/api/mapper/mapper_engine_api"
	"github.com/UFFeScience/akoflow/internal/api/requests"
	"github.com/UFFeScience/akoflow/internal/application/ports"
	"github.com/UFFeScience/akoflow/internal/application/utils"
	"github.com/UFFeScience/akoflow/internal/infrastructure/config"
)

type ListWorkflowsApiService struct {
	workflowRepository ports.WorkflowRepository
	activityRepository ports.ActivityRepository
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
