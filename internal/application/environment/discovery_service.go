package environment

import (
	"context"
	"fmt"
	"strings"
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
	nodeSnapshots, err := s.materializeNodes(ctx, *definition, connection, observation)
	if err != nil {
		return nil, err
	}
	resourceIDs := boundResourceIDs(*definition, connection.ID)
	if len(resourceIDs) == 0 {
		return nil, fmt.Errorf("connection %q has no bound resources", connection.ID)
	}
	now := time.Now().UTC()
	items := make([]domain.ResourceSnapshot, 0, len(resourceIDs)+len(nodeSnapshots))
	items = append(items, nodeSnapshots...)
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

func (s *DiscoveryCoordinator) materializeNodes(ctx context.Context, definition domain.EnvironmentDefinition, connection domain.EnvironmentConnection, observation ports.ConnectionDiscovery) ([]domain.ResourceSnapshot, error) {
	if len(observation.Nodes) == 0 {
		return nil, nil
	}
	runtimeIDs := runtimeIDsForConnection(definition, connection.ID)
	partitions := map[string]domain.Resource{}
	var clusterID string
	for _, resource := range definition.Resources {
		if resource.Type == domain.ResourceCluster && clusterID == "" {
			clusterID = resource.ID
		}
		if resource.Type == domain.ResourceHPCPartition {
			partitions[resource.ProviderID] = resource
			partitions[resource.Name] = resource
			if clusterID == "" && resource.ParentResourceID != nil {
				clusterID = *resource.ParentResourceID
			}
		}
	}
	now := time.Now().UTC()
	snapshots := make([]domain.ResourceSnapshot, 0, len(observation.Nodes))
	for _, discovered := range observation.Nodes {
		resourceID := discoveredNodeID(connection.ID, discovered.Name)
		var parent *string
		if clusterID != "" {
			parent = &clusterID
		}
		metadata := cloneMetadata(discovered.Metadata)
		metadata["connectionId"] = connection.ID
		metadata["discovered"] = true
		resource := domain.Resource{ID: resourceID, EnvironmentVersionID: definition.Version.ID,
			ParentResourceID: parent, ExecutionTarget: domain.ExecutionTargetBatch,
			Type: domain.ResourceHPCMachine, Name: discovered.Name, ProviderID: discovered.Name,
			Architecture: discovered.Architecture, CPUCores: discovered.CPUCores,
			CPUCapacity: float64(discovered.CPUCores), MemoryBytes: discovered.MemoryBytes,
			StorageBytes: discovered.StorageBytes, ComputeSpeedup: 1, Schedulable: true, Metadata: metadata}
		if err := s.resources.Upsert(ctx, resource); err != nil {
			return nil, fmt.Errorf("upsert discovered SLURM node %q: %w", discovered.Name, err)
		}
		for _, runtimeID := range runtimeIDs {
			if err := s.resources.UpsertRuntimeBinding(ctx, domain.ResourceRuntimeBinding{ResourceID: resourceID, RuntimeID: runtimeID, Enabled: true}); err != nil {
				return nil, fmt.Errorf("bind discovered SLURM node %q: %w", discovered.Name, err)
			}
		}
		for _, partitionName := range discovered.Partitions {
			partition, ok := partitions[partitionName]
			if !ok {
				continue
			}
			if err := s.resources.UpsertRelation(ctx, domain.ResourceRelation{EnvironmentVersionID: definition.Version.ID,
				SourceResourceID: partition.ID, TargetResourceID: resourceID, Type: domain.ResourceRelationContains,
				Metadata: map[string]any{"discovered": true}}); err != nil {
				return nil, fmt.Errorf("relate partition %q to node %q: %w", partitionName, discovered.Name, err)
			}
		}
		snapshot := domain.ResourceSnapshot{ID: provider.NewID("resource-discovery"), ResourceID: resourceID,
			CapturedAt: now, Available: slurmNodeAvailable(discovered.State), Metadata: metadata}
		if err := s.resources.CreateSnapshot(ctx, snapshot); err != nil {
			return nil, fmt.Errorf("snapshot discovered SLURM node %q: %w", discovered.Name, err)
		}
		snapshots = append(snapshots, snapshot)
	}
	return snapshots, nil
}

func runtimeIDsForConnection(definition domain.EnvironmentDefinition, connectionID string) []string {
	result := []string{}
	for _, runtime := range definition.Runtimes {
		if configuredConnectionID(runtime.Configuration) == connectionID {
			result = append(result, runtime.ID)
		}
	}
	return result
}

func discoveredNodeID(connectionID, nodeName string) string {
	replacer := strings.NewReplacer("/", "-", " ", "-", ":", "-")
	return replacer.Replace(connectionID + "-node-" + nodeName)
}

func slurmNodeAvailable(state string) bool {
	state = strings.ToLower(strings.TrimRight(strings.TrimSpace(state), "*~#$+"))
	for _, unavailable := range []string{"down", "drain", "drained", "fail", "failing", "unknown"} {
		if state == unavailable {
			return false
		}
	}
	return state != ""
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
