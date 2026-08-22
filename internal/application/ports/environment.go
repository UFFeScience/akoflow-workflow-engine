package ports

import (
	"context"

	"github.com/UFFeScience/akoflow/internal/domain"
)

type ConnectionHealth struct {
	Healthy bool
	Message string
}

type ConnectionProber interface {
	Probe(context.Context, domain.EnvironmentConnection) ConnectionHealth
}

type ConnectionHealthMonitor interface {
	Check(context.Context, string) (domain.ConnectionCheck, error)
	History(context.Context, string, int) ([]domain.ConnectionCheck, error)
}

type DiscoveryRequest struct {
	Environment domain.Environment
	Version     domain.EnvironmentVersion
	Connections []domain.EnvironmentConnection
	Runtimes    []domain.EnvironmentRuntime
}

type DiscoveryResult struct {
	Resources       []domain.Resource
	Snapshots       []domain.ResourceSnapshot
	NetworkTopology domain.NetworkTopology
	Capabilities    domain.EnvironmentCapabilities
}

type EnvironmentConnector interface {
	Connect(context.Context, domain.EnvironmentConnection) error
	Health(context.Context, domain.EnvironmentConnection) (ConnectionHealth, error)
	Close(context.Context, domain.EnvironmentConnection) error
}

type EnvironmentDiscoverer interface {
	Discover(context.Context, DiscoveryRequest) (DiscoveryResult, error)
}

// ConnectionDiscovery is a bounded, read-only observation of one real
// infrastructure connection. Resource snapshots receive this data; plans do
// not trigger discovery themselves.
type ConnectionDiscovery struct {
	Available bool
	Metadata  map[string]any
	Warnings  []string
}

type ConnectionDiscoverer interface {
	DiscoverConnection(context.Context, domain.EnvironmentConnection) (ConnectionDiscovery, error)
}

type EnvironmentDiscovery interface {
	DiscoverConnection(context.Context, string) ([]domain.ResourceSnapshot, error)
}

type EnvironmentStore interface {
	UpdateStatus(context.Context, string, domain.EnvironmentStatus) error
	UpsertConnection(context.Context, domain.EnvironmentConnection) error
	ListConnections(context.Context, string) ([]domain.EnvironmentConnection, error)
}

type ConnectionStore interface {
	FindConnection(context.Context, string) (*domain.EnvironmentConnection, error)
	SaveConnectionCheck(context.Context, domain.ConnectionCheck) error
	ListConnectionChecks(context.Context, string, int) ([]domain.ConnectionCheck, error)
}

type NetworkTopologyStore interface {
	Create(context.Context, domain.NetworkTopology) error
	List(context.Context) ([]domain.NetworkTopology, error)
	Find(context.Context, string) (*domain.NetworkTopology, error)
}

type ExecutionScopeStore interface {
	CreateScope(context.Context, domain.ExecutionScope) error
	ListScopes(context.Context) ([]domain.ExecutionScope, error)
	FindScope(context.Context, string) (*domain.ExecutionScope, error)
}
