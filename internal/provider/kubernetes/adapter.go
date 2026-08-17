package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/UFFeScience/akoflow/internal/domain"
	runtimecommon "github.com/UFFeScience/akoflow/internal/provider"
)

type Adapter struct {
	executor  runtimecommon.CommandExecutor
	namespace string
}

func New(executor runtimecommon.CommandExecutor, namespace string) *Adapter {
	if namespace == "" {
		namespace = "default"
	}
	return &Adapter{executor: executor, namespace: namespace}
}

func (*Adapter) Modes() []domain.ExecutionMode {
	return []domain.ExecutionMode{domain.ExecutionModeReal, domain.ExecutionModeInteractive}
}

func (a *Adapter) Start(ctx context.Context, execution domain.ActivityExecutionContext) (domain.ActivityHandle, error) {
	if a.executor == nil {
		return domain.ActivityHandle{}, fmt.Errorf("kubernetes command executor is required")
	}
	activity := execution.Activity
	if activity.Command.Image == "" {
		return domain.ActivityHandle{}, fmt.Errorf("activity image is required for Kubernetes")
	}
	name := kubernetesName("akoflow-" + execution.Run.ID + "-" + activity.ID)
	manifest, err := jobManifest(name, a.namespace, activity)
	if err != nil {
		return domain.ActivityHandle{}, err
	}
	if _, err := a.executor.Run(ctx, "kubectl", []string{"apply", "-f", "-"}, manifest); err != nil {
		return domain.ActivityHandle{}, err
	}
	return domain.ActivityHandle{ID: runtimecommon.NewID("activity"), RunID: execution.Run.ID,
		ActivityID: activity.ID, ResourceID: execution.Resource.ID,
		RuntimeID: execution.Resource.RuntimeID, ExternalID: name,
		Status: domain.HandleStarting, StartedAt: runtimecommon.UnixSeconds(time.Now()),
		Endpoints: serviceEndpoints(name, a.namespace, activity)}, nil
}

func (a *Adapter) Inspect(ctx context.Context, handle domain.ActivityHandle) (domain.ActivityHandle, error) {
	output, err := a.executor.Run(ctx, "kubectl", []string{"get", "job", handle.ExternalID,
		"-n", a.namespace, "-o", "json"}, nil)
	if err != nil {
		return handle, err
	}
	var job struct {
		Status struct {
			Active, Succeeded, Failed int
			Conditions                []struct{ Type, Status, Message string }
		}
	}
	if err := json.Unmarshal(output, &job); err != nil {
		return handle, fmt.Errorf("decode Kubernetes job: %w", err)
	}
	switch {
	case job.Status.Succeeded > 0:
		handle.Status = domain.HandleCompleted
	case job.Status.Failed > 0:
		handle.Status = domain.HandleFailed
	case job.Status.Active > 0:
		handle.Status = domain.HandleRunning
	default:
		handle.Status = domain.HandleStarting
	}
	if handle.Status == domain.HandleCompleted || handle.Status == domain.HandleFailed {
		handle.FinishedAt = runtimecommon.UnixSeconds(time.Now())
	}
	for _, condition := range job.Status.Conditions {
		if condition.Type == "Failed" && condition.Status == "True" {
			handle.Failure = condition.Message
		}
	}
	return handle, nil
}

func (a *Adapter) Stop(ctx context.Context, handle domain.ActivityHandle) error {
	_, err := a.executor.Run(ctx, "kubectl", []string{"delete", "job,service", handle.ExternalID,
		"-n", a.namespace, "--ignore-not-found=true"}, nil)
	return err
}

func jobManifest(name, namespace string, activity domain.Activity) ([]byte, error) {
	environment := make([]map[string]string, 0, len(activity.Command.Environment))
	for key, value := range activity.Command.Environment {
		environment = append(environment, map[string]string{"name": key, "value": value})
	}
	container := map[string]any{"name": "activity", "image": activity.Command.Image,
		"command": []string{activity.Command.Entrypoint}, "args": activity.Command.Arguments,
		"env": environment, "workingDir": activity.Command.WorkingDirectory,
		"resources": map[string]any{"requests": map[string]string{
			"cpu":    fmt.Sprintf("%g", activity.Resources.CPU),
			"memory": fmt.Sprintf("%d", activity.Resources.MemoryBytes)}}}
	job := map[string]any{"apiVersion": "batch/v1", "kind": "Job",
		"metadata": map[string]any{"name": name, "namespace": namespace,
			"labels": map[string]string{"app.kubernetes.io/managed-by": "akoflow"}},
		"spec": map[string]any{"backoffLimit": max(activity.Policy.MaxAttempts-1, 0),
			"template": map[string]any{"metadata": map[string]any{"labels": map[string]string{"akoflow.io/activity": activity.ID}},
				"spec": map[string]any{"restartPolicy": "Never", "containers": []any{container}}}}}
	items := []any{job}
	if activity.Service != nil && len(activity.Service.Ports) > 0 {
		ports := make([]map[string]any, 0, len(activity.Service.Ports))
		for _, port := range activity.Service.Ports {
			ports = append(ports, map[string]any{"port": port, "targetPort": port})
		}
		items = append(items, map[string]any{"apiVersion": "v1", "kind": "Service",
			"metadata": map[string]any{"name": name, "namespace": namespace,
				"labels": map[string]string{"app.kubernetes.io/managed-by": "akoflow"}},
			"spec": map[string]any{"selector": map[string]string{"akoflow.io/activity": activity.ID}, "ports": ports}})
	}
	return json.Marshal(map[string]any{"apiVersion": "v1", "kind": "List", "items": items})
}

func kubernetesName(value string) string {
	value = strings.ToLower(value)
	value = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' {
			return r
		}
		return '-'
	}, value)
	value = strings.Trim(value, "-")
	if len(value) > 63 {
		value = value[:63]
	}
	return strings.Trim(value, "-")
}

func serviceEndpoints(name, namespace string, activity domain.Activity) []string {
	if activity.Service == nil {
		return nil
	}
	result := make([]string, 0, len(activity.Service.Ports))
	for _, port := range activity.Service.Ports {
		result = append(result, fmt.Sprintf("tcp://%s.%s.svc:%d", name, namespace, port))
	}
	return result
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
