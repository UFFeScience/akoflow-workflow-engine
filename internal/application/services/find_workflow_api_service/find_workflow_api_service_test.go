package find_workflow_api_service

import (
	"testing"
	"time"

	"github.com/UFFeScience/akoflow/internal/api/requests"
	workflow_activity_entity "github.com/UFFeScience/akoflow/internal/domain/workflow/activity"
	workflow_entity "github.com/UFFeScience/akoflow/internal/domain/workflow/definition"
)

func TestParseTimestampAndDiskUsage(t *testing.T) {
	parsed, err := ParseTimestamp("2026-01-02 03:04:05")
	if err != nil || parsed.Year() != 2026 {
		t.Fatalf("parse failed: %v %v", parsed, err)
	}
	if _, err := ParseTimestamp("bad"); err == nil {
		t.Fatal("invalid timestamp must fail")
	}
	cases := map[string]int{"1Ki": 0, "2Mi": 2, "3Gi": 3072, "1Ti": 1048576, "42": 42, "bad": 0}
	for input, expected := range cases {
		if got := parseDiskUsage(input); got != expected {
			t.Errorf("%s: got %d want %d", input, got, expected)
		}
	}
}

func TestCalculateWorkflowMetricsDistributed(t *testing.T) {
	engine := workflow_entity.Workflow{Spec: workflow_entity.WorkflowSpec{
		StoragePolicy: workflow_entity.WorkflowSpecStoragePolicy{Type: workflow_entity.MODE_DISTRIBUTED, StorageSize: "2Gi"},
		Activities:    []workflow_activity_entity.WorkflowActivities{{CreatedAt: "2026-01-01 00:00:10"}},
	}}
	api := types_api.ApiWorkflowType{Spec: types_api.ApiWorkflowSpecType{Activities: []types_api.ApiWorkflowActivityType{
		{Id: 1, Name: "long", CreatedAt: "2026-01-01 00:00:00", StartedAt: "2026-01-01 00:00:00", FinishedAt: "2026-01-01 00:00:20"},
		{Id: 2, Name: "short", CreatedAt: "2026-01-01 00:00:10", StartedAt: "2026-01-01 00:00:10", FinishedAt: "2026-01-01 00:00:15"},
	}}}
	got := calculateWorkflowMetrics(api, engine)
	if got.Spec.StartExecution != "2026-01-01 00:00:00" || got.Spec.EndExecution != "2026-01-01 00:00:20" || got.Spec.ExecutionTime != "20" || got.Spec.LongestActivity.Name != "long" || got.Spec.DiskUsage != "2048" || got.Spec.Activities[0].ExecutionTime != "20" {
		t.Fatalf("unexpected metrics: %+v", got.Spec)
	}
}

func TestCalculateWorkflowMetricsStandaloneCountsPerActivity(t *testing.T) {
	stamp := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02 15:04:05")
	engine := workflow_entity.Workflow{Spec: workflow_entity.WorkflowSpec{StoragePolicy: workflow_entity.WorkflowSpecStoragePolicy{Type: workflow_entity.MODE_STANDALONE, StorageSize: "10Mi"}, Activities: []workflow_activity_entity.WorkflowActivities{{CreatedAt: stamp}}}}
	api := types_api.ApiWorkflowType{Spec: types_api.ApiWorkflowSpecType{Activities: []types_api.ApiWorkflowActivityType{{StartedAt: stamp, FinishedAt: stamp}, {StartedAt: stamp, FinishedAt: stamp}}}}
	if got := calculateWorkflowMetrics(api, engine); got.Spec.DiskUsage != "20" {
		t.Fatalf("got disk %s", got.Spec.DiskUsage)
	}
}
