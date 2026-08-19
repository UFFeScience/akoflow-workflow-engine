package local

import (
	"context"
	"testing"
	"time"

	"github.com/UFFeScience/akoflow/internal/domain"
)

func TestAdapterRunsLocalActivity(t *testing.T) {
	adapter := New()
	handle, err := adapter.Start(context.Background(), domain.ActivityExecutionContext{
		Run: domain.ExecutionRun{ID: "run"},
		Activity: domain.Activity{
			ID: "activity",
			Command: domain.ActivityCommand{
				Entrypoint: "sh", Arguments: []string{"-c", "exit 0"},
			},
		},
		Resource: domain.Resource{ID: "local"}, RuntimeID: "local",
	})
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for handle.Status == domain.HandleRunning && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
		updated, err := adapter.Inspect(context.Background(), handle)
		if err != nil {
			t.Fatal(err)
		}
		handle = updated
	}
	if handle.Status != domain.HandleCompleted {
		t.Fatalf("handle=%+v", handle)
	}
}
