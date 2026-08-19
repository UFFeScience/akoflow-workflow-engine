package ports

import (
	"context"
	"io"

	"github.com/UFFeScience/akoflow/internal/domain"
)

type PutObjectRequest struct {
	Storage domain.StorageResource
	Key     string
	Source  io.Reader
	Size    int64
}

type GetObjectRequest struct {
	Location domain.DataLocation
	Target   io.Writer
}

type ObjectStat struct {
	SizeBytes int64
	Checksum  string
}

type StorageDriver interface {
	Type() domain.StorageType
	Put(context.Context, PutObjectRequest) (domain.DataLocation, error)
	Get(context.Context, GetObjectRequest) error
	Stat(context.Context, domain.DataLocation) (ObjectStat, error)
	Delete(context.Context, domain.DataLocation) error
}

// StorageCatalog resolves the storage resources visible to a runtime. A
// runtime may expose several storages, but at most one is its implicit default.
type StorageCatalog interface {
	ListRuntimeStorages(context.Context, string, string) ([]domain.StorageResource, error)
	FindDefaultRuntimeStorage(context.Context, string, string) (*domain.StorageResource, error)
}
