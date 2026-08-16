package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/UFFeScience/akoflow/internal/api/httpserver"
	"github.com/UFFeScience/akoflow/internal/application/services/activity_execution_service"
	"github.com/UFFeScience/akoflow/internal/execution/lifecycle/eventloop"
	"github.com/UFFeScience/akoflow/internal/execution/orchestrator"
	simulation "github.com/UFFeScience/akoflow/internal/execution/simulation"
	"github.com/UFFeScience/akoflow/internal/infrastructure/config"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/activity_handle_repository"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/schema"
)

func main() {

	config.SetupEnv()

	config.App().Logger.Info("Starting Akoflow Server")

	dispatcher := eventloop.NewDispatcher()
	db := (&repository.Database{}).Connect()
	defer db.Close()
	if err := schema.Apply(db); err != nil {
		panic(err)
	}
	runtimes := orchestrator.NewRuntimeRegistry()
	if err := runtimes.Register("*", simulation.NewActivityRuntime()); err != nil {
		panic(err)
	}
	executions := activity_execution_service.New(runtimes, activity_handle_repository.New(db))
	if err := dispatcher.Register(eventloop.EventActivityExecutionRequested,
		eventloop.NewActivityExecutionHandler(executions)); err != nil {
		panic(err)
	}
	owner, _ := os.Hostname()
	owner = fmt.Sprintf("%s-%d", owner, os.Getpid())
	loopConfig := eventloop.DefaultConfig(owner)
	loopConfig.PollInterval = 500 * time.Millisecond
	loop, err := eventloop.New(config.App().Repository.QueueRepository, dispatcher, loopConfig)
	if err != nil {
		panic(err)
	}
	go func() {
		if err := loop.Run(context.Background()); err != nil {
			config.App().Logger.Error("Event loop stopped:", err)
		}
	}()
	httpserver.StartServer()

}
