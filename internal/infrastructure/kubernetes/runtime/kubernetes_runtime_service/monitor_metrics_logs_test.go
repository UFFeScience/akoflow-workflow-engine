package kubernetes_runtime_service

import (
	"github.com/UFFeScience/akoflow/internal/application/ports"
	"github.com/UFFeScience/akoflow/internal/domain"
	runtime_entity "github.com/UFFeScience/akoflow/internal/domain/resource/runtime"
	workflow_activity_entity "github.com/UFFeScience/akoflow/internal/domain/workflow/activity"
	workflow_entity "github.com/UFFeScience/akoflow/internal/domain/workflow/definition"
	connector_k8s "github.com/UFFeScience/akoflow/internal/infrastructure/kubernetes/connector"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type monitorMetricsRepo struct {
	ports.MetricsRepository
	calls int
}

func (r *monitorMetricsRepo) Create(ports.ActivityMetric) error { r.calls++; return nil }

type monitorLogRepo struct {
	ports.LogsRepository
	calls int
}

func (r *monitorLogRepo) Create(ports.ActivityLog) error { r.calls++; return nil }
func TestMonitorMetricsAndLogsSuccess(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "metrics.k8s.io") && strings.Contains(r.URL.Path, "/pods/"):
			_, _ = w.Write([]byte(`{"window":"30s","containers":[{"usage":{"cpu":"2m","memory":"4Mi"}}]}`))
		case strings.Contains(r.URL.Path, "metrics.k8s.io"):
			_, _ = w.Write([]byte(`{"items":[{"metadata":{"name":"node-a"},"usage":{"cpu":"10m","memory":"1024Ki"}}]}`))
		case strings.HasSuffix(r.URL.Path, "/log"):
			_, _ = w.Write([]byte("activity logs"))
		default:
			_, _ = w.Write([]byte(`{"items":[{"metadata":{"name":"pod-a"}}]}`))
		}
	}))
	defer server.Close()
	rt := runtime_entity.NewRuntime("k8s", 1, map[string]string{"K8S_API_SERVER_HOST": strings.TrimPrefix(server.URL, "https://"), "K8S_ENVIRONMENT_VERSION_ID": "env"}, "", "")
	rr := &healthRuntimeRepo{runtime: rt}
	a := workflow_activity_entity.WorkflowActivities{Id: 3, WorkflowId: 2, Name: "task", Runtime: "k8s"}
	ar := &jobActivityRepo{activity: a}
	wr := &jobWorkflowRepo{workflow: workflow_entity.Workflow{Id: 2}}
	metrics := &monitorMetricsRepo{}
	logs := &monitorLogRepo{}
	resources := &healthResources{resource: &domain.Resource{ID: "r1"}}
	m := &MonitorGetMetricsActivityService{namespace: "ns", metricsRepository: metrics, resourceRepository: resources, workflowRepository: wr, activityRepository: ar, runtimeRepository: rr, connector: connector_k8s.New()}
	m.GetMetrics(2, 3)
	if metrics.calls != 1 || resources.snapshots != 1 {
		t.Fatalf("metrics=%d snapshots=%d", metrics.calls, resources.snapshots)
	}
	l := &MonitorGetLogsActivityService{namespace: "ns", logsRepository: logs, runtimeRepository: rr, connector: connector_k8s.New()}
	l.GetLogs(workflow_entity.Workflow{Id: 2}, a)
	if logs.calls != 1 {
		t.Fatal("logs")
	}
}
