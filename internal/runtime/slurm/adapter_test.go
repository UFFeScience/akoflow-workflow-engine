package slurm

import (
	"context"
	"strings"
	"testing"

	"github.com/UFFeScience/akoflow/internal/domain"
)

type executorFake struct{ output, input []byte }

func (f *executorFake) Run(_ context.Context, _ string, _ []string, input []byte) ([]byte, error) {
	f.input = input
	return f.output, nil
}

func TestAdapterSubmitsSafeBatchScript(t *testing.T) {
	executor := &executorFake{output: []byte("123;cluster\n")}
	adapter := New(executor, "cpu")
	activity := domain.Activity{ID: "analysis", Command: domain.ActivityCommand{Image: "image.sif", Entrypoint: "python", Arguments: []string{"a'b"}}, Resources: domain.ActivityResources{CPU: 2, MemoryBytes: 1048576}}
	handle, err := adapter.Start(context.Background(), domain.ActivityExecutionContext{Run: domain.ExecutionRun{ID: "run"}, Activity: activity, Resource: domain.Resource{ID: "node", RuntimeID: "slurm"}})
	if err != nil {
		t.Fatal(err)
	}
	if handle.ExternalID != "123" || !strings.Contains(string(executor.input), `'a'"'"'b'`) {
		t.Fatalf("handle=%+v script=%s", handle, executor.input)
	}
}

func TestAdapterMapsSlurmStatus(t *testing.T) {
	executor := &executorFake{output: []byte("COMPLETED|0:0\n")}
	handle, err := New(executor, "").Inspect(context.Background(), domain.ActivityHandle{ExternalID: "1"})
	if err != nil || handle.Status != domain.HandleCompleted || handle.ExitCode == nil || *handle.ExitCode != 0 {
		t.Fatalf("handle=%+v err=%v", handle, err)
	}
}
