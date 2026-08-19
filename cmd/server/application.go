package main

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/UFFeScience/akoflow/internal/api/handlers/workflow_engine_api_handler"
	"github.com/UFFeScience/akoflow/internal/api/httpserver"
	applicationexecution "github.com/UFFeScience/akoflow/internal/application/execution"
	"github.com/UFFeScience/akoflow/internal/controlplane/eventloop"
	"github.com/UFFeScience/akoflow/internal/infrastructure/config"
	"github.com/UFFeScience/akoflow/internal/infrastructure/config/logger"
	"github.com/UFFeScience/akoflow/internal/provider/kubernetes"
)

type application struct {
	settings       config.Settings
	log            *logger.Logger
	database       *sql.DB
	api            *workflow_engine_api_handler.Handler
	eventLoop      *eventloop.Loop
	historyCleaner *kubernetes.HistoryCleaner
}

func newApplication(ctx context.Context, settings config.Settings, log *logger.Logger) (*application, error) {
	storage, err := openPersistence(ctx)
	if err != nil {
		return nil, err
	}
	fail := func(err error) (*application, error) {
		_ = storage.database.Close()
		return nil, err
	}
	kubernetesAPI, err := connectKubernetes(settings)
	if err != nil {
		return fail(err)
	}
	runtimes, err := buildRuntimes(settings, kubernetesAPI)
	if err != nil {
		return fail(err)
	}
	activities := applicationexecution.New(runtimes, storage.executions, storage.data)
	simulator, err := buildSimulator(settings)
	if err != nil {
		return fail(err)
	}
	loop, err := buildEventLoop(storage.events, storage.executions, activities, simulator)
	if err != nil {
		return fail(err)
	}
	api, err := buildAPI(storage)
	if err != nil {
		return fail(err)
	}
	cleaner, err := buildHistoryCleaner(settings, kubernetesAPI)
	if err != nil {
		return fail(err)
	}
	return &application{
		settings: settings, log: log, database: storage.database,
		api: api, eventLoop: loop, historyCleaner: cleaner,
	}, nil
}

func (a *application) Run(ctx context.Context) error {
	a.log.Info("Starting Akoflow Server")
	a.startEventLoop(ctx)
	a.startHistoryCleanup(ctx)
	if err := httpserver.Serve(ctx, a.settings.HTTPAddress, a.api); err != nil {
		return fmt.Errorf("serve Akoflow API: %w", err)
	}
	return nil
}

func (a *application) Close() error {
	return a.database.Close()
}
