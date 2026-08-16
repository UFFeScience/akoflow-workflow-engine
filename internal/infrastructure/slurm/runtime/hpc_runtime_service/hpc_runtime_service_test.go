package hpc_runtime_service

import (
	"errors"
	"testing"

	"github.com/UFFeScience/akoflow/internal/application/ports"
	runtime_entity "github.com/UFFeScience/akoflow/internal/domain/resource/runtime"
	workflow_activity_entity "github.com/UFFeScience/akoflow/internal/domain/workflow/activity"
	workflow_entity "github.com/UFFeScience/akoflow/internal/domain/workflow/definition"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/runtime_repository"
	connector_hpc "github.com/UFFeScience/akoflow/internal/infrastructure/slurm/connector"
)

type hpcActivityRepo struct {
	ports.ActivityRepository
	activity                        workflow_activity_entity.WorkflowActivities
	findErr, errorStatus, errorProc error
	statuses                        []int
	proc                            string
}

func (r *hpcActivityRepo) Find(int) (workflow_activity_entity.WorkflowActivities, error) {
	return r.activity, r.findErr
}
func (r *hpcActivityRepo) UpdateStatus(_ int, status int) error {
	r.statuses = append(r.statuses, status)
	return r.errorStatus
}
func (r *hpcActivityRepo) UpdateProcID(_ int, pid string) error { r.proc = pid; return r.errorProc }

type hpcWorkflowRepo struct {
	ports.WorkflowRepository
	workflow  workflow_entity.Workflow
	updateErr error
	status    int
}

func (r *hpcWorkflowRepo) Find(int) (workflow_entity.Workflow, error) { return r.workflow, nil }
func (r *hpcWorkflowRepo) UpdateStatus(_ int, status int) error {
	r.status = status
	return r.updateErr
}

type hpcRuntimeRepo struct {
	runtime_repository.IRuntimeRepository
	runtime *runtime_entity.Runtime
	err     error
	status  int
}

func (r *hpcRuntimeRepo) GetByName(string) (*runtime_entity.Runtime, error) { return r.runtime, r.err }
func (r *hpcRuntimeRepo) UpdateStatus(_ *runtime_entity.Runtime, status int) error {
	r.status = status
	return nil
}

type hpcConnector struct {
	connected                    bool
	vpnErr, errorRun, errorBuild error
	output                       string
	commands                     []string
	runtime                      runtime_entity.Runtime
}

func (c *hpcConnector) RunCommand(string, ...string) (string, error) { return c.output, c.errorRun }
func (c *hpcConnector) RunCommandWithOutput(string, ...string) (string, error) {
	return c.output, c.errorRun
}
func (c *hpcConnector) RunCommandWithOutputRemote(command string, _ ...string) (string, error) {
	c.commands = append(c.commands, command)
	return c.output, c.errorRun
}
func (c *hpcConnector) IsVPNConnected() (bool, error) { return c.connected, c.vpnErr }
func (c *hpcConnector) ExecuteMultiplesCommand(commands []string) {
	c.commands = append(c.commands, commands...)
}
func (c *hpcConnector) SetRuntime(r runtime_entity.Runtime) connector_hpc.IConnectorHPCRuntime {
	c.runtime = r
	return c
}
func (c *hpcConnector) BuildRemoteCommand(_ runtime_entity.Runtime, command string) (string, error) {
	return "remote " + command, c.errorBuild
}

func hpcFixture() (*HPCRuntimeService, *hpcActivityRepo, *hpcWorkflowRepo, *hpcRuntimeRepo, *hpcConnector) {
	a := &hpcActivityRepo{activity: workflow_activity_entity.WorkflowActivities{Id: 3, WorkflowId: 2, Status: ports.ActivityStatusCreated, Name: "task", ProcId: "99", Runtime: "hpc"}}
	w := &hpcWorkflowRepo{workflow: workflow_entity.Workflow{Id: 2, Status: ports.WorkflowStatusRunning, Spec: workflow_entity.WorkflowSpec{Runtime: "hpc", MountPath: "/data", Activities: []workflow_activity_entity.WorkflowActivities{{Id: 3, Name: "task", ProcId: "99", Runtime: "hpc"}}}}}
	runtime := runtime_entity.NewRuntime("hpc", 1, map[string]string{"HPC_SBATCHTEMPLATE": "IyNDT01NQU5EIw==", "HPC_QUEUE": "q", "HPC_MOUNT_PATH": "/data"}, "", "")
	r := &hpcRuntimeRepo{runtime: runtime}
	c := &hpcConnector{connected: true, output: "Submitted batch job 123"}
	s := &HPCRuntimeService{activityRepository: a, workflowRepository: w, runtimeRepository: r, connectorHPCRuntime: c, runtimeName: "hpc"}
	return s, a, w, r, c
}

