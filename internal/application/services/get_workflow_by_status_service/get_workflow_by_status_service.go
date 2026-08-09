package get_workflow_by_status_service

import (
	"github.com/UFFeScience/akoflow/internal/domain/workflow/activity"
	"github.com/UFFeScience/akoflow/internal/domain/workflow/definition"
)

type GetWorkflowByStatusService struct {
}

func New() GetWorkflowByStatusService {
	return GetWorkflowByStatusService{}
}

func (o *GetWorkflowByStatusService) GetActivitiesByStatus(wfs workflow_entity.Workflow, status int) []workflow_activity_entity.WorkflowActivities {
	var wfsSelected []workflow_activity_entity.WorkflowActivities
	for _, activity := range wfs.Spec.Activities {
		if activity.Status == status {
			wfsSelected = append(wfsSelected, activity)
		}
	}
	return wfsSelected
}

func (o *GetWorkflowByStatusService) GetActivitiesByStatuses(wfas []workflow_activity_entity.WorkflowActivities, status int) []workflow_activity_entity.WorkflowActivities {
	var wfsSelected []workflow_activity_entity.WorkflowActivities
	for _, activity := range wfas {
		if activity.Status == status {
			wfsSelected = append(wfsSelected, activity)
		}
	}
	return wfsSelected
}
