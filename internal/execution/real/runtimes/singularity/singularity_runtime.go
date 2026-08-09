package singularity_runtime

import (
	"fmt"

	"github.com/UFFeScience/akoflow/internal/domain/workflow/activity"
	"github.com/UFFeScience/akoflow/internal/domain/workflow/definition"
	"github.com/UFFeScience/akoflow/internal/execution/real/runtimes/singularity/singularity_runtime_service"
)

type SingularityRuntime struct {
	SingularityRuntimeService singularity_runtime_service.SingularityRuntimeService
}

func NewSingularityRuntime() *SingularityRuntime {
	return &SingularityRuntime{
		SingularityRuntimeService: singularity_runtime_service.NewSingularityRuntimeService(),
	}
}

func (s *SingularityRuntime) StartConnection() error {
	return nil
}

func (s *SingularityRuntime) StopConnection() error {
	return nil
}

func (s *SingularityRuntime) ApplyJob(workflowID int, activityID int) bool {
	s.SingularityRuntimeService.ApplyJob(workflowID, activityID)
	return true
}

func (s *SingularityRuntime) DeleteJob(workflowID int, activityID int) bool {
	return true
}

func (s *SingularityRuntime) GetMetrics(workflowID int, activityID int) string {
	return ""
}

func (s *SingularityRuntime) GetLogs(workflow workflow_entity.Workflow, workflowActivity workflow_activity_entity.WorkflowActivities) string {
	fmt.Println("[MonitorGetLogsActivityService] GetLogs - All implemented in one call")
	return ""
}

func (s *SingularityRuntime) GetStatus(workflowID int, activityID int) string {
	return ""
}

func (s *SingularityRuntime) VerifyActivitiesWasFinished(workflow workflow_entity.Workflow) bool {
	s.SingularityRuntimeService.VerifyActivitiesWasFinished(workflow)
	return true
}

func (s *SingularityRuntime) HealthCheck() bool {
	return true
}
