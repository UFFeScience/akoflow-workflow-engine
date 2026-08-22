package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/UFFeScience/akoflow/internal/application/ports"
	"github.com/UFFeScience/akoflow/internal/domain"
)

// Discovery reads the Kubernetes node inventory once. It does not create pods
// or inspect workloads, keeping the connection discovery inexpensive.
type Discovery struct{ fallback ClientConfig }

func NewDiscovery(fallback ClientConfig) *Discovery { return &Discovery{fallback: fallback} }

func (d *Discovery) DiscoverConnection(ctx context.Context, connection domain.EnvironmentConnection) (ports.ConnectionDiscovery, error) {
	endpoint := strings.TrimSpace(connection.Endpoint)
	if endpoint == "" {
		endpoint = d.fallback.Endpoint
	}
	token, err := resolveCredential(connection.CredentialRef)
	if err != nil {
		return ports.ConnectionDiscovery{}, err
	}
	if token == "" {
		token = d.fallback.Token
	}
	if endpoint == "" || token == "" {
		return ports.ConnectionDiscovery{}, fmt.Errorf("Kubernetes discovery needs endpoint and credential")
	}
	config := ClientConfig{
		Endpoint: endpoint, Token: token,
		CAFile:                configString(connection.Configuration, "caFile"),
		InsecureSkipTLSVerify: configBool(connection.Configuration, "insecureSkipTlsVerify", d.fallback.InsecureSkipTLSVerify),
	}
	client, err := NewClient(config)
	if err != nil {
		return ports.ConnectionDiscovery{}, err
	}
	payload, err := client.ListCluster(ctx, "nodes")
	if err != nil {
		return ports.ConnectionDiscovery{}, err
	}
	var nodes struct {
		Items []struct {
			Metadata struct {
				Name   string
				Labels map[string]string
			}
			Status struct {
				NodeInfo struct {
					Architecture            string `json:"architecture"`
					ContainerRuntimeVersion string `json:"containerRuntimeVersion"`
				} `json:"nodeInfo"`
				Allocatable map[string]string `json:"allocatable"`
				Conditions  []struct{ Type, Status string }
			}
		}
	}
	if err := json.Unmarshal(payload, &nodes); err != nil {
		return ports.ConnectionDiscovery{}, fmt.Errorf("decode Kubernetes nodes: %w", err)
	}
	ready, names, architectures, runtimes := 0, []string{}, map[string]bool{}, map[string]bool{}
	for _, node := range nodes.Items {
		names = append(names, node.Metadata.Name)
		architectures[node.Status.NodeInfo.Architecture] = true
		runtimes[node.Status.NodeInfo.ContainerRuntimeVersion] = true
		for _, condition := range node.Status.Conditions {
			if condition.Type == "Ready" && condition.Status == "True" {
				ready++
			}
		}
	}
	metadata := map[string]any{
		"nodeCount": len(nodes.Items), "readyNodeCount": ready, "nodeNames": names,
		"architectures": mapKeys(architectures), "containerRuntimes": mapKeys(runtimes),
	}
	return ports.ConnectionDiscovery{Available: ready > 0, Metadata: metadata}, nil
}

func mapKeys(values map[string]bool) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		if value != "" {
			result = append(result, value)
		}
	}
	return result
}
