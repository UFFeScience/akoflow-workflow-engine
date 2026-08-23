package slurm

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"

	"github.com/UFFeScience/akoflow/internal/application/ports"
	"github.com/UFFeScience/akoflow/internal/domain"
	"github.com/creack/pty"
)

var safeSchedulerTarget = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// TerminalRunner creates an actual interactive shell. SSH resources receive a
// forced TTY and scheduler resources are allocated through srun --pty.
type TerminalRunner struct{}

var _ ports.InteractiveConsoleRunner = TerminalRunner{}

func (TerminalRunner) StartInteractive(_ context.Context, connection domain.EnvironmentConnection, resource domain.Resource) (ports.InteractiveTerminal, error) {
	command, err := interactiveCommand(connection, resource)
	if err != nil {
		return nil, err
	}
	terminal, err := pty.Start(command)
	if err != nil {
		return nil, fmt.Errorf("start interactive terminal: %w", err)
	}
	return &terminalHandle{file: terminal, command: command}, nil
}

func interactiveCommand(connection domain.EnvironmentConnection, resource domain.Resource) (*exec.Cmd, error) {
	if connection.Type == domain.ConnectionLocal || connection.Type == domain.ConnectionAgent {
		return exec.Command("/bin/sh", "-l"), nil
	}
	if connection.Type == domain.ConnectionKubernetes {
		container, _ := resource.Metadata["interactiveDockerContainer"].(string)
		if container == "" {
			return nil, fmt.Errorf("interactive Kubernetes terminals are currently available only for discovered Kind control-plane resources")
		}
		return exec.Command("docker", "exec", "-it", container, "/bin/sh", "-l"), nil
	}
	if connection.Type != domain.ConnectionSSH {
		return nil, fmt.Errorf("interactive terminals are not supported for connection type %q", connection.Type)
	}
	args := []string{"-tt", "-o", "BatchMode=yes", "-o", "ConnectionAttempts=1", "-o", "ConnectTimeout=10", "-o", "CheckHostIP=no"}
	if port := configInt(connection.Configuration, "port"); port > 0 {
		args = append(args, "-p", fmt.Sprintf("%d", port))
	}
	if identityFile := credentialFile(connection.CredentialRef); identityFile != "" {
		args = append(args, "-i", identityFile)
	}
	if proxy := configString(connection.Configuration, "proxyCommand"); proxy != "" {
		args = append(args, "-o", "ProxyCommand="+proxy)
	}
	if alias := configString(connection.Configuration, "hostKeyAlias"); alias != "" {
		args = append(args, "-o", "HostKeyAlias="+alias)
	}
	if configBool(connection.Configuration, "forwardAgent", false) {
		args = append(args, "-A")
	}
	target := strings.TrimSpace(connection.Endpoint)
	if username := strings.TrimSpace(connection.Username); username != "" {
		target = username + "@" + target
	}
	if target == "" {
		return nil, fmt.Errorf("SSH endpoint is required for an interactive terminal")
	}
	args = append(args, target)
	if resource.ExecutionTarget == domain.ExecutionTargetDirect {
		args = append(args, "/bin/sh", "-l")
		return exec.Command("ssh", args...), nil
	}
	if !safeSchedulerTarget.MatchString(resource.ProviderID) {
		return nil, fmt.Errorf("invalid scheduler target %q", resource.ProviderID)
	}
	if resource.Type == domain.ResourceHPCPartition {
		args = append(args, "srun", "--partition="+resource.ProviderID, "--pty", "/bin/bash", "-l")
	} else if resource.Type == domain.ResourceHPCMachine {
		args = append(args, "srun", "--nodelist="+resource.ProviderID, "--pty", "/bin/bash", "-l")
	} else {
		return nil, fmt.Errorf("resource type %q cannot host an interactive terminal", resource.Type)
	}
	return exec.Command("ssh", args...), nil
}

type terminalHandle struct {
	file    *os.File
	command *exec.Cmd
	once    sync.Once
}

func (h *terminalHandle) Read(buffer []byte) (int, error)  { return h.file.Read(buffer) }
func (h *terminalHandle) Write(buffer []byte) (int, error) { return h.file.Write(buffer) }
func (h *terminalHandle) Resize(rows, columns uint16) error {
	return pty.Setsize(h.file, &pty.Winsize{Rows: rows, Cols: columns})
}
func (h *terminalHandle) Close() (err error) {
	h.once.Do(func() {
		_ = h.file.Close()
		if h.command.Process != nil {
			_ = h.command.Process.Kill()
		}
		err = h.command.Wait()
	})
	return err
}
