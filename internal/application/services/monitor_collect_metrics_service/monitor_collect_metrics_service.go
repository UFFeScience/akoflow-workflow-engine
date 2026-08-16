package monitor_collect_metrics_service

import (
	"fmt"

	"github.com/UFFeScience/akoflow/internal/application/ports"
	"github.com/UFFeScience/akoflow/internal/application/services/get_pending_workflow_service"
	"github.com/UFFeScience/akoflow/internal/application/services/get_workflow_by_status_service"
	"github.com/UFFeScience/akoflow/internal/domain/workflow/activity"
	"github.com/UFFeScience/akoflow/internal/domain/workflow/definition"
	"github.com/UFFeScience/akoflow/internal/execution/real/runtimes"
)

type MonitorCollectMetricsService struct {
	getPendingWorkflowService PendingWorkflowProvider
	getWorkflowByStatus       ActivityStatusSelector
	runtimeResolver           RuntimeResolver
}

type PendingWorkflowProvider interface {
	GetPendingWorkflows() ([]workflow_entity.Workflow, error)
}
type ActivityStatusSelector interface {
	GetActivitiesByStatus(workflow_entity.Workflow, int) []workflow_activity_entity.WorkflowActivities
}
type RuntimeMetrics interface {
	GetLogs(workflow_entity.Workflow, workflow_activity_entity.WorkflowActivities) string
	GetMetrics(int, int) string
}
type RuntimeResolver func(string) RuntimeMetrics

func New() *MonitorCollectMetricsService {
	pending := get_pending_workflow_service.New()
	status := get_workflow_by_status_service.New()
	return &MonitorCollectMetricsService{getPendingWorkflowService: &pending, getWorkflowByStatus: &status, runtimeResolver: func(id string) RuntimeMetrics { return runtimes.GetRuntimeInstance(id) }}
}

func NewWithDependencies(pending PendingWorkflowProvider, status ActivityStatusSelector, resolver RuntimeResolver) *MonitorCollectMetricsService {
	return &MonitorCollectMetricsService{getPendingWorkflowService: pending, getWorkflowByStatus: status, runtimeResolver: resolver}
}

func (m *MonitorCollectMetricsService) CollectMetrics() error {
	wfsPending, err := m.getPendingWorkflowService.GetPendingWorkflows()
	if err != nil {
		return err
	}

	for _, wf := range wfsPending {
		if err := m.handleCollectMetricsByWorkflow(wf); err != nil {
			return err
		}
	}
	return nil
}

func (m *MonitorCollectMetricsService) handleCollectMetricsByWorkflow(wf workflow_entity.Workflow) error {
	wfaRunning := m.getWorkflowByStatus.GetActivitiesByStatus(wf, ports.ActivityStatusRunning)

	println("Workflow: ", wf.Id)
	println("Running: ", len(wfaRunning))

	for _, a := range wfaRunning {
		runtimeService := m.runtimeResolver(a.GetRuntimeId())
		if runtimeService == nil {
			return fmt.Errorf("runtime %s not found", a.GetRuntimeId())
		}
		runtimeService.GetLogs(wf, a)
		runtimeService.GetMetrics(wf.Id, a.Id)
	}
	return nil
}
