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
	createStorageInDatabaseService StorageCreator
}

type StorageCreator interface{ CreateByWorkflow(int) error }

func New() *CreateWorkflowInDatabaseService {
	storageCreator := create_storage_in_database_service.New()
	return &CreateWorkflowInDatabaseService{
		namespace:                      "akoflow",
		workflowRepository:             config.App().Repository.WorkflowRepository,
		activityRepository:             config.App().Repository.ActivityRepository,
		storageRepository:              config.App().Repository.StoragesRepository,
		createStorageInDatabaseService: &storageCreator,
	}
}

func NewWithDependencies(namespace string, workflows ports.WorkflowRepository, activities ports.ActivityRepository, storageCreator StorageCreator) *CreateWorkflowInDatabaseService {
	return &CreateWorkflowInDatabaseService{namespace: namespace, workflowRepository: workflows, activityRepository: activities, createStorageInDatabaseService: storageCreator}
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
