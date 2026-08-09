package activity_repository

import (
	"github.com/UFFeScience/akoflow/internal/domain/workflow/activity"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository"
)

func (w *ActivityRepository) UpdatePreActivity(id int, preactivity workflow_activity_entity.WorkflowPreActivityDatabase) error {
	database := repository.Database{}
	c := database.Connect()

	_, err := c.Exec(
		"UPDATE "+w.tableNamePreActivity+" SET status = ?, log = ?, resource_k8s_base64 = ? WHERE activity_id = ?",
		preactivity.Status, preactivity.Log, preactivity.ResourceK8sBase64, id)

	if err != nil {
		return err
	}

	err = c.Close()

	if err != nil {
		return err
	}

	return nil
}
