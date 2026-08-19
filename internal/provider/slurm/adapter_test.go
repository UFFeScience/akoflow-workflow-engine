package slurm

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/UFFeScience/akoflow/internal/domain"
)

type executorFake struct {
	output, input []byte
	name          string
	args          []string
}

func (f *executorFake) Run(_ context.Context, name string, args []string, input []byte) ([]byte, error) {
	f.name = name
	f.args = args
	f.input = input
	return f.output, nil
}

func TestAdapterExecutesDirectlyOnLoginResource(t *testing.T) {
	adapter := NewWithConfig(&executorFake{}, Config{Partition: "cpu", ScriptDirectory: t.TempDir()})
	execution := domain.ActivityExecutionContext{
		Run: domain.ExecutionRun{ID: "run"},
		Activity: domain.Activity{ID: "lightweight", Command: domain.ActivityCommand{
			Entrypoint: "/bin/sh", Arguments: []string{"-c", "exit 0"},
		}},
		Resource: domain.Resource{ID: "login", ExecutionTarget: domain.ExecutionTargetDirect}, RuntimeID: "slurm",
	}
	handle, err := adapter.Start(context.Background(), execution)
	if err != nil {
		t.Fatal(err)
	}
	if handle.Metadata["slurmSubmission"] != "login-node" {
		t.Fatalf("metadata=%+v", handle.Metadata)
	}
	scriptPath, _ := handle.Metadata["scriptPath"].(string)
	script, err := os.ReadFile(scriptPath)
	if err != nil || !strings.Contains(string(script), "'/bin/sh' '-c' 'exit 0'") {
		t.Fatalf("script=%s err=%v", script, err)
	}
	for attempt := 0; attempt < 50; attempt++ {
		handle, err = adapter.Inspect(context.Background(), handle)
		if err != nil {
			t.Fatal(err)
		}
		if handle.Status == domain.HandleCompleted {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("direct activity did not complete: %+v", handle)
}

func TestAdapterSubmitsSafeBatchScript(t *testing.T) {
	executor := &executorFake{output: []byte("123;cluster\n")}
	adapter := NewWithConfig(executor, Config{Partition: "cpu", ScriptDirectory: t.TempDir()})
	activity := domain.Activity{ID: "analysis", Command: domain.ActivityCommand{Image: "image.sif", Entrypoint: "python", Arguments: []string{"a'b"}}, Resources: domain.ActivityResources{CPU: 2, MemoryBytes: 1048576}}
	handle, err := adapter.Start(context.Background(), domain.ActivityExecutionContext{Run: domain.ExecutionRun{ID: "run"}, Activity: activity, Resource: domain.Resource{ID: "node"}, RuntimeID: "slurm"})
	if err != nil {
		t.Fatal(err)
	}
	if handle.ExternalID != "123" || executor.name != "sbatch" || len(executor.args) != 2 {
		t.Fatalf("handle=%+v command=%s args=%v", handle, executor.name, executor.args)
	}
	script, err := os.ReadFile(executor.args[1])
	if err != nil || !strings.Contains(string(script), `'a'"'"'b'`) || strings.Index(string(script), "#SBATCH") > strings.Index(string(script), "set -eu") {
		t.Fatalf("script=%s err=%v", script, err)
	}
}

func TestAdapterMapsSlurmStatus(t *testing.T) {
	executor := &executorFake{output: []byte("COMPLETED|0:0\n")}
	handle, err := New(executor, "").Inspect(context.Background(), domain.ActivityHandle{ExternalID: "1"})
	if err != nil || handle.Status != domain.HandleCompleted || handle.ExitCode == nil || *handle.ExitCode != 0 {
		t.Fatalf("handle=%+v err=%v", handle, err)
	}
}