func TestHPCServiceParsingAndSetters(t *testing.T) {
	s := &HPCRuntimeService{}
	if s.SetRuntimeName("hpc") != s || s.SetRuntimeType("slurm") != s {
		t.Fatal("setters")
	}
	if id, err := s.extractJobID("Submitted batch job 12345"); err != nil || id != "12345" {
		t.Fatalf("job id %q %v", id, err)
	}
	if id, err := s.extractJobID("none"); err != nil || id != "" {
		t.Fatal("missing id contract")
	}
	if v, err := extractField(`name=(\w+)`, `name=value`); err != nil || v != "value" {
		t.Fatal("field")
	}
	if _, err := extractField(`id=(\d+)`, `none`); err == nil {
		t.Fatal("missing field")
	}
}

func TestHPCApplyJobSuccessAndBoundaries(t *testing.T) {
	s, a, w, _, _ := hpcFixture()
	if out := s.ApplyJob(2, 3); out == "" || a.proc != "123" || w.status == 0 {
		t.Fatalf("success out=%q proc=%q status=%d", out, a.proc, w.status)
	}
	tests := []func(*HPCRuntimeService, *hpcActivityRepo, *hpcWorkflowRepo, *hpcRuntimeRepo, *hpcConnector){
		func(_ *HPCRuntimeService, a *hpcActivityRepo, _ *hpcWorkflowRepo, _ *hpcRuntimeRepo, _ *hpcConnector) {
			a.findErr = errors.New("missing")
		},
		func(_ *HPCRuntimeService, a *hpcActivityRepo, _ *hpcWorkflowRepo, _ *hpcRuntimeRepo, _ *hpcConnector) {
			a.activity.Status = 99
		},
		func(_ *HPCRuntimeService, _ *hpcActivityRepo, _ *hpcWorkflowRepo, r *hpcRuntimeRepo, _ *hpcConnector) {
			r.err = errors.New("db")
		},
		func(_ *HPCRuntimeService, _ *hpcActivityRepo, _ *hpcWorkflowRepo, _ *hpcRuntimeRepo, c *hpcConnector) {
			c.vpnErr = errors.New("vpn")
		},
		func(_ *HPCRuntimeService, _ *hpcActivityRepo, _ *hpcWorkflowRepo, _ *hpcRuntimeRepo, c *hpcConnector) {
			c.connected = false
		},
		func(_ *HPCRuntimeService, _ *hpcActivityRepo, w *hpcWorkflowRepo, _ *hpcRuntimeRepo, _ *hpcConnector) {
			w.updateErr = errors.New("db")
		},
		func(_ *HPCRuntimeService, a *hpcActivityRepo, _ *hpcWorkflowRepo, _ *hpcRuntimeRepo, _ *hpcConnector) {
			a.errorProc = errors.New("db")
		},
	}
	for i, setup := range tests {
		s, a, w, r, c := hpcFixture()
		setup(s, a, w, r, c)
		if out := s.ApplyJob(2, 3); out != "" {
			t.Fatalf("case %d returned %q", i, out)
		}
	}
}

func TestHPCFinishedAndHealthChecks(t *testing.T) {
	s, a, w, r, c := hpcFixture()
	c.output = "/data/akoflow_finished_2_3.txt\ninvalid"
	a.activity.Status = ports.ActivityStatusRunning
	if !s.VerifyActivitiesWasFinished(w.workflow) || len(a.statuses) == 0 {
		t.Fatal("finished activity")
	}
	c.output = ""
	if !s.VerifyActivitiesWasFinished(w.workflow) {
		t.Fatal("empty finished set")
	}
	r.err = errors.New("db")
	if got := s.getFinishedActivities(w.workflow); len(got) != 0 {
		t.Fatal("runtime error")
	}
	r.err = nil
	c.connected = false
	if s.HealthCheck("hpc") {
		t.Fatal("disconnected")
	}
	c.connected = true
	c.vpnErr = errors.New("vpn")
	if s.HealthCheck("hpc") {
		t.Fatal("vpn error")
	}
	c.vpnErr = nil
	r.err = errors.New("db")
	if s.HealthCheck("hpc") {
		t.Fatal("runtime error")
	}
	r.err = nil
	c.errorRun = errors.New("remote")
	if s.HealthCheck("hpc") {
		t.Fatal("remote error")
	}
	c.errorRun = nil
	c.output = "No nodes available"
	if s.HealthCheck("hpc") {
		t.Fatal("no nodes")
	}
	c.output = "partition up"
	if !s.HealthCheck("hpc") || r.status != runtime_repository.STATUS_READY {
		t.Fatal("healthy")
	}
}
