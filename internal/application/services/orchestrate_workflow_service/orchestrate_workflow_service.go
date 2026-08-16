package orchestrate_workflow_service

import (
	"fmt"

	"github.com/UFFeScience/akoflow/internal/application/ports"
	"github.com/UFFeScience/akoflow/internal/application/services/get_workflow_by_status_service"
	"github.com/UFFeScience/akoflow/internal/application/services/orchestrate_schedule_service"
	"github.com/UFFeScience/akoflow/internal/domain/workflow/activity"
	"github.com/UFFeScience/akoflow/internal/domain/workflow/definition"
	"github.com/UFFeScience/akoflow/internal/execution/lifecycle/channel"
	"github.com/UFFeScience/akoflow/internal/infrastructure/config"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/schedule_repository"
)

type OrchestrateWorflowService struct {
	namespace           string
	channelManager      *channel.Manager
	getWorkflowByStatus ActivityStatusSelector
	scheduleRepository  schedule_repository.IScheduleRepository
	// runScheduleService  run_schedule_service.RunScheduleService
	orchestrateScheduleService orchestrate_schedule_service.OrchestrateScheduleService
	scheduleActivities         ScheduleActivities

	workflowsRunning map[int][]workflow_activity_entity.WorkflowActivities
}

type ActivityStatusSelector interface {
	GetActivitiesByStatus(workflow_entity.Workflow, int) []workflow_activity_entity.WorkflowActivities
}

type ScheduleActivities func(workflow_entity.Workflow, []workflow_activity_entity.WorkflowActivities) ([]workflow_activity_entity.WorkflowActivities, error)

func New() *OrchestrateWorflowService {
	statusSelector := get_workflow_by_status_service.New()
	service := &OrchestrateWorflowService{
		namespace:           "akoflow",
		channelManager:      channel.GetInstance(),
		getWorkflowByStatus: &statusSelector,
		scheduleRepository:  config.App().Repository.ScheduleRepository,
		// runScheduleService:  run_schedule_service.New(),
		orchestrateScheduleService: orchestrate_schedule_service.New(),
	}
	service.scheduleActivities = func(workflow workflow_entity.Workflow, activities []workflow_activity_entity.WorkflowActivities) ([]workflow_activity_entity.WorkflowActivities, error) {
		return service.orchestrateScheduleService.SetWorkflow(workflow).SetReadyToRunActivities(activities).Orchestrate()
	}
	return service
}

func NewWithDependencies(namespace string, channelManager *channel.Manager, statusSelector ActivityStatusSelector, scheduleActivities ScheduleActivities) *OrchestrateWorflowService {
	return &OrchestrateWorflowService{
		namespace:           namespace,
		channelManager:      channelManager,
		getWorkflowByStatus: statusSelector,
		scheduleActivities:  scheduleActivities,
		workflowsRunning:    make(map[int][]workflow_activity_entity.WorkflowActivities),
	}
}

func (o *OrchestrateWorflowService) SetWorkflowsRunning(workflowsRunning map[int][]workflow_activity_entity.WorkflowActivities) *OrchestrateWorflowService {
	o.workflowsRunning = workflowsRunning
	return o
}

func (o *OrchestrateWorflowService) dispatchToWorker(activities []workflow_activity_entity.WorkflowActivities) {
	for _, activity := range activities {
		println("Dispatching to worker activity: ", activity.Name, " with id: ", activity.Id)
		if o.channelManager != nil {
			o.channelManager.WorfklowChannel <- channel.DataChannel{Namespace: o.namespace, Job: activity, Id: activity.Id}
		}
	}
}

func (o *OrchestrateWorflowService) hasSyncingActivity(wf workflow_entity.Workflow) bool {
	for _, activity := range wf.Spec.Activities {
		if activity.Status == ports.ActivityStatusSyncing {
			return true
		}
	}

	return false
}

func (o *OrchestrateWorflowService) handleDispatchToWorker(wf workflow_entity.Workflow) []workflow_activity_entity.WorkflowActivities {
	if o.hasSyncingActivity(wf) {
		config.App().Logger.Infof("WORKER: Workflow %d is syncing, skip dispatch", wf.Id)
		return []workflow_activity_entity.WorkflowActivities{}
	}

	wfsFinished := o.getWorkflowByStatus.GetActivitiesByStatus(wf, ports.ActivityStatusFinished)
	wfsRunning := o.getWorkflowByStatus.GetActivitiesByStatus(wf, ports.ActivityStatusRunning)
	wfsNotStarted := o.getWorkflowByStatus.GetActivitiesByStatus(wf, ports.ActivityStatusCreated)

	println("wfsFinished: ", len(wfsFinished))
	println("wfsRunning: ", len(wfsRunning))
	println("wfsNotStarted: ", len(wfsNotStarted))

	wfNextToRun := o.nextToRun(wfsNotStarted, wfsFinished)

	wfNextToRun = o.handleSchedule(wf, wfNextToRun)

	for _, wfNextToRun := range wfNextToRun {
		o.dispatchToWorker([]workflow_activity_entity.WorkflowActivities{wfNextToRun})
	}

	return wfNextToRun

}

func (o *OrchestrateWorflowService) handleSchedule(wf workflow_entity.Workflow, wfsNextToRun []workflow_activity_entity.WorkflowActivities) []workflow_activity_entity.WorkflowActivities {

	newWfsNextToRun, err := o.scheduleActivities(wf, wfsNextToRun)

	if err != nil {
		fmt.Println("Erro ao executar o plugin:", err)
		return wfsNextToRun
	}

	return newWfsNextToRun
}

func (o *OrchestrateWorflowService) nextToRun(wfsPending []workflow_activity_entity.WorkflowActivities, wfsFinished []workflow_activity_entity.WorkflowActivities) []workflow_activity_entity.WorkflowActivities {

	wfsNextToRun := make([]workflow_activity_entity.WorkflowActivities, 0)
	for _, wfPending := range wfsPending {
		if o.isDependentOnFinished(wfPending, wfsFinished) {
			if wfPending.Id != 0 {
				wfsNextToRun = append(wfsNextToRun, wfPending)
			}
		}
	}

	return wfsNextToRun
}

func (o *OrchestrateWorflowService) isDependentOnFinished(wfaPending workflow_activity_entity.WorkflowActivities, wfasFinished []workflow_activity_entity.WorkflowActivities) bool {
	if len(wfaPending.DependsOn) == 0 {
		return true
	}

	mapNameCompleted := make(map[string]bool)

	for _, wfaFinished := range wfasFinished {
		for _, dependOn := range wfaPending.DependsOn {
			if dependOn == wfaFinished.Name {
				mapNameCompleted[wfaFinished.Name] = true
			}
		}
	}

	return len(wfaPending.DependsOn) == len(mapNameCompleted)
}

func (o *OrchestrateWorflowService) iterateWorkflows(workflows []workflow_entity.Workflow) map[int][]workflow_activity_entity.WorkflowActivities {

	mapWfWfs := make(map[int][]workflow_activity_entity.WorkflowActivities)
	for _, wf := range workflows {
		mapWfWfs[wf.Id] = o.handleDispatchToWorker(wf)
	}

	return mapWfWfs
}

func (d *OrchestrateWorflowService) Orchestrate(workflows []workflow_entity.Workflow) map[int][]workflow_activity_entity.WorkflowActivities {
	return d.iterateWorkflows(workflows)
}
