package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/UFFeScience/akoflow/internal/api/httpserver"
	"github.com/UFFeScience/akoflow/internal/application/services/run_activity_in_cluster_service"
	"github.com/UFFeScience/akoflow/internal/execution/lifecycle/eventloop"
	"github.com/UFFeScience/akoflow/internal/execution/lifecycle/garbagecollector"
	"github.com/UFFeScience/akoflow/internal/execution/lifecycle/healthcheck"
	"github.com/UFFeScience/akoflow/internal/execution/lifecycle/monitor"
	"github.com/UFFeScience/akoflow/internal/execution/orchestrator/engine"
	"github.com/UFFeScience/akoflow/internal/infrastructure/config"
)

func main() {

	config.SetupEnv()

	config.App().Logger.Info("Starting Akoflow Server")

	go healthcheck.New().StartHealthCheck()
	dispatcher := eventloop.NewDispatcher()
	if err := dispatcher.Register(
		eventloop.EventActivitySubmissionRequested,
		eventloop.NewLegacyActivityHandler(run_activity_in_cluster_service.New()),
	); err != nil {
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
	go orchestrator.StartOrchestrator()
	go monitor.StartMonitor()
	go garbagecollector.StartGarbageCollector()

	httpserver.StartServer()

}
