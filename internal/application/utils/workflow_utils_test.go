package utils

import (
	"strings"
	"testing"

	workflow_entity "github.com/UFFeScience/akoflow/internal/domain/workflow/definition"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/activity_repository"
)

func TestWorkflowHydrationAndIDs(t *testing.T) {
	workflows := []workflow_entity.Workflow{{Id: 1}, {Id: 2}}
	if ids := GetIds(workflows); len(ids) != 2 || ids[0] != 1 || ids[1] != 2 {
		t.Fatalf("unexpected ids: %v", ids)
	}
	activities := activity_repository.ResultGetActivitiesByWorkflowIds{1: {{Id: 10, Name: "a"}}}
	hydrated := HydrateWorkflows(workflows, activities)
	if len(hydrated) != 1 || len(hydrated[0].Spec.Activities) != 1 {
		t.Fatalf("unexpected hydration: %+v", hydrated)
	}
	if got := HydrateWorkflow(workflows[0], activities); got.Spec.Activities[0].Name != "a" {
		t.Fatal("single hydration failed")
	}
	unchanged := HydrateWorkflow(workflows[1], activities)
	if len(unchanged.Spec.Activities) != 0 {
		t.Fatal("missing map entry must preserve workflow")
	}
}

func TestParseTimestamp(t *testing.T) {
	if got := ParseTimestamp("1970-01-01 00:00:00"); got != "0" {
		t.Fatalf("got %s", got)
	}
	if got := ParseTimestamp("invalid"); !strings.HasPrefix(got, "Error parsing timestamp:") {
		t.Fatalf("unexpected error: %s", got)
	}
}
