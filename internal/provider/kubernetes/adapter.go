package kubernetes

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/UFFeScience/akoflow/internal/domain"
	runtimecommon "github.com/UFFeScience/akoflow/internal/provider"
)

const (
	defaultObserverImage = "ghcr.io/uffescience/akoflow-observer:latest"
	activityContainer    = "activity"
	observerExecutable   = "/akoflow/observer/akoflow-observer"
	manifestPath         = "/tmp/akoflow/artifact-manifest.json"
	manifestLogPrefix    = "AKOFLOW_ARTIFACT_MANIFEST="
)

type Adapter struct {
	api           API
	namespace     string
	observerImage string
}

func New(api API, namespace string) *Adapter {
	if namespace == "" {
		namespace = "default"
	}
	return &Adapter{api: api, namespace: namespace, observerImage: defaultObserverImage}
}

func (a *Adapter) WithObserverImage(image string) *Adapter {
	if strings.TrimSpace(image) != "" {
		a.observerImage = image
	}
	return a
}

func (*Adapter) Modes() []domain.ExecutionMode {
	return []domain.ExecutionMode{domain.ExecutionModeReal, domain.ExecutionModeInteractive}
}

func (a *Adapter) Start(ctx context.Context, execution domain.ActivityExecutionContext) (domain.ActivityHandle, error) {
	if a.api == nil {
		return domain.ActivityHandle{}, fmt.Errorf("Kubernetes API client is required")
	}
	activity := execution.Activity
	if activity.Command.Image == "" {
		return domain.ActivityHandle{}, fmt.Errorf("activity image is required for Kubernetes")
	}
	name := kubernetesName("akoflow-" + execution.Run.ID + "-" + activity.ID)
	job, service, err := resources(
		name, a.namespace, a.observerImage, activity, execution.Resource, execution.Run.ID,
	)
	if err != nil {
		return domain.ActivityHandle{}, err
	}
	if err := a.api.Create(ctx, a.namespace, "jobs", job); err != nil {
		return domain.ActivityHandle{}, err
	}
	if service != nil {
		if err := a.api.Create(ctx, a.namespace, "services", service); err != nil {
			_ = a.api.Delete(ctx, a.namespace, "jobs", name)
			return domain.ActivityHandle{}, err
		}
	}
	return domain.ActivityHandle{ID: runtimecommon.NewID("activity"), RunID: execution.Run.ID,
		ActivityID: activity.ID, ResourceID: execution.Resource.ID,
		RuntimeID: execution.Resource.RuntimeID, ExternalID: name,
		Status: domain.HandleStarting, StartedAt: runtimecommon.UnixSeconds(time.Now()),
		Endpoints: serviceEndpoints(name, a.namespace, activity),
		Metadata: map[string]any{
			"artifactObservationDriver": "filesystem-diff",
			"artifactObservationRoot":   observationRoot(activity),
		}}, nil
}

func (a *Adapter) Inspect(ctx context.Context, handle domain.ActivityHandle) (domain.ActivityHandle, error) {
	output, err := a.api.Get(ctx, a.namespace, "jobs", handle.ExternalID)
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
	if handle.Status == domain.HandleCompleted || handle.Status == domain.HandleFailed {
		manifest, observationErr := a.collectArtifacts(ctx, handle.ExternalID)
		if observationErr != nil {
			if handle.Metadata == nil {
				handle.Metadata = make(map[string]any)
			}
			handle.Metadata["artifactObservationError"] = observationErr.Error()
		} else {
			handle.Artifacts = manifest
		}
	}
	return handle, nil
}

func (a *Adapter) collectArtifacts(ctx context.Context, jobName string) (*domain.ArtifactManifest, error) {
	payload, err := a.api.List(ctx, a.namespace, "pods", "job-name="+jobName)
	if err != nil {
		return nil, fmt.Errorf("list activity pods: %w", err)
	}
	var pods struct {
		Items []struct {
			Metadata struct {
				Name string `json:"name"`
			} `json:"metadata"`
		} `json:"items"`
	}
	if err := json.Unmarshal(payload, &pods); err != nil {
		return nil, fmt.Errorf("decode activity pods: %w", err)
	}
	if len(pods.Items) == 0 {
		return nil, fmt.Errorf("no pod found for job %q", jobName)
	}
	logs, err := a.api.Logs(ctx, a.namespace, pods.Items[0].Metadata.Name, activityContainer)
	if err != nil {
		return nil, fmt.Errorf("read artifact observer logs: %w", err)
	}
	for _, line := range strings.Split(string(logs), "\n") {
		prefixAt := strings.Index(line, manifestLogPrefix)
		if prefixAt < 0 {
			continue
		}
		encoded := strings.TrimSpace(line[prefixAt+len(manifestLogPrefix):])
		manifestJSON, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			return nil, fmt.Errorf("decode artifact manifest: %w", err)
		}
		var manifest domain.ArtifactManifest
		if err := json.Unmarshal(manifestJSON, &manifest); err != nil {
			return nil, fmt.Errorf("parse artifact manifest: %w", err)
		}
		return &manifest, nil
	}
	return nil, fmt.Errorf("artifact manifest was not published")
}

