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
	getMapActivitiesKeepDiskService get_map_activities_keep_disk_service.GetMapActivitiesKeepDiskService
}

func New() CreateStorageInDatabaseService {
	return CreateStorageInDatabaseService{
		namespace:                       "akoflow",
		workflowRepository:              config.App().Repository.WorkflowRepository,
		activityRepository:              config.App().Repository.ActivityRepository,
		storageRepository:               config.App().Repository.StoragesRepository,
		getMapActivitiesKeepDiskService: get_map_activities_keep_disk_service.New(),
	}
}

func (c *CreateStorageInDatabaseService) CreateByWorkflow(wfId int) error {

	workflow, err := c.workflowRepository.Find(wfId)
	if err != nil {
		return err
	}

	mapActivitiesKeepDisk, _ := c.getMapActivitiesKeepDiskService.GetMapActivitiesKeepDisk(workflow.Id)

	println("mapActivitiesKeepDisk", len(mapActivitiesKeepDisk))

	err = c.storageRepository.Create(ports.CreateStorageParams{
		WorkflowID:            wfId,
		Namespace:             c.namespace,
		Status:                ports.StorageStatusCreated,
		ActivitiesKeepingDisk: mapActivitiesKeepDisk,
		StorageMountPath:      workflow.Spec.MountPath,
		StorageClass:          workflow.Spec.StoragePolicy.StorageClassName,
		StorageSize:           workflow.Spec.StoragePolicy.StorageSize,
	})

	return nil

}
