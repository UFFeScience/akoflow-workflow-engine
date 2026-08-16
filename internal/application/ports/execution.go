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
	Mode() domain.ExecutionMode
	Start(context.Context, domain.ActivityExecutionContext) (domain.ActivityHandle, error)
	Inspect(context.Context, domain.ActivityHandle) (domain.ActivityHandle, error)
	Stop(context.Context, domain.ActivityHandle) error
}

// RuntimeResolver selects an adapter from execution mode and runtime identity.
// It is the only point where application code knows that multiple execution
// technologies exist.
type RuntimeResolver interface {
	Resolve(domain.ExecutionMode, string) (RuntimeAdapter, error)
}

type ActivityHandleRepository interface {
	Save(context.Context, domain.ActivityHandle) error
	Find(context.Context, string) (*domain.ActivityHandle, error)
}
