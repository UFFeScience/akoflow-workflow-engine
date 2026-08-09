package main

import (
	"github.com/UFFeScience/akoflow/internal/api/httpserver"
	"github.com/UFFeScience/akoflow/internal/execution/lifecycle/garbagecollector"
	"github.com/UFFeScience/akoflow/internal/execution/lifecycle/healthcheck"
	"github.com/UFFeScience/akoflow/internal/execution/lifecycle/monitor"
	"github.com/UFFeScience/akoflow/internal/execution/orchestrator/engine"
	"github.com/UFFeScience/akoflow/internal/execution/orchestrator/worker"
	"github.com/UFFeScience/akoflow/internal/infrastructure/config"
)

func main() {

	config.SetupEnv()

	config.App().Logger.Info("Starting Akoflow Server")

	go healthcheck.New().StartHealthCheck()
	go worker.New().StartWorker()
	go orchestrator.StartOrchestrator()
	go monitor.StartMonitor()
	go garbagecollector.StartGarbageCollector()

	httpserver.StartServer()

}