func (a *Adapter) Stop(ctx context.Context, handle domain.ActivityHandle) error {
	jobErr := ignoreNotFound(a.api.Delete(ctx, a.namespace, "jobs", handle.ExternalID))
	serviceErr := ignoreNotFound(a.api.Delete(ctx, a.namespace, "services", handle.ExternalID))
	return errors.Join(jobErr, serviceErr)
}

func resources(
	name string,
	namespace string,
	observerImage string,
	activity domain.Activity,
	resource domain.Resource,
	runID string,
) ([]byte, []byte, error) {
	podSpec := observedPodSpec(observerImage, activity, resource, runID)
	job := map[string]any{"apiVersion": "batch/v1", "kind": "Job",
		"metadata": map[string]any{"name": name, "namespace": namespace,
			"labels": map[string]string{"app.kubernetes.io/managed-by": "akoflow"}},
		"spec": map[string]any{"backoffLimit": max(activity.Policy.MaxAttempts-1, 0),
			"template": map[string]any{
				"metadata": map[string]any{"labels": map[string]string{"akoflow.io/activity": activity.ID}},
				"spec":     podSpec,
			}}}
	jobJSON, err := json.Marshal(job)
	if err != nil {
		return nil, nil, err
	}
	if activity.Service == nil || len(activity.Service.Ports) == 0 {
		return jobJSON, nil, nil
	}
	ports := make([]map[string]any, 0, len(activity.Service.Ports))
	for _, port := range activity.Service.Ports {
		ports = append(ports, map[string]any{"port": port, "targetPort": port})
	}
	service := map[string]any{"apiVersion": "v1", "kind": "Service",
		"metadata": map[string]any{"name": name, "namespace": namespace,
			"labels": map[string]string{"app.kubernetes.io/managed-by": "akoflow"}},
		"spec": map[string]any{"selector": map[string]string{"akoflow.io/activity": activity.ID}, "ports": ports}}
	serviceJSON, err := json.Marshal(service)
	return jobJSON, serviceJSON, err
}

func observedPodSpec(
	observerImage string,
	activity domain.Activity,
	resource domain.Resource,
	runID string,
) map[string]any {
	environment := make([]map[string]string, 0, len(activity.Command.Environment))
	for key, value := range activity.Command.Environment {
		environment = append(environment, map[string]string{"name": key, "value": value})
	}
	environment = append(environment, map[string]string{"name": "AKOFLOW_ARTIFACT_MANIFEST", "value": manifestPath})
	observerArguments := []string{
		"run", "--run-id", runID, "--activity-id", activity.ID,
		"--attempt", "1", "--runtime", "kubernetes", "--root", observationRoot(activity),
		"--manifest", manifestPath, "--", activity.Command.Entrypoint,
	}
	observerArguments = append(observerArguments, activity.Command.Arguments...)
	volumeMounts := []map[string]any{
		{"name": "akoflow-observer", "mountPath": "/akoflow/observer", "readOnly": true},
	}
	container := map[string]any{"name": activityContainer, "image": activity.Command.Image,
		"command": []string{observerExecutable}, "args": observerArguments,
		"env": environment, "workingDir": activity.Command.WorkingDirectory,
		"volumeMounts": volumeMounts,
		"resources": map[string]any{"requests": map[string]string{
			"cpu":    fmt.Sprintf("%g", activity.Resources.CPU),
			"memory": fmt.Sprintf("%d", activity.Resources.MemoryBytes)}}}
	podSpec := map[string]any{
		"restartPolicy": "Never",
		"containers":    []any{container},
		"volumes": []map[string]any{
			{"name": "akoflow-observer", "image": map[string]any{
				"reference": observerImage, "pullPolicy": "IfNotPresent",
			}},
		},
	}
	if resource.Type == domain.ResourceKubernetesMachine && resource.ProviderID != "" {
		podSpec["nodeSelector"] = map[string]string{
			"kubernetes.io/hostname": resource.ProviderID,
		}
	}
	return podSpec
}

func observationRoot(activity domain.Activity) string {
	if configured, ok := activity.Metadata["artifactObservationRoot"].(string); ok && configured != "" {
		return configured
	}
	return "."
}

func ignoreNotFound(err error) error {
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	return err
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
