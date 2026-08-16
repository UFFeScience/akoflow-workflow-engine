package ports

import (
	"context"

	"github.com/UFFeScience/akoflow/internal/domain"
)

type WorkflowRepository interface {
	Create(context.Context, domain.WorkflowDefinition) error
	FindVersion(context.Context, string) (*domain.WorkflowVersion, error)
}

type EnvironmentCatalog interface {
	Create(context.Context, domain.EnvironmentDefinition) error
	EnvironmentRepository
}

type PlanningRepository interface {
	Save(context.Context, domain.SchedulePlan) error
	Find(context.Context, string) (*domain.SchedulePlan, error)
}
