package monitor_change_workflow_service

import (
	"fmt"

	"github.com/UFFeScience/akoflow/internal/application/ports"
	"github.com/UFFeScience/akoflow/internal/execution/real/runtimes"
	"github.com/UFFeScience/akoflow/internal/infrastructure/config"

	"github.com/UFFeScience/akoflow/internal/application/services/get_pending_workflow_service"
	"github.com/UFFeScience/akoflow/internal/application/services/get_workflow_by_status_service"
	"github.com/UFFeScience/akoflow/internal/domain/workflow/activity"
	"github.com/UFFeScience/akoflow/internal/domain/workflow/definition"
)

type MonitorChangeWorkflowService struct {
	namespace                 string
	workflowRepository        ports.WorkflowRepository
	getPendingWorkflowService PendingWorkflowProvider
	getWorkflowByStatus       WorkflowStatusSelector
	runtimeResolver           RuntimeResolver
}

type PendingWorkflowProvider interface {
	GetPendingWorkflows() ([]workflow_entity.Workflow, error)
}

type WorkflowStatusSelector interface {
	GetActivitiesByStatus(workflow_entity.Workflow, int) []workflow_activity_entity.WorkflowActivities
}

type RuntimeVerifier interface {
	VerifyActivitiesWasFinished(workflow_entity.Workflow) bool
}

type RuntimeResolver func(string) RuntimeVerifier

func New() *MonitorChangeWorkflowService {
	pending := get_pending_workflow_service.New()
	status := get_workflow_by_status_service.New()
	return &MonitorChangeWorkflowService{
		namespace:          "akoflow",
		workflowRepository: config.App().Repository.WorkflowRepository,

		getPendingWorkflowService: &pending,
		getWorkflowByStatus:       &status,
		runtimeResolver: func(runtimeID string) RuntimeVerifier {
			return runtimes.GetRuntimeInstance(runtimeID)
		},
	}
}

func NewWithDependencies(workflowRepository ports.WorkflowRepository, pending PendingWorkflowProvider, status WorkflowStatusSelector, runtimeResolver RuntimeResolver) *MonitorChangeWorkflowService {
	return &MonitorChangeWorkflowService{
		workflowRepository:        workflowRepository,
		getPendingWorkflowService: pending,
		getWorkflowByStatus:       status,
		runtimeResolver:           runtimeResolver,
	}
}

func (m *MonitorChangeWorkflowService) MonitorChangeWorkflow() error {
	wfsPending, err := m.getPendingWorkflowService.GetPendingWorkflows()
	if err != nil {
		return fmt.Errorf("load pending workflows: %w", err)
	}
	if err := m.handleVerifyWorkflowWasFinished(wfsPending); err != nil {
		return err
	}
	return m.handleVerifyWorkflowActivities(wfsPending)
}

func (m *MonitorChangeWorkflowService) handleVerifyWorkflowActivities(wfs []workflow_entity.Workflow) error {

	for _, wf := range wfs {
		runtimesByWorkflow := wf.GetRuntimeId()
		for _, runtimeID := range runtimesByWorkflow {
			runtime := m.runtimeResolver(runtimeID)
			if runtime == nil {
				return fmt.Errorf("runtime %s not found for workflow %d", runtimeID, wf.Id)
			}
			runtime.VerifyActivitiesWasFinished(wf)
		}
	}
	return nil
}

func (m *MonitorChangeWorkflowService) handleVerifyWorkflowWasFinished(wfs []workflow_entity.Workflow) error {
	for _, wf := range wfs {
		wfaRunning := m.getWorkflowByStatus.GetActivitiesByStatus(wf, ports.ActivityStatusRunning)
		wfaCreated := m.getWorkflowByStatus.GetActivitiesByStatus(wf, ports.ActivityStatusCreated)

		if len(wfaRunning) == 0 && len(wfaCreated) == 0 {
			println("Workflow finished: ", wf.Id)
			if err := m.workflowRepository.UpdateStatus(wf.Id, ports.WorkflowStatusFinished); err != nil {
				return fmt.Errorf("finish workflow %d: %w", wf.Id, err)
			}
		}
	}
	return nil
}
