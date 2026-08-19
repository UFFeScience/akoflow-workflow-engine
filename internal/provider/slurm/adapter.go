package slurm

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/UFFeScience/akoflow/internal/domain"
	runtimecommon "github.com/UFFeScience/akoflow/internal/provider"
	"github.com/UFFeScience/akoflow/internal/provider/local"
)

type Adapter struct {
	executor        runtimecommon.CommandExecutor
	partition       string
	direct          *local.Adapter
	scriptDirectory string
}

type Config struct {
	Partition       string
	ScriptDirectory string
}

func New(executor runtimecommon.CommandExecutor, partition string) *Adapter {
	return NewWithConfig(executor, Config{Partition: partition, ScriptDirectory: "storage/slurm/scripts"})
}

func NewWithConfig(executor runtimecommon.CommandExecutor, config Config) *Adapter {
	return &Adapter{executor: executor, partition: config.Partition, direct: local.New(),
		scriptDirectory: config.ScriptDirectory}
}

func (*Adapter) Modes() []domain.ExecutionMode {
	return []domain.ExecutionMode{domain.ExecutionModeReal}
}

func (a *Adapter) Start(ctx context.Context, execution domain.ActivityExecutionContext) (domain.ActivityHandle, error) {
	if execution.Resource.ExecutionTarget == domain.ExecutionTargetDirect {
		return a.startDirect(ctx, execution)
	}
	if a.executor == nil {
		return domain.ActivityHandle{}, fmt.Errorf("slurm command executor is required")
	}
	script, err := batchScript(execution.Activity, a.partition)
	if err != nil {
		return domain.ActivityHandle{}, err
	}
	scriptPath, err := a.saveScript(execution.Run.ID, execution.Activity.ID, ".sbatch", script)
	if err != nil {
		return domain.ActivityHandle{}, err
	}
	output, err := a.executor.Run(ctx, "sbatch", []string{"--parsable", scriptPath}, nil)
	if err != nil {
		return domain.ActivityHandle{}, err
	}
	jobID := strings.Split(strings.TrimSpace(string(output)), ";")[0]
	if _, err := strconv.ParseUint(jobID, 10, 64); err != nil {
		return domain.ActivityHandle{}, fmt.Errorf("invalid Slurm job id %q", jobID)
	}
	return domain.ActivityHandle{ID: runtimecommon.NewID("activity"), RunID: execution.Run.ID,
		ActivityID: execution.Activity.ID, ResourceID: execution.Resource.ID,
		RuntimeID: execution.Resource.RuntimeID, ExternalID: jobID,
		Status: domain.HandleStarting, StartedAt: runtimecommon.UnixSeconds(time.Now()),
		Metadata: map[string]any{"executionTarget": string(domain.ExecutionTargetBatch), "scriptPath": scriptPath}}, nil
}

func (a *Adapter) Inspect(ctx context.Context, handle domain.ActivityHandle) (domain.ActivityHandle, error) {
	if handle.Metadata["executionTarget"] == string(domain.ExecutionTargetDirect) {
		return a.direct.Inspect(ctx, handle)
	}
	output, err := a.executor.Run(ctx, "sacct", []string{"-j", handle.ExternalID,
		"--noheader", "--parsable2", "--format=State,ExitCode"}, nil)
	if err != nil {
		return handle, err
	}
	line := strings.TrimSpace(strings.Split(string(output), "\n")[0])
	fields := strings.Split(line, "|")
	if len(fields) == 0 || fields[0] == "" {
		return handle, nil
	}
	state := strings.Split(fields[0], "+")[0]
	switch state {
	case "PENDING", "CONFIGURING":
		handle.Status = domain.HandleStarting
	case "RUNNING", "COMPLETING":
		handle.Status = domain.HandleRunning
	case "COMPLETED":
		handle.Status = domain.HandleCompleted
	case "CANCELLED":
		handle.Status = domain.HandleStopped
	default:
		handle.Status = domain.HandleFailed
		handle.Failure = "Slurm state: " + state
	}
	if len(fields) > 1 {
		codeText := strings.Split(fields[1], ":")[0]
		if code, parseErr := strconv.Atoi(codeText); parseErr == nil {
			handle.ExitCode = &code
		}
	}
	if handle.Status == domain.HandleCompleted || handle.Status == domain.HandleFailed || handle.Status == domain.HandleStopped {
		handle.FinishedAt = runtimecommon.UnixSeconds(time.Now())
	}
	return handle, nil
}

