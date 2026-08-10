package logs_repository

import (
	"github.com/UFFeScience/akoflow/internal/application/ports"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository"
)

func (l *LogsRepository) Create(log ports.ActivityLog) error {

	database := repository.Database{}
	c := database.Connect()

	_, err := c.Exec(
		"INSERT INTO "+l.tableName+" (activity_id, logs) VALUES (?, ?)",
		log.ActivityID, log.Logs)

	if err != nil {
		return err
	}

	err = c.Close()

	if err != nil {
		return err
	}

	return nil

}
