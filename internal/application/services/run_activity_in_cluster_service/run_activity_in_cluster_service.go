package run_activity_in_cluster_service

import (
	"github.com/UFFeScience/akoflow/internal/execution/real/runtimes"
	"github.com/UFFeScience/akoflow/internal/infrastructure/config"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/activity_repository"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/workflow_repository"
)

type RunActivityInClusterService struct {
	namespace          string
	workflowRepository workflow_repository.IWorkflowRepository
	activityRepository activity_repository.IActivityRepository
}

func New() *RunActivityInClusterService {
	return &RunActivityInClusterService{
		namespace:          config.App().DefaultNamespace,
		workflowRepository: config.App().Repository.WorkflowRepository,
		activityRepository: config.App().Repository.ActivityRepository,
	}
}

func (r *RunActivityInClusterService) Run(activityID int) {

	wfa, err := r.activityRepository.Find(activityID)
	wf, _ := r.workflowRepository.Find(wfa.WorkflowId)

	if err != nil {
		config.App().Logger.Infof("WORKER: Activity not found %d", activityID)
		return
	}

	assignment, err := r.activityRepository.GetActivityScheduleByActivityId(activityID)
	if err != nil || assignment.ResourceID == "" {
		config.App().Logger.Infof("WORKER: Activity %d has no resource assignment", activityID)
		return
	}
	resource, err := config.App().Repository.ResourceRepository.FindByID(assignment.ResourceID)
	if err != nil || resource == nil {
		config.App().Logger.Infof("WORKER: Resource %s not found for activity %d", assignment.ResourceID, activityID)
		return
	}
	runtimeID := resource.RuntimeID

	workflowId := wf.GetId()
	workflowActivityId := wfa.GetId()

	runtimes.
		GetRuntimeInstance(runtimeID).
		ApplyJob(workflowId, workflowActivityId)

	config.App().Logger.Infof("WORKER: Activity %d started", activityID)

}
