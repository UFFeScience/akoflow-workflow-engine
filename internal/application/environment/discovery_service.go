package environment

import (
	"context"
	"fmt"
	"time"

	"github.com/UFFeScience/akoflow/internal/application/ports"
	"github.com/UFFeScience/akoflow/internal/domain"
	"github.com/UFFeScience/akoflow/internal/provider"
)

// DiscoveryCoordinator executes one cheap, read-only probe per connector and
// persists the resulting effective capabilities on its bound resources.
type DiscoveryCoordinator struct {
	catalog     ports.EnvironmentCatalog
	resources   ports.ResourceInventory
	discoverers map[domain.ConnectionType]ports.ConnectionDiscoverer
}

var _ ports.EnvironmentDiscovery = (*DiscoveryCoordinator)(nil)

func NewDiscoveryCoordinator(catalog ports.EnvironmentCatalog, resources ports.ResourceInventory, discoverers map[domain.ConnectionType]ports.ConnectionDiscoverer) *DiscoveryCoordinator {
	return &DiscoveryCoordinator{catalog: catalog, resources: resources, discoverers: discoverers}
}

func (s *DiscoveryCoordinator) DiscoverConnection(ctx context.Context, connectionID string) ([]domain.ResourceSnapshot, error) {
	definition, connection, err := s.findConnection(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	discoverer := s.discoverers[connection.Type]
	if discoverer == nil {
		return nil, fmt.Errorf("no discovery driver is configured for connection type %q", connection.Type)
	}
	observation, err := discoverer.DiscoverConnection(ctx, connection)
	if err != nil {
		return nil, err
	}
	resourceIDs := boundResourceIDs(*definition, connection.ID)
	if len(resourceIDs) == 0 {
		return nil, fmt.Errorf("connection %q has no bound resources", connection.ID)
	}
	now := time.Now().UTC()
	items := make([]domain.ResourceSnapshot, 0, len(resourceIDs))
	for _, resourceID := range resourceIDs {
		metadata := cloneMetadata(observation.Metadata)
		metadata["connectionId"] = connection.ID
		metadata["warnings"] = observation.Warnings
		metadata["discoverySource"] = string(connection.Type)
		snapshot := domain.ResourceSnapshot{
			ID: provider.NewID("resource-discovery"), ResourceID: resourceID,
			CapturedAt: now, Available: observation.Available, Metadata: metadata,
		}
		if err := s.resources.CreateSnapshot(ctx, snapshot); err != nil {
			return nil, err
		}
		items = append(items, snapshot)
	}
	return items, nil
}

func (s *DiscoveryCoordinator) findConnection(ctx context.Context, id string) (*domain.EnvironmentDefinition, domain.EnvironmentConnection, error) {
	definitions, err := s.catalog.List(ctx)
	if err != nil {
		return nil, domain.EnvironmentConnection{}, err
	}
	for index := range definitions {
		for _, connection := range definitions[index].Connections {
			if connection.ID == id {
				return &definitions[index], connection, nil
			}
		}
	}
	return nil, domain.EnvironmentConnection{}, fmt.Errorf("environment connection %q was not found", id)
}

func boundResourceIDs(definition domain.EnvironmentDefinition, connectionID string) []string {
	runtimeIDs := map[string]bool{}
	for _, runtime := range definition.Runtimes {
		if configuredConnectionID(runtime.Configuration) == connectionID {
			runtimeIDs[runtime.ID] = true
		}
	}
	ids := map[string]bool{}
	for _, binding := range definition.RuntimeBindings {
		if binding.Enabled && runtimeIDs[binding.RuntimeID] {
			ids[binding.ResourceID] = true
		}
	}
	result := make([]string, 0, len(ids))
	for _, resource := range definition.Resources {
		if ids[resource.ID] {
			result = append(result, resource.ID)
		}
	}
	return result
}

func configuredConnectionID(configuration map[string]any) string {
	value, _ := configuration["connectionId"].(string)
	return value
}
func cloneMetadata(input map[string]any) map[string]any {
	output := map[string]any{}
	for key, value := range input {
		output[key] = value
	}
	return output
}
