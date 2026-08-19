package ports

import (
	"context"

	"github.com/UFFeScience/akoflow/internal/domain"
)

type ResourceInventory interface {
	Upsert(context.Context, domain.Resource) error
	List(context.Context) ([]domain.Resource, error)
	FindByID(context.Context, string) (*domain.Resource, error)
	FindByProviderID(context.Context, string, string) (*domain.Resource, error)
	ListByRuntime(context.Context, string, string) ([]domain.Resource, error)
	ListSchedulable(context.Context, string) ([]domain.Resource, error)
	CreateSnapshot(context.Context, domain.ResourceSnapshot) error
	LatestSnapshot(context.Context, string) (*domain.ResourceSnapshot, error)
}
