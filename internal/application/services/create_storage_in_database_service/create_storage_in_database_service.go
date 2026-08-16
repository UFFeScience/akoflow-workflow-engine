package create_storage_in_database_service

import (
	"github.com/UFFeScience/akoflow/internal/application/ports"
	"github.com/UFFeScience/akoflow/internal/application/services/get_map_activities_keep_disk_service"
	"github.com/UFFeScience/akoflow/internal/infrastructure/config"
)

type CreateStorageInDatabaseService struct {
	namespace                       string
	workflowRepository              ports.WorkflowRepository
	activityRepository              ports.ActivityRepository
	storageRepository               ports.StorageRepository
	getMapActivitiesKeepDiskService KeepDiskProvider
}

type KeepDiskProvider interface {
	GetMapActivitiesKeepDisk(int) (get_map_activities_keep_disk_service.MapActivitiesKeepDisk, error)
}

func New() CreateStorageInDatabaseService {
	keepDisk := get_map_activities_keep_disk_service.New()
	return CreateStorageInDatabaseService{
		namespace:                       "akoflow",
		workflowRepository:              config.App().Repository.WorkflowRepository,
		activityRepository:              config.App().Repository.ActivityRepository,
		storageRepository:               config.App().Repository.StoragesRepository,
		getMapActivitiesKeepDiskService: &keepDisk,
	}
}

func NewWithDependencies(namespace string, workflows ports.WorkflowRepository, storages ports.StorageRepository, keepDisk KeepDiskProvider) CreateStorageInDatabaseService {
	return CreateStorageInDatabaseService{namespace: namespace, workflowRepository: workflows, storageRepository: storages, getMapActivitiesKeepDiskService: keepDisk}
}

func (c *CreateStorageInDatabaseService) CreateByWorkflow(wfId int) error {

	workflow, err := c.workflowRepository.Find(wfId)
	if err != nil {
		return err
	}

	mapActivitiesKeepDisk, err := c.getMapActivitiesKeepDiskService.GetMapActivitiesKeepDisk(workflow.Id)
	if err != nil {
		return err
	}

	println("mapActivitiesKeepDisk", len(mapActivitiesKeepDisk))

	return c.storageRepository.Create(ports.CreateStorageParams{
		WorkflowID:            wfId,
		Namespace:             c.namespace,
		Status:                ports.StorageStatusCreated,
		ActivitiesKeepingDisk: mapActivitiesKeepDisk,
		StorageMountPath:      workflow.Spec.MountPath,
		StorageClass:          workflow.Spec.StoragePolicy.StorageClassName,
		StorageSize:           workflow.Spec.StoragePolicy.StorageSize,
	})

}
