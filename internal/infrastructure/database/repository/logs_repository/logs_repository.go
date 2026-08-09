package logs_repository

import (
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/schema"
)

type LogsRepository struct {
	tableName string
}

var TableName = "logs"
var Columns = "(ID INTEGER PRIMARY KEY AUTOINCREMENT, activity_id INTEGER, logs TEXT, created_at DATETIME DEFAULT CURRENT_TIMESTAMP)"

type LogsDatabase struct {
	ID         int
	ActivityId int
	Logs       string
	CreatedAt  string
}

func New() ILogsRepository {

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

type ILogsRepository interface {
	Create(params ParamsLogsCreate) error
}
