package slurm

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/UFFeScience/akoflow/internal/application/ports"
	"github.com/UFFeScience/akoflow/internal/domain"
	runtimecommon "github.com/UFFeScience/akoflow/internal/provider"
)

// Discovery collects a bounded inventory without connecting to compute nodes.
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
		executor = runtimecommon.SSHCommandExecutor{Executor: d.executor, Endpoint: connection.Endpoint,
			Username: connection.Username, Port: configInt(connection.Configuration, "port"),
			IdentityFile: credentialFile(connection.CredentialRef), ProxyCommand: configString(connection.Configuration, "proxyCommand"),
			ForwardAgent: configBool(connection.Configuration, "forwardAgent", false)}
	}
	script := strings.Join([]string{
		`printf 'FACT|architecture|'; uname -m`,
		`printf 'FACT|hostname|'; hostname -f 2>/dev/null || hostname`,
		`printf 'FACT|kernel|'; uname -sr`,
		`printf 'FACT|os|'; (awk -F= '$1=="PRETTY_NAME" {gsub(/^"|"$/, "", $2); print $2}' /etc/os-release 2>/dev/null || true)`,
		`printf 'FACT|workingDirectory|'; pwd -P`,
		`printf 'FACT|homeDirectory|'; printf '%s\n' "$HOME"`,
		`printf 'FACT|temporaryDirectory|'; printf '%s\n' "${TMPDIR:-/tmp}"`,
		`printf 'FACT|slurmVersion|'; (sinfo --version 2>/dev/null || true)`,
		`printf 'FACT|queueLength|'; squeue -h 2>/dev/null | wc -l | tr -d ' '`,
		`(apptainer --version || singularity --version) 2>/dev/null | sed 's/^/FACT|containerRuntime|/'`,
		`sinfo -h -o 'PARTITION|%P|%a|%D|%c|%m|%l'`,
		`sinfo -h -N -o 'NODE|%P|%N|%T|%c|%m|%d|%w|%f|%G|%E'`,
		`df -PkT "$HOME" "${TMPDIR:-/tmp}" "$(pwd -P)" 2>/dev/null | awk 'NR>1 {printf "FILESYSTEM|%s|%s|%s|%s|%s|%s\n", $1, $2, $3, $4, $5, $7}' | sort -u`,
	}, "; ")
	output, err := executor.Run(ctx, "/bin/sh", []string{"-c", script}, nil)
	if err != nil {
		return ports.ConnectionDiscovery{}, fmt.Errorf("SLURM discovery: %w", err)
	}
	return parseDiscovery(output), nil
}

func parseDiscovery(output []byte) ports.ConnectionDiscovery {
	metadata := map[string]any{}
	partitions := []map[string]any{}
	nodesByName := map[string]map[string]any{}
	nodeOrder := []string{}
	filesystems := []map[string]any{}
	warnings := []string{}
	for _, raw := range strings.Split(string(output), "\n") {
		fields := strings.Split(strings.TrimSpace(raw), "|")
		if len(fields) == 0 || fields[0] == "" {
			continue
		}
		switch fields[0] {
		case "FACT":
			if len(fields) >= 3 {
				metadata[fields[1]] = strings.Join(fields[2:], "|")
			}
		case "PARTITION":
			if len(fields) < 7 {
				warnings = append(warnings, "ignored malformed partition record")
				continue
			}
			partitions = append(partitions, map[string]any{"name": cleanPartition(fields[1]),
				"default": strings.HasSuffix(fields[1], "*"), "availability": fields[2], "nodeCount": integer(fields[3]),
				"cpuCoresPerNode": integer(fields[4]), "memoryMiBPerNode": integer(fields[5]), "timeLimit": fields[6]})
		case "NODE":
			if len(fields) < 11 {
				warnings = append(warnings, "ignored malformed node record")
				continue
			}
			name, partition := fields[2], cleanPartition(fields[1])
			node := nodesByName[name]
			if node == nil {
				node = map[string]any{"name": name, "state": fields[3], "cpuCores": integer(fields[4]),
					"memoryMiB": integer(fields[5]), "temporaryDiskMiB": integer(fields[6]), "weight": integer(fields[7]),
					"features": csv(fields[8]), "gres": csv(fields[9]), "reason": emptyIfNone(fields[10]), "partitions": []string{}}
				nodesByName[name], nodeOrder = node, append(nodeOrder, name)
			}
			node["partitions"] = appendUnique(node["partitions"].([]string), partition)
		case "FILESYSTEM":
			if len(fields) >= 7 {
				filesystems = append(filesystems, map[string]any{"device": fields[1], "type": fields[2],
					"capacityKiB": integer(fields[3]), "usedKiB": integer(fields[4]), "availableKiB": integer(fields[5]), "mountPoint": fields[6]})
			}
		}
	}
	nodes := make([]map[string]any, 0, len(nodeOrder))
	for _, name := range nodeOrder {
		nodes = append(nodes, nodesByName[name])
	}
	metadata["partitions"], metadata["nodes"], metadata["filesystems"] = partitions, nodes, filesystems
	return ports.ConnectionDiscovery{Available: len(partitions) > 0, Metadata: metadata, Warnings: warnings}
}

func cleanPartition(value string) string { return strings.TrimSuffix(strings.TrimSpace(value), "*") }
func integer(value string) int64 {
	parsed, _ := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	return parsed
}
func csv(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" || value == "(null)" || value == "N/A" {
		return []string{}
	}
	return strings.Split(value, ",")
}
func emptyIfNone(value string) string {
	if value == "none" || value == "(null)" {
		return ""
	}
	return value
}
func appendUnique(values []string, candidate string) []string {
	for _, value := range values {
		if value == candidate {
			return values
		}
	}
	return append(values, candidate)
}
