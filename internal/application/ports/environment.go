package ports

import (
	"context"

	"github.com/UFFeScience/akoflow/internal/domain"
)

type ConnectionHealth struct {
	Healthy bool
	Message string
}

type DiscoveryRequest struct {
	Environment domain.Environment
	Version     domain.EnvironmentVersion
	Connections []domain.EnvironmentConnection
	Runtimes    []domain.EnvironmentRuntime
}

type DiscoveryResult struct {
	Resources    []domain.Resource
	Snapshots    []domain.ResourceSnapshot
	NetworkLinks []domain.NetworkLink
	Capabilities domain.EnvironmentCapabilities
}

type EnvironmentConnector interface {
	Connect(context.Context, domain.EnvironmentConnection) error
	Health(context.Context, domain.EnvironmentConnection) (ConnectionHealth, error)
	Close(context.Context, domain.EnvironmentConnection) error
}

type EnvironmentDiscoverer interface {
	Discover(context.Context, DiscoveryRequest) (DiscoveryResult, error)
}

type EnvironmentRepository interface {
	UpdateStatus(context.Context, string, domain.EnvironmentStatus) error
	UpsertConnection(context.Context, domain.EnvironmentConnection) error
	ListConnections(context.Context, string) ([]domain.EnvironmentConnection, error)
}
