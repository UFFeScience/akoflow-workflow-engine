package execution

import (
	"github.com/UFFeScience/akoflow/internal/domain/planning"
	"github.com/UFFeScience/akoflow/internal/domain/resource"
	"github.com/UFFeScience/akoflow/internal/domain/workflow"
)

type ExecutionRunStatus string
type TaskExecutionStatus string

const (
	ExecutionRunCreated   ExecutionRunStatus = "created"
	ExecutionRunRunning   ExecutionRunStatus = "running"
	ExecutionRunCompleted ExecutionRunStatus = "completed"
	ExecutionRunFailed    ExecutionRunStatus = "failed"

	TaskBlocked   TaskExecutionStatus = "blocked"
	TaskReady     TaskExecutionStatus = "ready"
	TaskPreparing TaskExecutionStatus = "preparing"
	TaskRunning   TaskExecutionStatus = "running"
	TaskCompleted TaskExecutionStatus = "completed"
	TaskFailed    TaskExecutionStatus = "failed"
)

type ExecutionRun struct {
	ID                    string                 `json:"id"`
	SchedulePlanID        string                 `json:"schedulePlanId"`
	Mode                  planning.ExecutionMode `json:"mode"`
	Seed                  int64                  `json:"seed"`
	Status                ExecutionRunStatus     `json:"status"`
	EnvironmentSnapshotID string                 `json:"environmentSnapshotId,omitempty"`
}

type TaskExecution struct {
	ID                  string              `json:"id"`
	ExecutionRunID      string              `json:"executionRunId"`
	PlanAssignmentID    string              `json:"planAssignmentId"`
	ActivityID          string              `json:"activityId"`
	PlannedResourceID   string              `json:"plannedResourceId"`
	AllocatedResourceID string              `json:"allocatedResourceId,omitempty"`
	Attempt             int                 `json:"attempt"`
	Status              TaskExecutionStatus `json:"status"`
	ReadyAt             float64             `json:"readyAt"`
	DataReadyAt         float64             `json:"dataReadyAt"`
	QueuedAt            float64             `json:"queuedAt"`
	StartedAt           float64             `json:"startedAt"`
	FinishedAt          float64             `json:"finishedAt"`
	RuntimeSeconds      float64             `json:"runtimeSeconds"`
	QueueSeconds        float64             `json:"queueSeconds"`
	TransferSeconds     float64             `json:"transferSeconds"`
	InterferenceSeconds float64             `json:"interferenceSeconds"`
	OverheadSeconds     float64             `json:"overheadSeconds"`
	Cost                float64             `json:"cost"`
	FailureReason       string              `json:"failureReason,omitempty"`
}

type ExecutionMetrics struct {
	MakespanSeconds     float64 `json:"makespanSeconds"`
	Cost                float64 `json:"cost"`
	ComputeSeconds      float64 `json:"computeSeconds"`
	TransferSeconds     float64 `json:"transferSeconds"`
	QueueSeconds        float64 `json:"queueSeconds"`
	InterferenceSeconds float64 `json:"interferenceSeconds"`
	OverheadSeconds     float64 `json:"overheadSeconds"`
	Feasible            bool    `json:"feasible"`
}

type ExecutionTrace struct {
	RunID     string                    `json:"runId"`
	PlanID    string                    `json:"planId"`
	Mode      planning.ExecutionMode    `json:"mode"`
	Predicted planning.PredictedMetrics `json:"predicted"`
	Executed  ExecutionMetrics          `json:"executed"`
	Tasks     []TaskExecution           `json:"tasks"`
	Transfers []DataTransfer            `json:"transfers"`
}

type DataTransfer struct {
	ID                 string  `json:"id"`
	ExecutionRunID     string  `json:"executionRunId"`
	ProducerActivityID string  `json:"producerActivityId"`
	ConsumerActivityID string  `json:"consumerActivityId"`
	SourceResourceID   string  `json:"sourceResourceId"`
	TargetResourceID   string  `json:"targetResourceId"`
	Bytes              int64   `json:"bytes"`
	StartedAt          float64 `json:"startedAt"`
	FinishedAt         float64 `json:"finishedAt"`
	DurationSeconds    float64 `json:"durationSeconds"`
	Cost               float64 `json:"cost"`
}

type ActivityExecutionContext struct {
	Run        ExecutionRun             `json:"run"`
	Workflow   workflow.WorkflowVersion `json:"workflow"`
	Activity   workflow.Activity        `json:"activity"`
	Assignment planning.PlanAssignment  `json:"assignment"`
	Resource   resource.Resource        `json:"resource"`
}
