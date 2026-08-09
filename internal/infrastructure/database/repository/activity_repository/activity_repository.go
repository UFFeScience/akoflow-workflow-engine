package activity_repository

import (
	"github.com/UFFeScience/akoflow/internal/domain/workflow/activity"
	"github.com/UFFeScience/akoflow/internal/domain/workflow/definition"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/schema"
)

type ActivityRepository struct {
	tableNameActivity             string
	tableNameActivityDependencies string
	tableNamePreActivity          string
	tableNameActivitySchedule     string
}

var StatusCreated = 0
var StatusRunning = 1
var StatusFinished = 2
var StatusCompleted = 3
var StatusSyncing = 4

func New() IActivityRepository {
	database := repository.Database{}
	c := database.Connect()
	defer c.Close()
	if err := schema.Apply(c); err != nil {
		return nil
	}

	return &ActivityRepository{
		tableNameActivity:             "activities",
		tableNameActivityDependencies: "activities_dependencies",
		tableNamePreActivity:          "pre_activities",
		tableNameActivitySchedule:     "activities_schedules",
	}
}

type IActivityRepository interface {
	Create(namespace string, workflow workflow_entity.Workflow, activities []workflow_activity_entity.WorkflowActivities) error
	GetActivitiesByWorkflowIds(ids []int) (ResultGetActivitiesByWorkflowIds, error)
	UpdateStatus(id int, status int) error
	UpdateProcID(id int, pid string) error
	Find(id int) (workflow_activity_entity.WorkflowActivities, error)
	GetByWorkflowId(id int) ([]workflow_activity_entity.WorkflowActivities, error)
	GetWfaDependencies(workflowId int) ([]workflow_activity_entity.WorkflowActivityDependencyDatabase, error)
	FindPreActivity(id int) (workflow_activity_entity.WorkflowPreActivityDatabase, error)
	UpdatePreActivity(id int, preactivity workflow_activity_entity.WorkflowPreActivityDatabase) error
	GetPreactivitiesCompleted() ([]workflow_activity_entity.WorkflowPreActivityDatabase, error)
	UpdateResourceSelector(id int, resourceSelector string) error
	SetActivitySchedule(workflowId int, activity int, resourceID string, scheduleName string, cpuRequired float64, memoryRequired float64, metadata string) error
	GetActivitySchedulesByResourceID(resourceID string) ([]workflow_activity_entity.ActivitySchedule, error)
	GetAllRunningActivities() ([]workflow_activity_entity.WorkflowActivities, error)
	GetActivityScheduleByActivityId(activityId int) (workflow_activity_entity.ActivitySchedule, error)
	IsActivityScheduled(workflowId int, activityId int) (bool, error)
}
