package get_workflow_by_status_service

import (
	"reflect"
	"testing"

	workflow_activity_entity "github.com/UFFeScience/akoflow/internal/domain/workflow/activity"
	workflow_entity "github.com/UFFeScience/akoflow/internal/domain/workflow/definition"
)

func TestSelectActivitiesByStatus(t *testing.T) {
	service := New()
	activities := []workflow_activity_entity.WorkflowActivities{{Id: 1, Status: 0}, {Id: 2, Status: 1}, {Id: 3, Status: 0}}
	workflow := workflow_entity.Workflow{Spec: workflow_entity.WorkflowSpec{Activities: activities}}

	fromWorkflow := service.GetActivitiesByStatus(workflow, 0)
	fromSlice := service.GetActivitiesByStatuses(activities, 0)
	want := []workflow_activity_entity.WorkflowActivities{activities[0], activities[2]}
	if !reflect.DeepEqual(fromWorkflow, want) || !reflect.DeepEqual(fromSlice, want) {
		t.Fatalf("fromWorkflow=%v fromSlice=%v want=%v", fromWorkflow, fromSlice, want)
	}
	if got := service.GetActivitiesByStatus(workflow, 99); len(got) != 0 {
		t.Fatalf("unknown status returned %v", got)
	}
}
