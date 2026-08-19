package main

import (
	"github.com/UFFeScience/akoflow/internal/api/handlers/workflow_engine_api_handler"
	planningplugin "github.com/UFFeScience/akoflow/internal/infrastructure/plugins/planning"
)

func buildAPI(storage persistence) (*workflow_engine_api_handler.Handler, error) {
	return workflow_engine_api_handler.New(workflow_engine_api_handler.Dependencies{
		Environments: storage.environments,
		Workflows:    storage.workflows,
		Plans:        storage.plans,
		Events:       storage.events,
		Validator:    planningplugin.NewValidator(),
		Executions:   storage.executions,
		Topologies:   storage.topologies,
		Data:         storage.data,
		Resources:    storage.resources,
		Instance:     storage.instance,
	})
}
