package local

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/UFFeScience/akoflow/internal/domain"
	runtimecommon "github.com/UFFeScience/akoflow/internal/provider"
)

type processResult struct {
	exitCode   int
	err        error
	finishedAt float64
}

type Adapter struct {
	mu      sync.RWMutex
	results map[string]processResult
}

func New() *Adapter { return &Adapter{results: make(map[string]processResult)} }
func (*Adapter) Modes() []domain.ExecutionMode {
	return []domain.ExecutionMode{domain.ExecutionModeReal, domain.ExecutionModeInteractive}
}

func (a *Adapter) Start(_ context.Context, execution domain.ActivityExecutionContext) (domain.ActivityHandle, error) {
	activity := execution.Activity
	if activity.Command.Entrypoint == "" {
		return domain.ActivityHandle{}, fmt.Errorf("activity entrypoint is required")
	}
	command := exec.Command(activity.Command.Entrypoint, activity.Command.Arguments...)
	command.Dir = activity.Command.WorkingDirectory
	command.Env = os.Environ()
	for key, value := range activity.Command.Environment {
		command.Env = append(command.Env, key+"="+value)
	}
	if err := command.Start(); err != nil {
		return domain.ActivityHandle{}, fmt.Errorf("start local activity: %w", err)
	}
	handle := domain.ActivityHandle{ID: runtimecommon.NewID("activity"), RunID: execution.Run.ID,
		ActivityID: activity.ID, ResourceID: execution.Resource.ID,
		RuntimeID: execution.RuntimeID, ExternalID: strconv.Itoa(command.Process.Pid),
		Status: domain.HandleRunning, StartedAt: runtimecommon.UnixSeconds(time.Now()),
		Endpoints: localEndpoints(activity)}
	go func(id string) {
		err := command.Wait()
		exitCode := command.ProcessState.ExitCode()
		a.mu.Lock()
		a.results[id] = processResult{exitCode: exitCode, err: err, finishedAt: runtimecommon.UnixSeconds(time.Now())}
		a.mu.Unlock()
	}(handle.ID)
	return handle, nil
}

func (a *Adapter) Inspect(_ context.Context, handle domain.ActivityHandle) (domain.ActivityHandle, error) {
	a.mu.RLock()
	result, done := a.results[handle.ID]
	a.mu.RUnlock()
	if done {
		handle.FinishedAt = result.finishedAt
		handle.ExitCode = &result.exitCode
		if result.err != nil {
			handle.Status = domain.HandleFailed
			handle.Failure = result.err.Error()
		} else {
			handle.Status = domain.HandleCompleted
		}
		return handle, nil
	}
	pid, err := strconv.Atoi(handle.ExternalID)
	if err != nil {
		return handle, fmt.Errorf("invalid local process id: %w", err)
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return handle, err
	}
	if err := process.Signal(syscall.Signal(0)); err != nil {
		handle.Status = domain.HandleFailed
		handle.Failure = "process is no longer available"
	}
	return handle, nil
}

func (*Adapter) Stop(_ context.Context, handle domain.ActivityHandle) error {
	pid, err := strconv.Atoi(handle.ExternalID)
	if err != nil {
		return fmt.Errorf("invalid local process id: %w", err)
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return process.Signal(os.Interrupt)
}

func localEndpoints(activity domain.Activity) []string {
	if activity.Service == nil {
		return nil
	}
	result := make([]string, 0, len(activity.Service.Ports))
	for _, port := range activity.Service.Ports {
		result = append(result, fmt.Sprintf("tcp://127.0.0.1:%d", port))
	}
	return result
}
