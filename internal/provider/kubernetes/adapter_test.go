package kubernetes

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/UFFeScience/akoflow/internal/domain"
)

type apiFake struct {
	created   map[string][]byte
	getOutput []byte
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
}

func TestAdapterMapsKubernetesStatus(t *testing.T) {
	api := &apiFake{getOutput: []byte(`{"status":{"succeeded":1}}`)}
	handle, err := New(api, "default").Inspect(context.Background(), domain.ActivityHandle{
		ExternalID: "job", Status: domain.HandleRunning,
	})
	if err != nil || handle.Status != domain.HandleCompleted {
		t.Fatalf("handle=%+v err=%v", handle, err)
	}
}
