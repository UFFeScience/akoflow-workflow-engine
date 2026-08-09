package monitor_collect_metrics_service

import (
	"github.com/UFFeScience/akoflow/internal/application/services/get_pending_workflow_service"
	"github.com/UFFeScience/akoflow/internal/application/services/get_workflow_by_status_service"
	"github.com/UFFeScience/akoflow/internal/domain/workflow/definition"
	"github.com/UFFeScience/akoflow/internal/execution/real/runtimes"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/activity_repository"
)

type MonitorCollectMetricsService struct {
	getPendingWorkflowService get_pending_workflow_service.GetPendingWorkflowService
	getWorkflowByStatus       get_workflow_by_status_service.GetWorkflowByStatusService
}

func New() *MonitorCollectMetricsService {
	return &MonitorCollectMetricsService{
		getPendingWorkflowService: get_pending_workflow_service.New(),
		getWorkflowByStatus:       get_workflow_by_status_service.New(),
	}
}

func (m *MonitorCollectMetricsService) CollectMetrics() {
	wfsPending, _ := m.getPendingWorkflowService.GetPendingWorkflows()

	for _, wf := range wfsPending {
		m.handleCollectMetricsByWorkflow(wf)
	}
}

func (m *MonitorCollectMetricsService) handleCollectMetricsByWorkflow(wf workflow_entity.Workflow) {
	wfaRunning := m.getWorkflowByStatus.GetActivitiesByStatus(wf, activity_repository.StatusRunning)

	println("Workflow: ", wf.Id)
	println("Running: ", len(wfaRunning))

	for _, a := range wfaRunning {
		runtimeService := runtimes.GetRuntimeInstance(a.GetRuntimeId())
		runtimeService.GetLogs(wf, a)
		runtimeService.GetMetrics(wf.Id, a.Id)
	}
}
