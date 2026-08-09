package ports

import (
	"context"

	"github.com/UFFeScience/akoflow/internal/domain"
)

type ExecutionRequest struct {
	Run              domain.ExecutionRun
	Plan             domain.SchedulePlan
	Workflow         domain.WorkflowVersion
	Resources        []domain.Resource
	NetworkLinks     []domain.NetworkLink
	ActivityProfiles []domain.ActivityResourceProfile
}

type PlanExecutor interface {
	Execute(context.Context, ExecutionRequest) (domain.ExecutionTrace, error)
}

type RuntimeAdapter interface {
	Submit(context.Context, domain.ActivityExecutionContext) (string, error)
	Status(context.Context, string) (domain.TaskExecutionStatus, error)
	Cancel(context.Context, string) error
}
