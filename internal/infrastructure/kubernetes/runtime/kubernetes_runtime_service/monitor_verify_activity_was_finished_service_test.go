package kubernetes_runtime_service

import (
	"errors"
	"github.com/UFFeScience/akoflow/internal/application/ports"
	runtime_entity "github.com/UFFeScience/akoflow/internal/domain/resource/runtime"
	workflow_activity_entity "github.com/UFFeScience/akoflow/internal/domain/workflow/activity"
	workflow_entity "github.com/UFFeScience/akoflow/internal/domain/workflow/definition"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/runtime_repository"
	connector_k8s "github.com/UFFeScience/akoflow/internal/infrastructure/kubernetes/connector"
	connector_job_k8s "github.com/UFFeScience/akoflow/internal/infrastructure/kubernetes/connector/connector_job_k8s"
	connector_pod_k8s "github.com/UFFeScience/akoflow/internal/infrastructure/kubernetes/connector/connector_pod_k8s"
	"testing"
)

type monitorActivityRepo struct {
	ports.ActivityRepository
	activity   workflow_activity_entity.WorkflowActivities
	pre        workflow_activity_entity.WorkflowPreActivityDatabase
	statuses   []int
	preUpdates int
}

func (r *monitorActivityRepo) Find(int) (workflow_activity_entity.WorkflowActivities, error) {
	return r.activity, nil
}
func (r *monitorActivityRepo) FindPreActivity(int) (workflow_activity_entity.WorkflowPreActivityDatabase, error) {
	return r.pre, nil
}
func (r *monitorActivityRepo) UpdateStatus(_ int, s int) error {
	r.statuses = append(r.statuses, s)
	return nil
}
func (r *monitorActivityRepo) UpdatePreActivity(_ int, _ workflow_activity_entity.WorkflowPreActivityDatabase) error {
	r.preUpdates++
	return nil
}

type monitorRuntimeRepo struct {
	runtime_repository.IRuntimeRepository
	runtime *runtime_entity.Runtime
	err     error
}

func (r *monitorRuntimeRepo) GetByName(string) (*runtime_entity.Runtime, error) {
	return r.runtime, r.err
}

type monitorLogsRepo struct {
	ports.LogsRepository
	calls int
}

func (r *monitorLogsRepo) Create(ports.ActivityLog) error { r.calls++; return nil }

type monitorJob struct {
	connector_job_k8s.IConnectorJob
	response connector_job_k8s.ResponseGetJob
	err      error
}

func (j *monitorJob) GetJob(string, string) (connector_job_k8s.ResponseGetJob, error) {
	return j.response, j.err
}

type monitorPod struct {
	connector_pod_k8s.IConnectorPod
}

func (p *monitorPod) GetPodByJob(string, string) (connector_pod_k8s.ResponseGetJobByPod, error) {
	var r connector_pod_k8s.ResponseGetJobByPod
	r.Items = []connector_pod_k8s.ResponseGetJobByPodItem{{Metadata: connector_pod_k8s.ResponseGetJobByPodItemMetadata{Name: "pod"}}}
	return r, nil
}
func (p *monitorPod) GetPodLogs(string, string) (string, error) { return "logs", nil }

type monitorConnector struct {
	connector_k8s.IConnector
	job *monitorJob
	pod *monitorPod
}

func (c *monitorConnector) Job(*runtime_entity.Runtime) connector_job_k8s.IConnectorJob { return c.job }
func (c *monitorConnector) Pod(*runtime_entity.Runtime) connector_pod_k8s.IConnectorPod { return c.pod }

