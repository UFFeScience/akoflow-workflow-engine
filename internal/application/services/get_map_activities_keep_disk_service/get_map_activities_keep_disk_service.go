package get_map_activities_keep_disk_service

import (
	"github.com/UFFeScience/akoflow/internal/application/ports"
	"github.com/UFFeScience/akoflow/internal/application/services/get_activity_dependencies_service"
	"github.com/UFFeScience/akoflow/internal/domain/workflow/activity"
	"github.com/UFFeScience/akoflow/internal/infrastructure/config"
)

type GetMapActivitiesKeepDiskService struct {
	namespace                     string
	workflowRepository            ports.WorkflowRepository
	activityRepository            ports.ActivityRepository
	storageRepository             ports.StorageRepository
	getActivityDependeciesService DependencyProvider
}

type DependencyProvider interface {
	GetActivityDependencies(int) workflow_activity_entity.MapActivityDependencies
}

func New() GetMapActivitiesKeepDiskService {
	dependencies := get_activity_dependencies_service.New()
	return GetMapActivitiesKeepDiskService{
		namespace:                     "akoflow",
		workflowRepository:            config.App().Repository.WorkflowRepository,
		activityRepository:            config.App().Repository.ActivityRepository,
		storageRepository:             config.App().Repository.StoragesRepository,
		getActivityDependeciesService: &dependencies,
	}
}

func NewWithDependencies(activities ports.ActivityRepository, dependencies DependencyProvider) GetMapActivitiesKeepDiskService {
	return GetMapActivitiesKeepDiskService{activityRepository: activities, getActivityDependeciesService: dependencies}
}

type MapActivitiesKeepDisk map[int]bool

func (c *GetMapActivitiesKeepDiskService) GetMapActivitiesKeepDisk(wfId int) (MapActivitiesKeepDisk, error) {

	activitiesByWorkflow, err := c.activityRepository.GetActivitiesByWorkflowIds([]int{wfId})
	activitiesDependencies := c.getActivityDependeciesService.GetActivityDependencies(wfId)

	if err != nil {
		return nil, err
	}

	activities := activitiesByWorkflow[wfId]

	mapActivitiesKeepDisk := make(map[int]bool)

	// default keep all disk
	for _, activity := range activities {
		mapActivitiesKeepDisk[activity.Id] = true
	}

	for _, dependencies := range activitiesDependencies {
		for _, dependency := range dependencies {
			if !dependency.KeepDisk {
				mapActivitiesKeepDisk[dependency.Id] = false
			}
		}
	}

	return mapActivitiesKeepDisk, nil
}
