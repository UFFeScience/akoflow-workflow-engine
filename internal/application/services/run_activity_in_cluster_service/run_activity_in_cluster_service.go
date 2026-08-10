package run_activity_in_cluster_service

import (
	"fmt"

	"github.com/UFFeScience/akoflow/internal/application/ports"
	"github.com/UFFeScience/akoflow/internal/execution/real/runtimes"
	"github.com/UFFeScience/akoflow/internal/infrastructure/config"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/resource_repository"
)

type RunActivityInClusterService struct {
	namespace          string
	workflowRepository ports.WorkflowRepository
	activityRepository ports.ActivityRepository
	resourceRepository resource_repository.IRepository
	runtimeResolver    RuntimeResolver
}

type RuntimeExecutor interface {
	ApplyJob(workflowID int, activityID int) bool
}

type RuntimeResolver func(runtimeID string) RuntimeExecutor

func New() *RunActivityInClusterService {
	return &RunActivityInClusterService{
		namespace:          config.App().DefaultNamespace,
		workflowRepository: config.App().Repository.WorkflowRepository,
		activityRepository: config.App().Repository.ActivityRepository,
		resourceRepository: config.App().Repository.ResourceRepository,
		runtimeResolver: func(runtimeID string) RuntimeExecutor {
			return runtimes.GetRuntimeInstance(runtimeID)
		},
	}
}

func NewWithDependencies(
	workflowRepository ports.WorkflowRepository,
	activityRepository ports.ActivityRepository,
	resourceRepository resource_repository.IRepository,
	runtimeResolver RuntimeResolver,
) *RunActivityInClusterService {
	return &RunActivityInClusterService{
		workflowRepository: workflowRepository,
		activityRepository: activityRepository,
		resourceRepository: resourceRepository,
		runtimeResolver:    runtimeResolver,
	}
}

func (r *RunActivityInClusterService) Run(activityID int) error {

	wfa, err := r.activityRepository.Find(activityID)
	if err != nil {
		config.App().Logger.Infof("WORKER: Activity not found %d", activityID)
		return fmt.Errorf("find activity %d: %w", activityID, err)
	}

	wf, err := r.workflowRepository.Find(wfa.WorkflowId)
	if err != nil {
		return fmt.Errorf("find workflow %d for activity %d: %w", wfa.WorkflowId, activityID, err)
	}

	assignment, err := r.activityRepository.GetActivityScheduleByActivityId(activityID)
	if err != nil || assignment.ResourceID == "" {
		config.App().Logger.Infof("WORKER: Activity %d has no resource assignment", activityID)
		if err != nil {
			return fmt.Errorf("find resource assignment for activity %d: %w", activityID, err)
		}
		return fmt.Errorf("activity %d has no resource assignment", activityID)
	}
	resource, err := r.resourceRepository.FindByID(assignment.ResourceID)
	if err != nil || resource == nil {
		config.App().Logger.Infof("WORKER: Resource %s not found for activity %d", assignment.ResourceID, activityID)
		if err != nil {
			return fmt.Errorf("find resource %s for activity %d: %w", assignment.ResourceID, activityID, err)
		}
		return fmt.Errorf("resource %s not found for activity %d", assignment.ResourceID, activityID)
	}
	runtimeID := resource.RuntimeID

	workflowId := wf.GetId()
	workflowActivityId := wfa.GetId()

	runtime := r.runtimeResolver(runtimeID)
	if runtime == nil {
		return fmt.Errorf("runtime %s not found for resource %s", runtimeID, resource.ID)
	}
	if !runtime.ApplyJob(workflowId, workflowActivityId) {
		return fmt.Errorf("runtime %s rejected activity %d from workflow %d", runtimeID, workflowActivityId, workflowId)
	}

	config.App().Logger.Infof("WORKER: Activity %d started", activityID)
	return nil
}
