package singularity_runtime_service

import (
	"errors"
	"github.com/UFFeScience/akoflow/internal/application/ports"
	workflow_activity_entity "github.com/UFFeScience/akoflow/internal/domain/workflow/activity"
	workflow_entity "github.com/UFFeScience/akoflow/internal/domain/workflow/definition"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type singularActivityRepo struct {
	ports.ActivityRepository
	activity       workflow_activity_entity.WorkflowActivities
	err, errorProc error
	statuses       []int
}

func (r *singularActivityRepo) Find(int) (workflow_activity_entity.WorkflowActivities, error) {
	return r.activity, r.err
}
func (r *singularActivityRepo) UpdateStatus(_ int, s int) error {
	r.statuses = append(r.statuses, s)
	return nil
}
func (r *singularActivityRepo) UpdateProcID(int, string) error { return r.errorProc }

type singularWorkflowRepo struct {
	ports.WorkflowRepository
	workflow workflow_entity.Workflow
	err      error
}

func (r *singularWorkflowRepo) Find(int) (workflow_entity.Workflow, error) { return r.workflow, nil }
func (r *singularWorkflowRepo) UpdateStatus(int, int) error                { return r.err }

type singularMetricsRepo struct {
	ports.MetricsRepository
	calls int
	err   error
}

func (r *singularMetricsRepo) Create(ports.ActivityMetric) error { r.calls++; return r.err }

type singularLogsRepo struct {
	ports.LogsRepository
	calls int
}

func (r *singularLogsRepo) Create(ports.ActivityLog) error { r.calls++; return nil }

type singularConnectorFake struct{ output, pid string }

func (c *singularConnectorFake) RunCommand(string, ...string) (string, error) { return c.pid, nil }
func (c *singularConnectorFake) RunCommandWithOutput(string, ...string) (string, error) {
	return c.output, nil
}

func TestExtractLogsMetricsAndCompletion(t *testing.T) {
	s := SingularityRuntimeService{}
	out, stderr, err := s.ExtractLogs("##START_LOG_OUTPUT## hello ##END_LOG_OUTPUT## ##START_LOG_ERROR## warning ##END_LOG_ERROR##")
	if err != nil || out != "hello" || stderr != "warning" {
		t.Fatalf("out=%q stderr=%q err=%v", out, stderr, err)
	}
	cpu, memory, err := s.ExtractMetrics("TOTAL_CPU=(10%) TOTAL_MEM=(20%)")
	if err != nil || cpu != "10" || memory != "20" {
		t.Fatalf("cpu=%q memory=%q err=%v", cpu, memory, err)
	}
	if _, _, err = s.ExtractMetrics("none"); err == nil {
		t.Fatal("metrics")
	}
	if !s.ProcessCompleted("#NO_PROCESS_FOUND") || s.ProcessCompleted("running") {
		t.Fatal("completion")
	}
}
func TestMakeSingularityCommands(t *testing.T) {
	s := NewMakeSingularityActivityService()
	wf := workflow_entity.Workflow{Id: 1, Spec: workflow_entity.WorkflowSpec{MountPath: "/data", Image: "image.sif"}}
	a := workflow_activity_entity.WorkflowActivities{Id: 2, WorkflowId: 1, Run: "echo ok", MemoryLimit: "1G", CpuLimit: "2"}
	command := s.Handle(wf, a)
	for _, part := range []string{"singularity exec", "--memory 1G", "--cpus 2", "akoflow_out1_2.out"} {
		if !strings.Contains(command, part) {
			t.Fatalf("missing %q: %s", part, command)
		}
	}
	hpc := s.MakeContainerCommandActivityToHPC(wf, a)
	if !strings.Contains(hpc, "/akoflow-wfa-shared") || !strings.Contains(hpc, "image.sif") {
		t.Fatal(hpc)
	}
}
func TestMonitorReadsAndRendersScript(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "monitor.sh")
	content := "pid=##PARENT_PID## wf=##WORKFLOW_ID## task=##WORKFLOW_ACTIVITY_ID## path=##WORKFLOW_PATH_DATA_DIR##"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	monitor := &AkfMonitorSingularity{FilePath: path}
	lines, err := monitor.ReadFile()
	if err != nil || len(lines) != 1 {
		t.Fatalf("lines=%v err=%v", lines, err)
	}
	text, err := monitor.ReadFileAsString()
	if err != nil || text != content {
		t.Fatalf("text=%q err=%v", text, err)
	}
	wf := workflow_entity.Workflow{Id: 7, Spec: workflow_entity.WorkflowSpec{MountPath: "/data"}}
	a := workflow_activity_entity.WorkflowActivities{Id: 8, ProcId: "123"}
	if monitor.SetWorkflow(wf) != monitor || monitor.SetWorkflowActivity(a) != monitor || monitor.GetWorkflow().Id != 7 || monitor.GetWorkflowActivity().Id != 8 {
		t.Fatal("state")
	}
	script, err := monitor.GetScript()
	if err != nil || strings.Contains(script, "##") || !strings.Contains(script, "pid=123") {
		t.Fatalf("script=%q err=%v", script, err)
	}
	monitor.WorkflowActivity.ProcId = ""
	if script, err = monitor.GetScript(); err != nil || script != "" {
		t.Fatalf("empty script=%q err=%v", script, err)
	}
	monitor.FilePath = filepath.Join(directory, "missing")
	if _, err = monitor.ReadFile(); err == nil {
		t.Fatal("read missing")
	}
	if _, err = monitor.ReadFileAsString(); err == nil {
		t.Fatal("string missing")
	}
	if _, err = monitor.GetScript(); err == nil {
		t.Fatal("script missing")
	}
}
func TestNewMonitorHasScriptPath(t *testing.T) {
	if monitor := NewAkfMonitorSingularity(); monitor.FilePath == "" {
		t.Fatal("path")
	}
}

