package ports

import (
	"context"

	"github.com/UFFeScience/akoflow/internal/domain/instance"
)

type InstanceStore interface {
	Find(context.Context) (*instance.Instance, error)
	Save(context.Context, instance.Instance) error
}
