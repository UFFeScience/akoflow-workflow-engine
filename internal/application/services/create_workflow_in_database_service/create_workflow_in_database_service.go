package create_workflow_in_database_service

import (
	"github.com/UFFeScience/akoflow/internal/application/ports"
	"github.com/UFFeScience/akoflow/internal/application/services/create_storage_in_database_service"
	"github.com/UFFeScience/akoflow/internal/domain/workflow/definition"
	"github.com/UFFeScience/akoflow/internal/infrastructure/config"
)

type CreateWorkflowInDatabaseService struct {
	namespace                      string
	workflowRepository             ports.WorkflowRepository
	activityRepository             ports.ActivityRepository
	storageRepository              ports.StorageRepository
	createStorageInDatabaseService create_storage_in_database_service.CreateStorageInDatabaseService
}

func New() *CreateWorkflowInDatabaseService {
	return &CreateWorkflowInDatabaseService{
		namespace:                      "akoflow",
		workflowRepository:             config.App().Repository.WorkflowRepository,
		activityRepository:             config.App().Repository.ActivityRepository,
		storageRepository:              config.App().Repository.StoragesRepository,
		createStorageInDatabaseService: create_storage_in_database_service.New(),
	}
}

func (c *CreateWorkflowInDatabaseService) Create(workflow workflow_entity.Workflow) (int, error) {
	workflowId, err := c.workflowRepository.Create(c.namespace, workflow)
	if err != nil {
		return 0, err
	}
	workflowDb, err := c.workflowRepository.Find(workflowId)
	if err != nil {
		return 0, err
	}

	err = c.activityRepository.Create(c.namespace, workflowDb, workflow.Spec.Activities)
	if err != nil {
		return 0, err
	}

	err = c.createStorageInDatabaseService.CreateByWorkflow(workflowId)
	if err != nil {
		return 0, err
	}

	return workflowId, nil

}
