package monitor_collect_metrics_service

import (
	"errors"
	"testing"

	"github.com/UFFeScience/akoflow/internal/application/ports"
	workflow_activity_entity "github.com/UFFeScience/akoflow/internal/domain/workflow/activity"
	workflow_entity "github.com/UFFeScience/akoflow/internal/domain/workflow/definition"
)

type pendingFake struct {
	workflows []workflow_entity.Workflow
	err       error
}

func (f pendingFake) GetPendingWorkflows() ([]workflow_entity.Workflow, error) {
	return f.workflows, f.err
}

type statusFake struct{}

func (statusFake) GetActivitiesByStatus(w workflow_entity.Workflow, status int) []workflow_activity_entity.WorkflowActivities {
	r := []workflow_activity_entity.WorkflowActivities{}
	for _, a := range w.Spec.Activities {
		if a.Status == status {
			r = append(r, a)
		}
	}
	return r
}

type runtimeFake struct{ logs, metrics int }

func (f *runtimeFake) GetLogs(workflow_entity.Workflow, workflow_activity_entity.WorkflowActivities) string {
	f.logs++
	return "logs"
}
func (f *runtimeFake) GetMetrics(int, int) string { f.metrics++; return "metrics" }

func TestCollectMetricsOnlyForRunningActivities(t *testing.T) {
	wf := workflow_entity.Workflow{Id: 1, Spec: workflow_entity.WorkflowSpec{Activities: []workflow_activity_entity.WorkflowActivities{{Id: 1, Status: ports.ActivityStatusRunning, Runtime: "local"}, {Id: 2, Status: ports.ActivityStatusFinished, Runtime: "local"}}}}
	runtime := &runtimeFake{}
	service := NewWithDependencies(pendingFake{workflows: []workflow_entity.Workflow{wf}}, statusFake{}, func(string) RuntimeMetrics { return runtime })
	if err := service.CollectMetrics(); err != nil || runtime.logs != 1 || runtime.metrics != 1 {
		t.Fatalf("runtime=%+v err=%v", runtime, err)
	}
}
func TestCollectMetricsReturnsPendingAndRuntimeErrors(t *testing.T) {
	service := NewWithDependencies(pendingFake{err: errors.New("db")}, statusFake{}, nil)
	if err := service.CollectMetrics(); err == nil {
		t.Fatal("expected pending error")
	}
	wf := workflow_entity.Workflow{Spec: workflow_entity.WorkflowSpec{Activities: []workflow_activity_entity.WorkflowActivities{{Status: ports.ActivityStatusRunning, Runtime: "gone"}}}}
	service = NewWithDependencies(pendingFake{workflows: []workflow_entity.Workflow{wf}}, statusFake{}, func(string) RuntimeMetrics { return nil })
	if err := service.CollectMetrics(); err == nil {
		t.Fatal("expected runtime error")
	}
}
func TestNewInitializesDependencies(t *testing.T) {
	s := New()
	if s.getPendingWorkflowService == nil || s.getWorkflowByStatus == nil || s.runtimeResolver == nil {
		t.Fatalf("incomplete: %+v", s)
	}
}
