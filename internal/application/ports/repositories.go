package ports

import (
	workflow_activity_entity "github.com/UFFeScience/akoflow/internal/domain/workflow/activity"
	workflow_entity "github.com/UFFeScience/akoflow/internal/domain/workflow/definition"
)

const (
	ActivityStatusCreated = iota
	ActivityStatusRunning
	ActivityStatusFinished
	ActivityStatusCompleted
	ActivityStatusSyncing
)

const (
	WorkflowStatusCreated = iota
	WorkflowStatusRunning
	WorkflowStatusFinished
)

const (
	StorageStatusNotCreated = iota + 1
	StorageStatusCreated
	StorageStatusCompleted
)

const (
	ExecutionStatusRunning = iota + 1
	ExecutionStatusCompleted
	ExecutionStatusFailed
	ExecutionStatusCancelled
)

type ActivitiesByWorkflow map[int][]workflow_activity_entity.WorkflowActivities

type ActivityRepository interface {
	Create(string, workflow_entity.Workflow, []workflow_activity_entity.WorkflowActivities) error
	GetActivitiesByWorkflowIds([]int) (ActivitiesByWorkflow, error)
	UpdateStatus(int, int) error
	UpdateProcID(int, string) error
	Find(int) (workflow_activity_entity.WorkflowActivities, error)
	GetByWorkflowId(int) ([]workflow_activity_entity.WorkflowActivities, error)
	GetWfaDependencies(int) ([]workflow_activity_entity.WorkflowActivityDependencyDatabase, error)
	FindPreActivity(int) (workflow_activity_entity.WorkflowPreActivityDatabase, error)
	UpdatePreActivity(int, workflow_activity_entity.WorkflowPreActivityDatabase) error
	GetPreactivitiesCompleted() ([]workflow_activity_entity.WorkflowPreActivityDatabase, error)
	UpdateResourceSelector(int, string) error
	SetActivitySchedule(int, int, string, string, float64, float64, string) error
	GetActivitySchedulesByResourceID(string) ([]workflow_activity_entity.ActivitySchedule, error)
	GetAllRunningActivities() ([]workflow_activity_entity.WorkflowActivities, error)
	GetActivityScheduleByActivityId(int) (workflow_activity_entity.ActivitySchedule, error)
	IsActivityScheduled(int, int) (bool, error)
}

type WorkflowListOptions struct {
	All     bool
	Page    *int
	PerPage *int
}

type WorkflowRepository interface {
	Create(string, workflow_entity.Workflow) (int, error)
	Find(int) (workflow_entity.Workflow, error)
	GetPendingWorkflows(string) ([]workflow_entity.Workflow, error)
	UpdateStatus(int, int) error
	ListAllWorkflows(*WorkflowListOptions) ([]workflow_entity.Workflow, error)
}

type Storage struct {
	ID                     int
	WorkflowID             int
	ActivityID             int
	PVCName                *string
	Namespace              string
	Status                 int
	StorageMountPath       string
	StorageClass           string
	StorageSize            string
	InitialFileList        string
	EndFileList            string
	InitialDiskSpec        string
	EndDiskSpec            string
	KeepStorageAfterFinish int
	Detached               *string
	CreatedAt              string
}

type CreateStorageParams struct {
	WorkflowID            int
	Namespace             string
	Status                int
	StorageMountPath      string
	StorageClass          string
	StorageSize           string
	ActivitiesKeepingDisk map[int]bool
}

type UpdateStorageParams struct {
	Status     int
	PVCName    string
	ActivityID int
}

type StorageRepository interface {
	Create(CreateStorageParams) error
	Update(UpdateStorageParams) error
	Find(int) (Storage, error)
	FindByWorkflow(int) ([]Storage, error)
	GetCreatedStorages(string) ([]Storage, error)
	UpdateInitialFileListDisk(int, string) error
	UpdateEndFileListDisk(int, string) error
	UpdateInitialDiskSpec(int, string) error
	UpdateEndDiskSpec(int, string) error
	UpdateDetached(int) error
}

type ActivityMetric struct {
	ID, ActivityID                 int
	CPU, Memory, Window, Timestamp string
	CreatedAt                      string
}

type MetricsRepository interface{ Create(ActivityMetric) error }

type ActivityLog struct {
	ID, ActivityID  int
	Logs, CreatedAt string
}

type LogsRepository interface{ Create(ActivityLog) error }
