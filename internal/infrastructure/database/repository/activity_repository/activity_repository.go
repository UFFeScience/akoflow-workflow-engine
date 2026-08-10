package activity_repository

import (
	"github.com/UFFeScience/akoflow/internal/application/ports"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/schema"
)

type ActivityRepository struct {
	tableNameActivity             string
	tableNameActivityDependencies string
	tableNamePreActivity          string
	tableNameActivitySchedule     string
}

const StatusCreated = ports.ActivityStatusCreated
const StatusRunning = ports.ActivityStatusRunning
const StatusFinished = ports.ActivityStatusFinished
const StatusCompleted = ports.ActivityStatusCompleted
const StatusSyncing = ports.ActivityStatusSyncing

func New() ports.ActivityRepository {
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

type IActivityRepository = ports.ActivityRepository
