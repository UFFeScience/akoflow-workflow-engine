package logs_repository

import (
	"github.com/UFFeScience/akoflow/internal/application/ports"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/schema"
)

type LogsRepository struct {
	tableName string
}

var TableName = "logs"
var Columns = "(ID INTEGER PRIMARY KEY AUTOINCREMENT, activity_id INTEGER, logs TEXT, created_at DATETIME DEFAULT CURRENT_TIMESTAMP)"

type LogsDatabase = ports.ActivityLog

func New() ports.LogsRepository {

	database := repository.Database{}
	c := database.Connect()
	defer c.Close()
	if err := schema.Apply(c); err != nil {
		return nil
	}

	return &LogsRepository{
		tableName: TableName,
	}
}

type ILogsRepository = ports.LogsRepository
