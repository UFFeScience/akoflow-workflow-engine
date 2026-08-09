package get_workflow_by_status_service

import (
	"testing"

	"github.com/UFFeScience/akoflow/internal/domain/workflow/activity"
	"github.com/UFFeScience/akoflow/internal/domain/workflow/definition"
	"github.com/stretchr/testify/require"
)

func TestGetActivitiesByStatusFiltersWorkflowActivities(t *testing.T) {
	service := New()
	workflow := workflow_entity.Workflow{Spec: workflow_entity.WorkflowSpec{Activities: []workflow_activity_entity.WorkflowActivities{
		{Id: 1, Status: 1}, {Id: 2, Status: 2}, {Id: 3, Status: 1},
	}}}
	selected := service.GetActivitiesByStatus(workflow, 1)
	require.Equal(t, []int{1, 3}, []int{selected[0].Id, selected[1].Id})
}

func TestGetActivitiesByStatusesFiltersProvidedSlice(t *testing.T) {
	service := New()
	selected := service.GetActivitiesByStatuses([]workflow_activity_entity.WorkflowActivities{
		{Id: 1, Status: 0}, {Id: 2, Status: 2},
	}, 2)
	require.Len(t, selected, 1)
	require.Equal(t, 2, selected[0].Id)
}
