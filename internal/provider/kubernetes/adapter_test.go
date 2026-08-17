package kubernetes

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/UFFeScience/akoflow/internal/domain"
)

type apiFake struct {
	created    map[string][]byte
	getOutput  []byte
	listOutput []byte
	logsOutput []byte
}

func (f *apiFake) Create(_ context.Context, _, resource string, body []byte) error {
	if f.created == nil {
		f.created = map[string][]byte{}
	}
	f.created[resource] = body
	return nil
}
func (f *apiFake) Get(context.Context, string, string, string) ([]byte, error) {
	return f.getOutput, nil
}
func (f *apiFake) List(context.Context, string, string, string) ([]byte, error) {
	return f.listOutput, nil
}
func (f *apiFake) Logs(context.Context, string, string, string) ([]byte, error) {
	return f.logsOutput, nil
}
func (f *apiFake) Delete(context.Context, string, string, string) error { return nil }

func TestAdapterCreatesJobAndServiceFromActivity(t *testing.T) {
	api := &apiFake{}
	adapter := New(api, "science")
	activity := domain.Activity{
		ID: "A 1",
		Command: domain.ActivityCommand{
			Image: "image:1", Entrypoint: "python", Arguments: []string{"run.py"},
		},
		Resources: domain.ActivityResources{CPU: 2, MemoryBytes: 1048576},
		Service:   &domain.ServiceSpec{Ports: []int{8080}},
	}
	handle, err := adapter.Start(context.Background(), domain.ActivityExecutionContext{
		Run: domain.ExecutionRun{ID: "run"}, Activity: activity,
		Resource: domain.Resource{ID: "node", RuntimeID: "kubernetes",
			Type: domain.ResourceKubernetesMachine, ProviderID: "kind-worker"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if handle.Status != domain.HandleStarting || len(handle.Endpoints) != 1 {
		t.Fatalf("handle=%+v", handle)
	}
	if len(api.created) != 2 {
		t.Fatalf("created resources=%d", len(api.created))
	}
	var job map[string]any
	if err := json.Unmarshal(api.created["jobs"], &job); err != nil {
		t.Fatal(err)
	}
	template := job["spec"].(map[string]any)["template"].(map[string]any)
	podSpec := template["spec"].(map[string]any)
	selector := podSpec["nodeSelector"].(map[string]any)
	if selector["kubernetes.io/hostname"] != "kind-worker" {
		t.Fatalf("node selector=%v", selector)
	}
	containers := podSpec["containers"].([]any)
	if len(containers) != 1 {
		t.Fatalf("activity must be the only container: %v", podSpec)
	}
	if _, exists := podSpec["initContainers"]; exists {
		t.Fatalf("observer must not require an init container: %v", podSpec)
	}
	activityContainer := containers[0].(map[string]any)
	command := activityContainer["command"].([]any)
	if command[0] != observerExecutable {
		t.Fatalf("activity command=%v", command)
	}
	volumes := podSpec["volumes"].([]any)
	observerVolume := volumes[0].(map[string]any)
	if _, exists := observerVolume["image"]; !exists {
		t.Fatalf("observer must be mounted as an OCI image volume: %v", volumes)
	}
}

func TestAdapterMapsKubernetesStatus(t *testing.T) {
	manifest := []byte(`{"schemaVersion":1,"runId":"run","activityId":"activity","files":[{"path":"result.csv","change":"created","sizeBytes":42}]}`)
	api := &apiFake{
		getOutput:  []byte(`{"status":{"succeeded":1}}`),
		listOutput: []byte(`{"items":[{"metadata":{"name":"job-pod"}}]}`),
		logsOutput: []byte(manifestLogPrefix + base64.StdEncoding.EncodeToString(manifest)),
	}
	handle, err := New(api, "default").Inspect(context.Background(), domain.ActivityHandle{
		ExternalID: "job", Status: domain.HandleRunning,
	})
	if err != nil || handle.Status != domain.HandleCompleted || handle.Artifacts == nil {
		t.Fatalf("handle=%+v err=%v", handle, err)
	}
	if len(handle.Artifacts.Files) != 1 || handle.Artifacts.Files[0].SizeBytes != 42 {
		t.Fatalf("artifacts=%+v", handle.Artifacts)
	}
}

func TestObservationFailureDoesNotChangeExecutionStatus(t *testing.T) {
	api := &apiFake{getOutput: []byte(`{"status":{"succeeded":1}}`), listOutput: []byte(`{"items":[]}`)}
	handle, err := New(api, "default").Inspect(context.Background(), domain.ActivityHandle{ExternalID: "job"})
	if err != nil || handle.Status != domain.HandleCompleted || handle.Metadata["artifactObservationError"] == nil {
		t.Fatalf("handle=%+v err=%v", handle, err)
	}
}
