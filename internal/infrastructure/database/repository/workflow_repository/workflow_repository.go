package workflow_repository

import (
	"github.com/UFFeScience/akoflow/internal/application/ports"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/schema"
)

type WorkflowRepository struct {
	tableName string
}

var TableName = "workflows"
var Columns = "(id INTEGER PRIMARY KEY AUTOINCREMENT, namespace TEXT, runtime TEXT, name TEXT, raw_workflow TEXT, status INTEGER)"

const StatusCreated = ports.WorkflowStatusCreated
const StatusRunning = ports.WorkflowStatusRunning
const StatusFinished = ports.WorkflowStatusFinished

func New() ports.WorkflowRepository {

	database := repository.Database{}
	c := database.Connect()

	err := schema.Apply(c)

	if err != nil {
		println("Error creating table", err.Error())
		return nil
	}

	err = c.Close()
	if err != nil {
		println("Error closing connection", err.Error())
		return nil
	}

	return &WorkflowRepository{tableName: TableName}
}

type IWorkflowRepository = ports.WorkflowRepository
