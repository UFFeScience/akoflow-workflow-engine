package slurm

import (
	"context"
	"fmt"
	"strings"

	"github.com/UFFeScience/akoflow/internal/application/ports"
	"github.com/UFFeScience/akoflow/internal/domain"
	runtimecommon "github.com/UFFeScience/akoflow/internal/provider"
)

// Discovery collects the scheduler view from the login node once. It queries
// partitions, queue depth, container runtime and mounted filesystems only.
type Discovery struct{ executor runtimecommon.CommandExecutor }

func NewDiscovery(executors ...runtimecommon.CommandExecutor) *Discovery {
	executor := runtimecommon.CommandExecutor(runtimecommon.OSCommandExecutor{})
	if len(executors) > 0 && executors[0] != nil {
		executor = executors[0]
	}
	return &Discovery{executor: executor}
}

func (d *Discovery) DiscoverConnection(ctx context.Context, connection domain.EnvironmentConnection) (ports.ConnectionDiscovery, error) {
	if connection.Type != domain.ConnectionSSH && connection.Type != domain.ConnectionAgent && connection.Type != domain.ConnectionLocal {
		return ports.ConnectionDiscovery{}, fmt.Errorf("unsupported SLURM connection type %q", connection.Type)
	}
	executor := d.executor
	if connection.Type == domain.ConnectionSSH {
		executor = runtimecommon.SSHCommandExecutor{
			Executor: d.executor, Endpoint: connection.Endpoint, Username: connection.Username,
			Port: configInt(connection.Configuration, "port"), IdentityFile: credentialFile(connection.CredentialRef),
			ProxyCommand: configString(connection.Configuration, "proxyCommand"), ForwardAgent: configBool(connection.Configuration, "forwardAgent", false),
		}
	}
	script := strings.Join([]string{
		`printf 'architecture='; uname -m`,
		`sinfo -h -o '%P|%a|%D|%c|%m|%l'`,
		`printf 'queueLength='; squeue -h | wc -l`,
		`(apptainer --version || singularity --version) 2>/dev/null | sed 's/^/containerRuntime=/'`,
		`df -Pk /home /tmp 2>/dev/null | awk 'NR>1 {printf "storage=%s|%s|%s\n", $6, $2, $4}'`,
	}, "; ")
	output, err := executor.Run(ctx, "/bin/sh", []string{"-c", script}, nil)
	if err != nil {
		return ports.ConnectionDiscovery{}, fmt.Errorf("SLURM discovery: %w", err)
	}
	metadata := map[string]any{"partitions": []string{}, "storage": []string{}}
	partitions, storage := []string{}, []string{}
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if found {
			switch key {
			case "architecture", "queueLength", "containerRuntime":
				metadata[key] = value
			case "storage":
				storage = append(storage, value)
			}
			continue
		}
		if strings.Count(line, "|") >= 5 {
			partitions = append(partitions, line)
		}
	}
	metadata["partitions"] = partitions
	metadata["storage"] = storage
	return ports.ConnectionDiscovery{Available: len(partitions) > 0, Metadata: metadata}, nil
}
