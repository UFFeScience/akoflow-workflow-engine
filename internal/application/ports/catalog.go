package ports

import (
	"context"

	"github.com/UFFeScience/akoflow/internal/domain"
)

type WorkflowStore interface {
	Create(context.Context, domain.WorkflowDefinition) error
	List(context.Context) ([]domain.WorkflowDefinition, error)
	Find(context.Context, string) (*domain.WorkflowDefinition, error)
	FindVersion(context.Context, string) (*domain.WorkflowVersion, error)
}

type EnvironmentCatalog interface {
	Create(context.Context, domain.EnvironmentDefinition) error
	List(context.Context) ([]domain.EnvironmentDefinition, error)
	Find(context.Context, string) (*domain.EnvironmentDefinition, error)
	EnvironmentStore
}

type PlanStore interface {
	Save(context.Context, domain.SchedulePlan) error
	List(context.Context) ([]domain.SchedulePlan, error)
	Find(context.Context, string) (*domain.SchedulePlan, error)
}

type DataCatalog interface {
	CatalogArtifacts(context.Context, domain.ActivityHandle) error
	ListInstances(context.Context, string) ([]domain.DataObjectInstance, error)
	ListLocations(context.Context, string) ([]domain.DataLocation, error)
	ListArtifacts(context.Context) ([]domain.ExecutableArtifact, error)
	ListArtifactLocations(context.Context) ([]domain.ArtifactLocation, error)
	ListArtifactMaterializations(context.Context, string) ([]domain.ArtifactMaterialization, error)
	SaveArtifactMaterialization(context.Context, domain.ArtifactMaterialization) error
}
