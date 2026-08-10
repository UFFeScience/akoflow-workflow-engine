package metrics_repository

import (
	"github.com/UFFeScience/akoflow/internal/application/ports"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository"
)

func (m *MetricsRepository) Create(metric ports.ActivityMetric) error {

	database := repository.Database{}
	c := database.Connect()

	_, err := c.Exec(
		"INSERT INTO "+m.tableName+" (activity_id, cpu, memory, window, timestamp) VALUES (?, ?, ?, ?, ?)",
		metric.ActivityID, metric.CPU, metric.Memory, metric.Window, metric.Timestamp)

	if err != nil {
		return err
	}

	err = c.Close()
	if err != nil {
		return err
	}

	return nil
}