func monitorFixture() (*MonitorVerifyActivityWasFinishedService, *monitorActivityRepo, *monitorRuntimeRepo, *monitorJob, *monitorLogsRepo) {
	a := workflow_activity_entity.WorkflowActivities{Id: 3, Name: "task", Runtime: "k8s", Status: 1}
	ar := &monitorActivityRepo{activity: a}
	rr := &monitorRuntimeRepo{runtime: runtime_entity.NewRuntime("k8s", 1, nil, "", "")}
	j := &monitorJob{}
	l := &monitorLogsRepo{}
	s := &MonitorVerifyActivityWasFinishedService{namespace: "ns", runtimeName: "k8s", activityRepository: ar, runtimeRepository: rr, logsRepository: l, connector: &monitorConnector{job: j, pod: &monitorPod{}}}
	return s, ar, rr, j, l
}
func TestMonitorActivityStates(t *testing.T) {
	s, a, r, j, l := monitorFixture()
	wf := workflow_entity.Workflow{Id: 2}
	if s.SetRuntimeName("k8s") != s || s.SetRuntimeType("k8s") != s {
		t.Fatal("setters")
	}
	a.activity.Status = 2
	if s.handleVerifyActivityWasFinished(a.activity, wf) != 2 {
		t.Fatal("finished")
	}
	a.activity.Status = 0
	if s.handleVerifyActivityWasFinished(a.activity, wf) != 0 {
		t.Fatal("created")
	}
	a.activity.Status = 1
	r.err = errors.New("db")
	if s.handleVerifyActivityWasFinished(a.activity, wf) != 0 {
		t.Fatal("runtime error")
	}
	r.err = nil
	r.runtime = runtime_entity.NewRuntime("other", 1, nil, "", "")
	if s.handleVerifyActivityWasFinished(a.activity, wf) != 1 {
		t.Fatal("other runtime")
	}
	r.runtime = runtime_entity.NewRuntime("k8s", 1, nil, "", "")
	j.response.Status.Active = 1
	if s.handleVerifyActivityWasFinished(a.activity, wf) != 1 {
		t.Fatal("active")
	}
	j.response.Status.Active = 0
	j.response.Status.Succeeded = 1
	if s.handleVerifyActivityWasFinished(a.activity, wf) != 2 || l.calls != 1 {
		t.Fatal("succeeded")
	}
	j.response = connector_job_k8s.ResponseGetJob{}
	if s.handleVerifyActivityWasFinished(a.activity, wf) != 0 {
		t.Fatal("missing")
	}
	j.response.Metadata.Name = "job"
	j.response.Status.Failed = 1
	if s.handleVerifyActivityWasFinished(a.activity, wf) != 2 {
		t.Fatal("failed")
	}
}
func TestMonitorPreActivityStates(t *testing.T) {
	s, a, r, j, _ := monitorFixture()
	act := a.activity
	act.DependsOn = []string{"dep"}
	wf := workflow_entity.Workflow{}
	r.err = errors.New("db")
	if s.handleVerifyPreActivityWasFinished(act, wf) != 0 {
		t.Fatal("runtime")
	}
	r.err = nil
	j.err = connector_job_k8s.ErrJobNotFound
	if s.handleVerifyPreActivityWasFinished(act, wf) != 0 {
		t.Fatal("not found")
	}
	j.err = errors.New("api")
	if s.handleVerifyPreActivityWasFinished(act, wf) != 0 {
		t.Fatal("api")
	}
	j.err = nil
	j.response.Status.Active = 1
	if s.handleVerifyPreActivityWasFinished(act, wf) != 1 {
		t.Fatal("active")
	}
	j.response.Status.Active = 0
	j.response.Status.Succeeded = 1
	if s.handleVerifyPreActivityWasFinished(act, wf) != 2 {
		t.Fatal("success")
	}
	j.response = connector_job_k8s.ResponseGetJob{}
	if s.handleVerifyPreActivityWasFinished(act, wf) != 0 {
		t.Fatal("missing")
	}
	j.response.Metadata.Name = "job"
	j.response.Status.Failed = 1
	if s.handleVerifyPreActivityWasFinished(act, wf) != 2 {
		t.Fatal("failed")
	}
	s.VerifyActivities(workflow_entity.Workflow{Spec: workflow_entity.WorkflowSpec{Activities: []workflow_activity_entity.WorkflowActivities{act}}})
}
