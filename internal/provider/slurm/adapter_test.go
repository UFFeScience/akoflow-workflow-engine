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

func TestSingularityImageConvertsPlainOCIReference(t *testing.T) {
	if got := singularityImage("alpine:3.20"); got != "docker://alpine:3.20" {
		t.Fatalf("plain image = %q", got)
	}
	for _, image := range []string{"docker://alpine:3.20", "library://alpine", "/shared/tool.sif"} {
		if got := singularityImage(image); got != image {
			t.Fatalf("explicit image %q became %q", image, got)
		}
	}
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
	if !strings.Contains(string(script), "trap finish EXIT") || !strings.Contains(string(script), "state=running") || !strings.Contains(string(script), "akoflow-run-analysis-%j.log") || handle.Metadata["sentinelPath"] != "akoflow-run-analysis-123.status" {
		t.Fatalf("missing execution sentinel: script=%s metadata=%+v", script, handle.Metadata)
	}
}

func TestAdapterParsesJobIDAfterSSHWarning(t *testing.T) {
	adapter := NewWithConfig(&executorFake{output: []byte("warning from ssh\n4847261;cluster\n")}, Config{ScriptDirectory: t.TempDir()})
	handle, err := adapter.Start(context.Background(), domain.ActivityExecutionContext{
		Run: domain.ExecutionRun{ID: "run"}, RuntimeID: "slurm", Resource: domain.Resource{ID: "node"},
		Activity: domain.Activity{ID: "task", Command: domain.ActivityCommand{Entrypoint: "echo", Arguments: []string{"ok"}}},
	})
	if err != nil || handle.ExternalID != "4847261" {
		t.Fatalf("handle=%+v err=%v", handle, err)
	}
}

func TestAdapterMapsSlurmStatus(t *testing.T) {
	executor := &executorFake{output: []byte("COMPLETED|0:0\n")}
	handle, err := New(executor, "").Inspect(context.Background(), domain.ActivityHandle{ExternalID: "1"})
	if err != nil || handle.Status != domain.HandleCompleted || handle.ExitCode == nil || *handle.ExitCode != 0 {
		t.Fatalf("handle=%+v err=%v", handle, err)
	}
}

func TestSentinelAndScontrolStatusParsing(t *testing.T) {
	values := sentinelValues("state=failed\nexit_code=17\n")
	if values["state"] != "failed" || values["exit_code"] != "17" {
		t.Fatalf("sentinel values=%+v", values)
	}
	if state := slurmControlState("JobId=123 JobState=RUNNING Reason=None"); state != "RUNNING" {
		t.Fatalf("scontrol state=%q", state)
	}
}

func TestConnectionFactorySubmitsToConfiguredSSHLoginNode(t *testing.T) {
	executor := &executorFake{output: []byte("123;cluster\n")}
	adapter, err := (ConnectionFactory{Executor: executor, DefaultScriptDirectory: t.TempDir()}).Build(
		domain.EnvironmentRuntime{ID: "hpc", Driver: domain.RuntimeDriverSlurm,
			Configuration: map[string]any{"connectionId": "hpc-ssh", "partition": "cpu"}},
		domain.EnvironmentConnection{ID: "hpc-ssh", Type: domain.ConnectionSSH,
			Endpoint: "login.example", Username: "wes", Configuration: map[string]any{"port": float64(2222)}},
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = adapter.Start(context.Background(), domain.ActivityExecutionContext{
		Run: domain.ExecutionRun{ID: "run"}, RuntimeID: "hpc", Resource: domain.Resource{ID: "node"},
		Activity: domain.Activity{ID: "task", Command: domain.ActivityCommand{Entrypoint: "echo", Arguments: []string{"ok"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if executor.name != "ssh" || !strings.Contains(strings.Join(executor.args, " "), "wes@login.example") ||
		!strings.Contains(string(executor.input), "#SBATCH --partition=cpu") {
		t.Fatalf("name=%s args=%v input=%s", executor.name, executor.args, executor.input)
	}
}

func TestConnectionProberChecksSlurmThroughSSH(t *testing.T) {
	executor := &executorFake{output: []byte("cpu*\ngpu\n")}
	health := NewConnectionProber(executor).Probe(context.Background(), domain.EnvironmentConnection{
		Type: domain.ConnectionSSH, Endpoint: "login.example", Username: "wes",
	})
	if !health.Healthy || executor.name != "ssh" || !strings.Contains(health.Message, "cpu") {
		t.Fatalf("health=%+v command=%s", health, executor.name)
	}
}
