package local_runtime_service

import (
	"errors"
	"github.com/UFFeScience/akoflow/internal/application/ports"
	workflow_activity_entity "github.com/UFFeScience/akoflow/internal/domain/workflow/activity"
	workflow_entity "github.com/UFFeScience/akoflow/internal/domain/workflow/definition"
	"strings"
	"testing"
)

type localActivityRepo struct {
	ports.ActivityRepository
	activity       workflow_activity_entity.WorkflowActivities
	err, errorProc error
	statuses       []int
}

func (r *localActivityRepo) Find(int) (workflow_activity_entity.WorkflowActivities, error) {
	return r.activity, r.err
}
func (r *localActivityRepo) UpdateStatus(_ int, s int) error {
	r.statuses = append(r.statuses, s)
	return nil
}
func (r *localActivityRepo) UpdateProcID(int, string) error { return r.errorProc }

type localWorkflowRepo struct {
	ports.WorkflowRepository
	workflow workflow_entity.Workflow
	err      error
}

func (r *localWorkflowRepo) Find(int) (workflow_entity.Workflow, error) { return r.workflow, nil }
func (r *localWorkflowRepo) UpdateStatus(int, int) error                { return r.err }

type localMetricsRepo struct {
	ports.MetricsRepository
	calls int
	err   error
}

func (r *localMetricsRepo) Create(ports.ActivityMetric) error { r.calls++; return r.err }

type localLogsRepo struct {
	ports.LogsRepository
	calls int
}

func (r *localLogsRepo) Create(ports.ActivityLog) error { r.calls++; return nil }

type localConnectorFake struct{ output, pid string }

func (c *localConnectorFake) RunCommand(string, ...string) (string, error) { return c.pid, nil }
func (c *localConnectorFake) RunCommandWithOutput(string, ...string) (string, error) {
	return c.output, nil
}

func TestExtractLogsMetricsAndCompletion(t *testing.T) {
	s := LocalRuntimeService{}
	out, stderr, err := s.ExtractLogs("##START_LOG_OUTPUT## hello ##END_LOG_OUTPUT## ##START_LOG_ERROR## warning ##END_LOG_ERROR##")
	if err != nil || out != "hello" || stderr != "warning" {
		t.Fatalf("out=%q errlog=%q err=%v", out, stderr, err)
	}
	out, stderr, _ = s.ExtractLogs("none")
	if out != "" || stderr != "" {
		t.Fatal("empty markers")
	}
	cpu, memory, err := s.ExtractMetrics("TOTAL_CPU=(12.5%) anything TOTAL_MEM=(44%)")
	if err != nil || cpu != "12.5" || memory != "44" {
		t.Fatalf("cpu=%q mem=%q err=%v", cpu, memory, err)
	}
	if _, _, err = s.ExtractMetrics("none"); err == nil {
		t.Fatal("missing metrics")
	}
	if !s.ProcessCompleted("AKOFLOW_JOB_FINISHED") || s.ProcessCompleted("running") {
		t.Fatal("completion")
	}
}
func TestMakeLocalActivityAndConfiguration(t *testing.T) {
	s := NewLocalRuntimeService()
	if s.SetRuntimeName("node") != &s || s.SetRuntimeType("local") != &s {
		t.Fatal("setters")
	}
	wf := workflow_entity.Workflow{Id: 3, Spec: workflow_entity.WorkflowSpec{MountPath: "/data/"}}
	activity := workflow_activity_entity.WorkflowActivities{Id: 4, WorkflowId: 3, Run: "echo ok", MountPath: "/data/"}
	command := s.makeLocalActivity(wf, activity)
	for _, part := range []string{"base64 -d | bash", "akoflow_out3_4.out", "akoflow_err3_4.err"} {
		if !strings.Contains(command, part) {
			t.Fatalf("command missing %q: %s", part, command)
		}
	}
}

func TestLocalApplyAndMonitor(t *testing.T) {
	a := workflow_activity_entity.WorkflowActivities{Id: 4, WorkflowId: 3, Status: 1, Runtime: "local", Run: "echo ok", MountPath: "/data/"}
	w := workflow_entity.Workflow{Id: 3, Spec: workflow_entity.WorkflowSpec{MountPath: "/data/", Activities: []workflow_activity_entity.WorkflowActivities{a}}}
	ar := &localActivityRepo{activity: a}
	wr := &localWorkflowRepo{workflow: w}
	mr := &localMetricsRepo{}
	lr := &localLogsRepo{}
	c := &localConnectorFake{pid: "22"}
	s := LocalRuntimeService{activityRepository: ar, workflowRepository: wr, metricsRepository: mr, logsRepository: lr, localConnector: c, runtimeName: "local"}
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
	c.output = "AKOFLOW_JOB_FINISHED"
	s.VerifyActivitiesWasFinished(w)
	if ar.statuses[len(ar.statuses)-1] != 2 {
		t.Fatal("finished")
	}
	c.output = "TOTAL_CPU=(10%) TOTAL_MEM=(20%) ##START_LOG_OUTPUT##ok##END_LOG_OUTPUT## ##START_LOG_ERROR##warn##END_LOG_ERROR##"
	s.VerifyActivitiesWasFinished(w)
	if mr.calls == 0 || lr.calls != 2 {
		t.Fatalf("metrics=%d logs=%d", mr.calls, lr.calls)
	}
	mr.err = errors.New("db")
	s.VerifyActivitiesWasFinished(w)
}
