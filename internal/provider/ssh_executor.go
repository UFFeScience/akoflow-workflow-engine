package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// SSHCommandExecutor runs a command on a login node while preserving the
// CommandExecutor contract used by the SLURM adapter. Authentication is left
// to SSH configuration, an agent, or a configured identity file; no secret is
// stored in the environment definition.
type SSHCommandExecutor struct {
	Executor     CommandExecutor
	Endpoint     string
	Username     string
	Port         int
	IdentityFile string
	ProxyCommand string
	HostKeyAlias string
	ForwardAgent bool
}

func (e SSHCommandExecutor) Run(ctx context.Context, name string, args []string, input []byte) ([]byte, error) {
	if e.Executor == nil {
		return nil, fmt.Errorf("SSH command executor is required")
	}
	if strings.TrimSpace(e.Endpoint) == "" {
		return nil, fmt.Errorf("SSH endpoint is required")
	}
	target := strings.TrimSpace(e.Endpoint)
	if e.Username != "" && !strings.Contains(target, "@") {
		target = e.Username + "@" + target
	}
	// Health checks and workers must never block indefinitely on an unreachable
	// login node or a ProxyCommand that keeps a pipe open after its parent dies.
	sshArgs := make([]string, 0, len(args)+14)
	sshArgs = append(sshArgs, "-o", "BatchMode=yes", "-o", "ConnectionAttempts=1", "-o", "ConnectTimeout=10", "-o", "CheckHostIP=no")
	if e.Port > 0 {
		sshArgs = append(sshArgs, "-p", strconv.Itoa(e.Port))
	}
	if e.IdentityFile != "" {
		sshArgs = append(sshArgs, "-i", e.IdentityFile)
	}
	if e.ProxyCommand != "" {
		sshArgs = append(sshArgs, "-o", "ProxyCommand="+e.ProxyCommand)
	}
	if e.HostKeyAlias != "" {
		sshArgs = append(sshArgs, "-o", "HostKeyAlias="+e.HostKeyAlias)
	}
	if e.ForwardAgent {
		sshArgs = append(sshArgs, "-A")
	}
	sshArgs = append(sshArgs, target, "--", name)
	sshArgs = append(sshArgs, args...)
	return e.Executor.Run(ctx, "ssh", sshArgs, input)
}