func (a *Adapter) Stop(ctx context.Context, handle domain.ActivityHandle) error {
	if handle.Metadata["executionTarget"] == string(domain.ExecutionTargetDirect) {
		return a.direct.Stop(ctx, handle)
	}
	_, err := a.executor.Run(ctx, "scancel", []string{handle.ExternalID}, nil)
	return err
}

func (a *Adapter) startDirect(ctx context.Context, execution domain.ActivityExecutionContext) (domain.ActivityHandle, error) {
	activity := execution.Activity
	script, err := directScript(activity)
	if err != nil {
		return domain.ActivityHandle{}, err
	}
	scriptPath, err := a.saveScript(execution.Run.ID, activity.ID, ".sh", script)
	if err != nil {
		return domain.ActivityHandle{}, err
	}
	execution.Activity.Command = domain.ActivityCommand{Entrypoint: "/bin/sh", Arguments: []string{scriptPath}}
	handle, err := a.direct.Start(ctx, execution)
	if err != nil {
		return handle, err
	}
	if handle.Metadata == nil {
		handle.Metadata = make(map[string]any)
	}
	handle.Metadata["executionTarget"] = string(domain.ExecutionTargetDirect)
	handle.Metadata["slurmSubmission"] = "login-node"
	handle.Metadata["scriptPath"] = scriptPath
	return handle, nil
}

func (a *Adapter) saveScript(runID, activityID, extension, content string) (string, error) {
	if a.scriptDirectory == "" {
		return "", fmt.Errorf("slurm script directory is required")
	}
	directory := filepath.Join(a.scriptDirectory, shellToken(runID), shellToken(activityID))
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return "", fmt.Errorf("create Slurm script directory: %w", err)
	}
	path := filepath.Join(directory, fmt.Sprintf("%d%s", time.Now().UnixNano(), extension))
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return "", fmt.Errorf("save Slurm script: %w", err)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve Slurm script path: %w", err)
	}
	return abs, nil
}

func directScript(activity domain.Activity) (string, error) {
	if activity.Command.Entrypoint == "" {
		return "", fmt.Errorf("activity entrypoint is required")
	}
	var script strings.Builder
	script.WriteString("#!/bin/sh\nset -eu\n")
	writeActivityCommand(&script, activity)
	return script.String(), nil
}

func batchScript(activity domain.Activity, partition string) (string, error) {
	if activity.Command.Entrypoint == "" {
		return "", fmt.Errorf("activity entrypoint is required")
	}
	var script strings.Builder
	script.WriteString("#!/bin/sh\n")
	script.WriteString("#SBATCH --job-name=")
	script.WriteString(shellToken("akoflow-" + activity.ID))
	script.WriteByte('\n')
	if partition != "" {
		script.WriteString("#SBATCH --partition=")
		script.WriteString(shellToken(partition))
		script.WriteByte('\n')
	}
	if activity.Resources.CPU > 0 {
		script.WriteString(fmt.Sprintf("#SBATCH --cpus-per-task=%d\n", int(activity.Resources.CPU+0.999)))
	}
	if activity.Resources.MemoryBytes > 0 {
		script.WriteString(fmt.Sprintf("#SBATCH --mem=%dM\n", (activity.Resources.MemoryBytes+(1<<20)-1)/(1<<20)))
	}
	script.WriteString("set -eu\n")
	writeActivityCommand(&script, activity)
	return script.String(), nil
}

func writeActivityCommand(script *strings.Builder, activity domain.Activity) {
	for key, value := range activity.Command.Environment {
		script.WriteString("export ")
		script.WriteString(shellToken(key))
		script.WriteByte('=')
		script.WriteString(shellQuote(value))
		script.WriteByte('\n')
	}
	if activity.Command.WorkingDirectory != "" {
		script.WriteString("cd ")
		script.WriteString(shellQuote(activity.Command.WorkingDirectory))
		script.WriteByte('\n')
	}
	if activity.Command.Image != "" {
		script.WriteString("singularity exec ")
		script.WriteString(shellQuote(activity.Command.Image))
		script.WriteByte(' ')
	}
	script.WriteString(shellQuote(activity.Command.Entrypoint))
	for _, argument := range activity.Command.Arguments {
		script.WriteByte(' ')
		script.WriteString(shellQuote(argument))
	}
	script.WriteByte('\n')
}

func shellToken(value string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("._-", r) {
			return r
		}
		return '-'
	}, value)
}
func shellQuote(value string) string { return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'" }
