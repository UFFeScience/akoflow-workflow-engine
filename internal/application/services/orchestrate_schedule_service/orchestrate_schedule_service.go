package orchestrate_schedule_service

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"plugin"

	"github.com/UFFeScience/akoflow/internal/application/services/resource_current_metrics_service"
	"github.com/UFFeScience/akoflow/internal/domain/workflow/activity"
	"github.com/UFFeScience/akoflow/internal/domain/workflow/definition"
	"github.com/UFFeScience/akoflow/internal/infrastructure/config"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/activity_repository"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/resource_repository"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/schedule_repository"
)

type OrchestrateScheduleService struct {
	scheduleRepository schedule_repository.IScheduleRepository
	resourceRepository resource_repository.IRepository
	activityRepository activity_repository.IActivityRepository

	resourceMetricsService resource_current_metrics_service.Service

	workflow             workflow_entity.Workflow
	readyToRunActivities []workflow_activity_entity.WorkflowActivities
}

func New() OrchestrateScheduleService {
	return OrchestrateScheduleService{
		scheduleRepository: config.App().Repository.ScheduleRepository,
		activityRepository: config.App().Repository.ActivityRepository,
		resourceRepository: config.App().Repository.ResourceRepository,

		resourceMetricsService: resource_current_metrics_service.New(),

		workflow:             workflow_entity.Workflow{},
		readyToRunActivities: []workflow_activity_entity.WorkflowActivities{},
	}
}

func (r *OrchestrateScheduleService) SetWorkflow(workflow workflow_entity.Workflow) *OrchestrateScheduleService {
	r.workflow = workflow
	return r
}

func (r *OrchestrateScheduleService) SetReadyToRunActivities(activities []workflow_activity_entity.WorkflowActivities) *OrchestrateScheduleService {
	r.readyToRunActivities = activities
	return r
}

type ResponseStartSchedule map[string]any

func (o *OrchestrateScheduleService) Orchestrate() ([]workflow_activity_entity.WorkflowActivities, error) {

	scheduleName := o.workflow.Spec.Schedule

	scheduleDB, err := o.scheduleRepository.GetScheduleByName(scheduleName)

	if err != nil {
		// default schedule
		println("No schedule found, using default")
		return o.readyToRunActivities, nil
	}

	println("Schedule found: ", scheduleDB.Name)

	newReadyToRunActivities := []workflow_activity_entity.WorkflowActivities{}

	for _, activity := range o.readyToRunActivities {
		response := make([]ResponseStartSchedule, 0)

		activityIScheduled, err := o.activityRepository.IsActivityScheduled(activity.WorkflowId, activity.Id)
		if err != nil {
			config.App().Logger.Error("Error checking if activity is scheduled: " + err.Error())
			return nil, err
		}
		if activityIScheduled {
			continue
		}

		resources, err := o.resourceRepository.ListByRuntime(o.workflow.Spec.EnvironmentVersionID, o.workflow.Spec.Runtime)
		if err != nil {
			config.App().Logger.Error("Error getting nodes: " + err.Error())
			return nil, err
		}
		for _, resource := range resources {

			resourceMetrics, err := o.resourceMetricsService.Get(resource.ID, newReadyToRunActivities)

			if err != nil {
				config.App().Logger.Error("Error getting activity schedule by node name: " + err.Error())
				return nil, err
			}

			input := map[string]any{
				"time_estimate":   1.0,
				"memory_required": activity.GetMemoryRequired(),
				"vcpus_required":  activity.GetCpuRequired(),
				"memory_free":     resourceMetrics.MemoryFree(),
				"memory_max":      resourceMetrics.MemoryTotal,
				"vcpus_available": resourceMetrics.CPUFree(),
				"alpha":           0.0,
				"activity_name":   activity.GetName(),
				"machine_type":    resource.Name,
			}

			akoScore, _ := o.StartRunSchedule(scheduleName, input)
			response = append(response, ResponseStartSchedule{
				"activity_id": activity.Id,
				"resource_id": resource.ID,
				"ako_score":   akoScore,
				"input":       input,
			})

		}

		bestNode := o.getBestNode(response)

		if bestNode == nil {
			config.App().Logger.Info("No suitable node found for activity: " + activity.GetName())
			continue
		}

		if bestNode["ako_score"].(float64) == 0 {
			config.App().Logger.Info("No suitable node found for activity (score 0): " + activity.GetName())
			continue
		}

		metadataMap := map[string]any{
			"cpu":          activity.GetCpuRequired(),
			"memory":       activity.GetMemoryRequired(),
			"currentScore": bestNode["ako_score"].(float64),
			"othersScores": response,
		}

		metadataByte, err := json.Marshal(metadataMap)
		if err != nil {
			config.App().Logger.Error("Error marshalling metadata: " + err.Error())
			return nil, err
		}

		metadata := string(metadataByte)

		o.activityRepository.SetActivitySchedule(
			activity.WorkflowId,
			activity.Id,
			bestNode["resource_id"].(string),
			scheduleName,
			activity.GetCpuRequired(),
			activity.GetMemoryRequired(),
			metadata,
		)

		newReadyToRunActivities = append(newReadyToRunActivities, activity)
	}

	return newReadyToRunActivities, nil

}

func (o *OrchestrateScheduleService) getBestNode(response []ResponseStartSchedule) ResponseStartSchedule {
	if len(response) == 0 {
		return nil
	}
	bestNode := response[0]

	for _, res := range response {
		if res["ako_score"].(float64) > bestNode["ako_score"].(float64) {
			bestNode = res
		}
	}

	return bestNode
}

func (r *OrchestrateScheduleService) StartRunSchedule(scheduleName string, input map[string]any) (float64, error) {
	// Here you would implement the logic to start running the schedule
	// For example, you might want to fetch the schedule by name and then execute it with the provided input

	schedule, err := r.scheduleRepository.GetScheduleByName(scheduleName)

	if err != nil {
		config.App().Logger.Error("Error getting schedule: " + err.Error())
		return 0, err
	}

	println("Schedule found: ", schedule.Name)

	p, err := plugin.Open(filepath.Clean(schedule.PluginSoPath))

	if err != nil {
		fmt.Println("Erro ao abrir plugin:", err)
		return 0, err
	}

	sym, err := p.Lookup("AkoScore")
	if err != nil {
		fmt.Println("Erro ao procurar símbolo 'AkoScore':", err)
		return 0, err
	}

	akoScoreFunc, ok := sym.(func(any) float64)

	if !ok {
		fmt.Println("Símbolo 'AkoScore' não é uma função válida")
		return 0, fmt.Errorf("invalid AkoScore function")
	}

	result := akoScoreFunc(input)

	return result, nil
}
