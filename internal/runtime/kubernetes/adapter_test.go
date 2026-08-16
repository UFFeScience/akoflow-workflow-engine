package kubernetes

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/UFFeScience/akoflow/internal/domain"
)

type executorFake struct{ input, output []byte }

func (f *executorFake) Run(_ context.Context, _ string, _ []string, input []byte) ([]byte, error) {
	f.input = input
	return f.output, nil
}

func TestAdapterCreatesJobAndServiceFromActivity(t *testing.T) {
	executor := &executorFake{}
	adapter := New(executor, "science")
	activity := domain.Activity{ID: "A 1", Command: domain.ActivityCommand{Image: "image:1", Entrypoint: "python", Arguments: []string{"run.py"}}, Resources: domain.ActivityResources{CPU: 2, MemoryBytes: 1048576}, Service: &domain.ActivityService{Ports: []int{8080}}}
	handle, err := adapter.Start(context.Background(), domain.ActivityExecutionContext{Run: domain.ExecutionRun{ID: "run"}, Activity: activity, Resource: domain.Resource{ID: "node", RuntimeID: "kubernetes"}})
	if err != nil {
		t.Fatal(err)
	}
	if handle.Status != domain.HandleStarting || len(handle.Endpoints) != 1 {
		t.Fatalf("handle=%+v", handle)
	}
	var manifest struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(executor.input, &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Items) != 2 {
		t.Fatalf("items=%d", len(manifest.Items))
	}
}

func TestAdapterMapsKubernetesStatus(t *testing.T) {
	executor := &executorFake{output: []byte(`{"status":{"succeeded":1}}`)}
	handle, err := New(executor, "default").Inspect(context.Background(), domain.ActivityHandle{ExternalID: "job", Status: domain.HandleRunning})
	if err != nil || handle.Status != domain.HandleCompleted {
		t.Fatalf("handle=%+v err=%v", handle, err)
	}
}
