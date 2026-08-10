package metrics_repository

import (
	"github.com/UFFeScience/akoflow/internal/application/ports"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/schema"
)

type MetricsRepository struct {
	tableName string
}

var TableName = "metrics"
var Columns = "(ID INTEGER PRIMARY KEY AUTOINCREMENT, activity_id INTEGER, cpu TEXT, memory TEXT, window TEXT, timestamp TEXT, created_at DATETIME DEFAULT CURRENT_TIMESTAMP)"

type MetricsDatabase = ports.ActivityMetric

func New() ports.MetricsRepository {

	database := repository.Database{}
	c := database.Connect()
	defer c.Close()
	if err := schema.Apply(c); err != nil {
		return nil
	}

	return &MetricsRepository{
		tableName: TableName,
	}
}

type IMetricsRepository = ports.MetricsRepository
