package workflow_repository

import (
	"github.com/UFFeScience/akoflow/internal/domain/workflow/definition"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository"
)

func (w *WorkflowRepository) Create(namespace string, workflow workflow_entity.Workflow) (int, error) {

	database := repository.Database{}
	c := database.Connect()

	rawWorkflow := workflow.GetBase64Workflow()

	result, err := c.Exec(
		"INSERT INTO "+w.tableName+" (namespace, runtime, name, raw_workflow, status) VALUES (?, ?, ?, ?, ?)",
		namespace,
		workflow.Spec.Runtime,
		workflow.Name,
		rawWorkflow,
		StatusCreated,
	)

	defer c.Close()

	if err != nil {
		return 0, err
	}
	workflowId, _ := result.LastInsertId()

	err = c.Close()
	if err != nil {
		return 0, err
	}

	return int(workflowId), nil
}
