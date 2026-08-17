package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/UFFeScience/akoflow/internal/api/handlers/workflow_engine_api_handler"
	"github.com/UFFeScience/akoflow/internal/api/httpserver"
	applicationexecution "github.com/UFFeScience/akoflow/internal/application/execution"
	"github.com/UFFeScience/akoflow/internal/application/ports"
	"github.com/UFFeScience/akoflow/internal/controlplane/eventloop"
	controlexecution "github.com/UFFeScience/akoflow/internal/controlplane/execution"
	"github.com/UFFeScience/akoflow/internal/infrastructure/config"
	"github.com/UFFeScience/akoflow/internal/infrastructure/config/logger"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database"
	dbenvironment "github.com/UFFeScience/akoflow/internal/infrastructure/database/environment"
	dbexecution "github.com/UFFeScience/akoflow/internal/infrastructure/database/execution"
	dbplanning "github.com/UFFeScience/akoflow/internal/infrastructure/database/planning"
	dbqueue "github.com/UFFeScience/akoflow/internal/infrastructure/database/queue"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/schema"
	dbworkflow "github.com/UFFeScience/akoflow/internal/infrastructure/database/workflow"
	planningplugin "github.com/UFFeScience/akoflow/internal/infrastructure/plugins/planning"
	"github.com/UFFeScience/akoflow/internal/provider"
	"github.com/UFFeScience/akoflow/internal/provider/kubernetes"
	"github.com/UFFeScience/akoflow/internal/provider/local"
	"github.com/UFFeScience/akoflow/internal/provider/registry"
	"github.com/UFFeScience/akoflow/internal/provider/simgrid"
	"github.com/UFFeScience/akoflow/internal/provider/slurm"
)

func main() {

	settings := config.Load()
	log := logger.NewStdLogger()
	log.Info("Starting Akoflow Server")

	dispatcher := eventloop.NewDispatcher()
	db := (&database.Database{}).Connect()
	defer db.Close()
	if err := schema.Apply(db); err != nil {
		panic(err)
	}
	events, err := dbqueue.New(db)
	if err != nil {
		panic(err)
	}
	runtimes := registry.New()
	if err := runtimes.Register("*", simgrid.NewActivityRuntime()); err != nil {
		panic(err)
	}
	commandExecutor := provider.OSCommandExecutor{}
	for runtimeID, adapter := range map[string]ports.RuntimeAdapter{
		"local":      local.New(),
		"kubernetes": kubernetes.New(commandExecutor, settings.DefaultNamespace),
		"slurm":      slurm.New(commandExecutor, ""),
	} {
		if err := runtimes.Register(runtimeID, adapter); err != nil {
			panic(err)
		}
	}
	executionStore := dbexecution.New(db)
	activities := applicationexecution.New(runtimes, executionStore)
	supervisor, err := controlexecution.New(executionStore, activities,
		simgrid.NewSimulationExecutor(), controlexecution.Config{PollInterval: time.Second, MaxParallel: 8})
	if err != nil {
		panic(err)
	}
	if err := dispatcher.Register(eventloop.EventExecutionRunRequested,
		eventloop.NewExecutionRunHandler(supervisor)); err != nil {
		panic(err)
	}
	owner, _ := os.Hostname()
	owner = fmt.Sprintf("%s-%d", owner, os.Getpid())
	loopConfig := eventloop.DefaultConfig(owner)
	loopConfig.PollInterval = 500 * time.Millisecond
	loop, err := eventloop.New(events, dispatcher, loopConfig)
	if err != nil {
		panic(err)
	}
	go func() {
		if err := loop.Run(context.Background()); err != nil {
			log.Error("Event loop stopped:", err)
		}
	}()
	api, err := workflow_engine_api_handler.New(workflow_engine_api_handler.Dependencies{
		Environments: dbenvironment.New(db), Workflows: dbworkflow.New(db),
		Plans: dbplanning.New(db), Events: events,
		Validator: planningplugin.NewValidator(),
	})
	if err != nil {
		panic(err)
	}
	if err := httpserver.StartServer(settings.HTTPAddress, api); err != nil {
		panic(err)
	}

}
