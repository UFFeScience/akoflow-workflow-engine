package main

import (
	"fmt"
	"os"
	"time"

	applicationexecution "github.com/UFFeScience/akoflow/internal/application/execution"
	"github.com/UFFeScience/akoflow/internal/application/ports"
	"github.com/UFFeScience/akoflow/internal/controlplane/eventloop"
	controlexecution "github.com/UFFeScience/akoflow/internal/controlplane/execution"
	domainevents "github.com/UFFeScience/akoflow/internal/domain/events"
	"github.com/UFFeScience/akoflow/internal/provider/simgrid"
)

func buildEventLoop(
	events ports.QueueStore,
	executions ports.ExecutionStore,
	activities *applicationexecution.Controller,
) (*eventloop.Loop, error) {
	dispatcher := eventloop.NewDispatcher()
	if err := registerExecutionHandlers(dispatcher, executions, activities); err != nil {
		return nil, err
	}
	for _, eventType := range domainEventTypes() {
		if err := dispatcher.Register(eventType, eventloop.DomainEventHandler{}); err != nil {
			return nil, err
		}
	}
	owner, _ := os.Hostname()
	config := eventloop.DefaultConfig(fmt.Sprintf("%s-%d", owner, os.Getpid()))
	config.PollInterval = 500 * time.Millisecond
	return eventloop.New(events, dispatcher, config)
}

func registerExecutionHandlers(
	dispatcher *eventloop.Dispatcher,
	executions ports.ExecutionStore,
	activities *applicationexecution.Controller,
) error {
	if err := dispatcher.Register(eventloop.EventActivityExecutionRequested,
		eventloop.NewActivityExecutionHandler(activities)); err != nil {
		return err
	}
	supervisor, err := controlexecution.New(executions, activities,
		simgrid.NewSimulationExecutor(), controlexecution.Config{PollInterval: time.Second, MaxParallel: 8})
	if err != nil {
		return err
	}
	return dispatcher.Register(eventloop.EventExecutionRunRequested,
		eventloop.NewExecutionRunHandler(supervisor))
}

func domainEventTypes() []string {
	return []string{
		domainevents.ExecutionStarted, domainevents.ExecutionCompleted, domainevents.ExecutionFailed,
		domainevents.ActivityStarted, domainevents.ActivityCompleted, domainevents.ActivityFailed,
	}
}
