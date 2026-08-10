package garbage_collector_remove_storage_service

import (
	"fmt"
	"strconv"

	"github.com/UFFeScience/akoflow/internal/application/ports"
	"github.com/UFFeScience/akoflow/internal/application/services/get_activity_dependencies_service"
	"github.com/UFFeScience/akoflow/internal/application/services/get_pending_storage_service"
	"github.com/UFFeScience/akoflow/internal/application/services/get_pending_workflow_service"
	"github.com/UFFeScience/akoflow/internal/application/services/get_workflow_by_status_service"
	"github.com/UFFeScience/akoflow/internal/domain/workflow/activity"
	"github.com/UFFeScience/akoflow/internal/infrastructure/config"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/runtime_repository"
	"github.com/UFFeScience/akoflow/internal/infrastructure/kubernetes/connector"
)

type GarbageCollectorRemoveStorageService struct {
	namespace          string
	workflowRepository ports.WorkflowRepository
	activityRepository ports.ActivityRepository
	storageRepository  ports.StorageRepository
	runtimeRepository  runtime_repository.IRuntimeRepository

	getActivityDependeciesService get_activity_dependencies_service.GetActivityDependenciesService
	getPendingWorkflowService     get_pending_workflow_service.GetPendingWorkflowService
	getWorkflowByStatusService    get_workflow_by_status_service.GetWorkflowByStatusService
	getPendingStorageService      get_pending_storage_service.GetPendingStorageService
	connector                     connector_k8s.IConnector
}

func New() GarbageCollectorRemoveStorageService {
	return GarbageCollectorRemoveStorageService{
		namespace:          "akoflow",
		workflowRepository: config.App().Repository.WorkflowRepository,
		activityRepository: config.App().Repository.ActivityRepository,
		storageRepository:  config.App().Repository.StoragesRepository,
		runtimeRepository:  config.App().Repository.RuntimeRepository,

		connector: config.App().Connector.K8sConnector,

		getActivityDependeciesService: get_activity_dependencies_service.New(),
		getPendingWorkflowService:     get_pending_workflow_service.New(),
		getWorkflowByStatusService:    get_workflow_by_status_service.New(),
		getPendingStorageService:      get_pending_storage_service.New(),
	}
}

type MapActivitiesKeepDisk map[int]bool

func (c *GarbageCollectorRemoveStorageService) RemoveStorages() {
	fmt.Printf("Garbage Collector Disabled")
}

func (c *GarbageCollectorRemoveStorageService) RemoveStoragesDeprecated() {
	wfaFinished, _ := c.getPendingStorageService.GetPendingStorages()

	wfaPreactivities, _ := c.activityRepository.GetPreactivitiesCompleted()

	for _, activity := range wfaFinished {
		if activity.KeepDisk {
			c.handleKeepDisk(activity)
		} else {
			c.removeResource(activity)
		}
	}

	for _, preactivity := range wfaPreactivities {
		wfa, _ := c.activityRepository.Find(preactivity.ActivityId)

		runtime, _ := c.runtimeRepository.GetByName(wfa.GetRuntimeId())
		if runtime == nil {
			println("Runtime not found")
			return
		}

		podJob, _ := c.connector.Pod(runtime).GetPodByJob(c.namespace, "preactivity-"+strconv.Itoa(preactivity.ActivityId))
		podNameJob, _ := podJob.GetPodName()

		_ = c.connector.Pod(runtime).DeletePod(c.namespace, podNameJob)

		preactivity.Status = ports.ActivityStatusCompleted
		_ = c.activityRepository.UpdatePreActivity(preactivity.ActivityId, preactivity)

	}

}

func (c *GarbageCollectorRemoveStorageService) removeResource(activity workflow_activity_entity.WorkflowActivities) {

	runtime, _ := c.runtimeRepository.GetByName(activity.GetRuntimeId())
	if runtime == nil {
		println("Runtime not found")
		return
	}

	_ = c.connector.PersistentVolumeClain(runtime).DeletePersistentVolumeClaim(activity.GetVolumeName(), c.namespace)

	podJob, _ := c.connector.Pod(runtime).GetPodByJob(c.namespace, activity.GetNameJob())
	podNameJob, _ := podJob.GetPodName()

	_ = c.connector.Pod(runtime).DeletePod(c.namespace, podNameJob)

	_ = c.storageRepository.Update(ports.UpdateStorageParams{
		PVCName:    activity.GetVolumeName(),
		Status:     ports.StorageStatusCompleted,
		ActivityID: activity.Id,
	})

	_ = c.storageRepository.UpdateDetached(activity.Id)
}

func (c *GarbageCollectorRemoveStorageService) handleKeepDisk(activity workflow_activity_entity.WorkflowActivities) {
	err := c.storageRepository.Update(ports.UpdateStorageParams{
		PVCName:    activity.GetVolumeName(),
		Status:     ports.StorageStatusCompleted,
		ActivityID: activity.Id,
	})
	if err != nil {
		println("Error updating storage")
		return
	}
}
