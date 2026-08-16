package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/UFFeScience/akoflow/internal/api/handlers/workflow_engine_api_handler"
	"github.com/UFFeScience/akoflow/internal/api/httpserver"
	"github.com/UFFeScience/akoflow/internal/application/ports"
	"github.com/UFFeScience/akoflow/internal/application/services/activity_execution_service"
	"github.com/UFFeScience/akoflow/internal/application/services/execution_supervisor"
	"github.com/UFFeScience/akoflow/internal/execution/lifecycle/eventloop"
	"github.com/UFFeScience/akoflow/internal/execution/orchestrator"
	simulation "github.com/UFFeScience/akoflow/internal/execution/simulation"
	"github.com/UFFeScience/akoflow/internal/infrastructure/config"
	"github.com/UFFeScience/akoflow/internal/infrastructure/config/logger"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/environment_repository"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/execution_repository"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/queue_repository"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/schedule_plan_repository"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/workflow_definition_repository"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/schema"
	planningplugin "github.com/UFFeScience/akoflow/internal/infrastructure/plugins/planning"
	runtimecommon "github.com/UFFeScience/akoflow/internal/runtime"
	kubernetesruntime "github.com/UFFeScience/akoflow/internal/runtime/kubernetes"
	localruntime "github.com/UFFeScience/akoflow/internal/runtime/local"
	slurmruntime "github.com/UFFeScience/akoflow/internal/runtime/slurm"
)

func main() {

	settings := config.Load()
	log := logger.NewStdLogger()
	log.Info("Starting Akoflow Server")

	dispatcher := eventloop.NewDispatcher()
	db := (&repository.Database{}).Connect()
	defer db.Close()
	if err := schema.Apply(db); err != nil {
		panic(err)
	}
	events, err := queue_repository.NewWithDB(db)
	if err != nil {
		panic(err)
	}
	runtimes := orchestrator.NewRuntimeRegistry()
	if err := runtimes.Register("*", simulation.NewActivityRuntime()); err != nil {
		panic(err)
	}
	commandExecutor := runtimecommon.OSCommandExecutor{}
	for runtimeID, adapter := range map[string]ports.RuntimeAdapter{
		"local":      localruntime.New(),
		"kubernetes": kubernetesruntime.New(commandExecutor, settings.DefaultNamespace),
		"slurm":      slurmruntime.New(commandExecutor, ""),
	} {
		if err := runtimes.Register(runtimeID, adapter); err != nil {
			panic(err)
		}
	}
	executionStore := execution_repository.New(db)
	activities := activity_execution_service.New(runtimes, executionStore)
	supervisor, err := execution_supervisor.New(executionStore, activities,
		simulation.NewSimulationExecutor(), execution_supervisor.Config{PollInterval: time.Second, MaxParallel: 8})
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
		Environments: environment_repository.NewWithDB(db), Workflows: workflow_definition_repository.NewWithDB(db),
		Plans: schedule_plan_repository.NewWithDB(db), Events: events,
		Validator: planningplugin.NewValidatePlanService(),
	})
	if err != nil {
		panic(err)
	}
	if err := httpserver.StartServer(settings.HTTPAddress, api); err != nil {
		panic(err)
	}

}
