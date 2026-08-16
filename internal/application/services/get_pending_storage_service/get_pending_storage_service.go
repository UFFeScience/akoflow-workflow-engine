package get_pending_storage_service

import (
	"github.com/UFFeScience/akoflow/internal/application/ports"
	"github.com/UFFeScience/akoflow/internal/application/services/get_activity_dependencies_service"
	"github.com/UFFeScience/akoflow/internal/application/services/get_workflow_by_status_service"
	"github.com/UFFeScience/akoflow/internal/domain/workflow/activity"
	"github.com/UFFeScience/akoflow/internal/infrastructure/config"
	"github.com/UFFeScience/akoflow/internal/infrastructure/kubernetes/connector"
)

type GetPendingStorageService struct {
	namespace                  string
	workflowRepository         ports.WorkflowRepository
	activityRepository         ports.ActivityRepository
	storageRepository          ports.StorageRepository
	getWorkflowByStatusService ActivityStatusSelector
	getActivityDependencies    DependencyProvider

	connector connector_k8s.IConnector
}

type ActivityStatusSelector interface {
	GetActivitiesByStatuses([]workflow_activity_entity.WorkflowActivities, int) []workflow_activity_entity.WorkflowActivities
}

type DependencyProvider interface {
	GetActivityDependenciesByWorkflow(int) workflow_activity_entity.MapActivityDependencies
}

func New() GetPendingStorageService {
	status := get_workflow_by_status_service.New()
	dependencies := get_activity_dependencies_service.New()
	return GetPendingStorageService{
		namespace:          "akoflow",
		workflowRepository: config.App().Repository.WorkflowRepository,
		activityRepository: config.App().Repository.ActivityRepository,
		storageRepository:  config.App().Repository.StoragesRepository,

		getWorkflowByStatusService: &status,
		getActivityDependencies:    &dependencies,

		connector: connector_k8s.New(),
	}
}

func NewWithDependencies(namespace string, activities ports.ActivityRepository, storages ports.StorageRepository, status ActivityStatusSelector, dependencies DependencyProvider) GetPendingStorageService {
	return GetPendingStorageService{namespace: namespace, activityRepository: activities, storageRepository: storages, getWorkflowByStatusService: status, getActivityDependencies: dependencies}
}

func (g *GetPendingStorageService) GetPendingStorages() ([]workflow_activity_entity.WorkflowActivities, error) {
	storages, err := g.storageRepository.GetCreatedStorages(g.namespace)
	if err != nil {
		return nil, err
	}

	mapWorkflowByStorage := make(map[int][]ports.Storage)
	mapActivityByStorage := make(map[int]ports.Storage)
	workflowsIds := make([]int, 0)

	for _, storage := range storages {
		mapWorkflowByStorage[storage.WorkflowID] = append(mapWorkflowByStorage[storage.WorkflowID], storage)
		mapActivityByStorage[storage.ActivityID] = storage
	}

	for key := range mapWorkflowByStorage {
		workflowsIds = append(workflowsIds, key)
	}

	wfActivities, err := g.activityRepository.GetActivitiesByWorkflowIds(workflowsIds)
	if err != nil {
		return nil, err
	}

	allActivities := make([]workflow_activity_entity.WorkflowActivities, 0)

	for wfaId, activities := range wfActivities {
		allActivities = append(allActivities, g.handleWorkflowActivities(wfaId, activities, mapWorkflowByStorage[wfaId])...)
	}

	allActivitiesFiltered := make([]workflow_activity_entity.WorkflowActivities, 0)
	for _, activity := range allActivities {
		storage := mapActivityByStorage[activity.Id]

		if storage.ID == 0 {
			continue
		}

		if storage.KeepStorageAfterFinish == 1 {
			activity.KeepDisk = true
		}

		if storage.Status == ports.StorageStatusCompleted {
			continue
		}

		allActivitiesFiltered = append(allActivitiesFiltered, activity)

	}

	return allActivitiesFiltered, nil
}

func (g *GetPendingStorageService) handleWorkflowActivities(wfId int, activities []workflow_activity_entity.WorkflowActivities, storages []ports.Storage) []workflow_activity_entity.WorkflowActivities {

	wfaFinisheds := g.getWorkflowByStatusService.GetActivitiesByStatuses(activities, ports.ActivityStatusFinished)
	wfaRunning := g.getWorkflowByStatusService.GetActivitiesByStatuses(activities, ports.ActivityStatusRunning)

	wfaStarted := append(wfaFinisheds, wfaRunning...)

	mapWfaToBeDeleted := make(map[int]workflow_activity_entity.WorkflowActivities)
	for _, activity := range wfaStarted {
		mapWfaToBeDeleted[activity.Id] = activity
	}

	allDependencies := g.getActivityDependencies.GetActivityDependenciesByWorkflow(wfId)

	activitiesToDelete := make([]workflow_activity_entity.WorkflowActivities, 0)

	workflowThatNeedByActivity := make(map[int][]int)
	for activityId, dependencies := range allDependencies {
		for _, dependency := range dependencies {
			workflowThatNeedByActivity[dependency.Id] = append(workflowThatNeedByActivity[dependency.Id], activityId)
		}
	}

	for _, wfaFinished := range wfaFinisheds {
		activitiesShouldStarted := workflowThatNeedByActivity[wfaFinished.Id]
		activitiesStarted := make([]workflow_activity_entity.WorkflowActivities, 0)
		for _, activityShouldStarted := range activitiesShouldStarted {
			if _, ok := mapWfaToBeDeleted[activityShouldStarted]; ok {
				activitiesStarted = append(activitiesStarted, mapWfaToBeDeleted[activityShouldStarted])
			}
		}

		if len(activitiesStarted) == len(activitiesShouldStarted) {
			activitiesToDelete = append(activitiesToDelete, wfaFinished)
		}

	}

	return activitiesToDelete

}