func TestSingularityApplyAndRunningHandler(t *testing.T) {
	a := workflow_activity_entity.WorkflowActivities{Id: 4, WorkflowId: 3, Status: 1, Run: "echo ok", MountPath: "/data/"}
	w := workflow_entity.Workflow{Id: 3, Spec: workflow_entity.WorkflowSpec{MountPath: "/data/", Image: "image.sif", Activities: []workflow_activity_entity.WorkflowActivities{a}}}
	ar := &singularActivityRepo{activity: a}
	wr := &singularWorkflowRepo{workflow: w}
	mr := &singularMetricsRepo{}
	lr := &singularLogsRepo{}
	c := &singularConnectorFake{pid: "22"}
	s := SingularityRuntimeService{activityRepository: ar, workflowRepository: wr, metricsRepository: mr, logsRepository: lr, singularityConnector: c, makeSingularityActivity: NewMakeSingularityActivityService()}
	s.ApplyJob(3, 4)
	if len(ar.statuses) == 0 {
		t.Fatal("apply")
	}
	ar.err = errors.New("missing")
	s.ApplyJob(3, 4)
	ar.err = nil
	wr.err = errors.New("db")
	s.ApplyJob(3, 4)
	wr.err = nil
	ar.errorProc = errors.New("db")
	s.ApplyJob(3, 4)
	ar.errorProc = nil
	s.handleProcessRunning(w, a, "", "TOTAL_CPU=(10%) TOTAL_MEM=(20%) ##START_LOG_OUTPUT##ok##END_LOG_OUTPUT## ##START_LOG_ERROR##warn##END_LOG_ERROR##")
	if mr.calls == 0 || lr.calls != 2 {
		t.Fatalf("metrics=%d logs=%d", mr.calls, lr.calls)
	}
	mr.err = errors.New("db")
	s.handleProcessRunning(w, a, "", "TOTAL_CPU=(10%) TOTAL_MEM=(20%)")
	s.handleProcessRunning(w, a, "", "invalid")
	s.handleProcessCompleted(w, a, "", "")
}
