package worker

import (
	"github.com/UFFeScience/akoflow/internal/application/services/run_activity_in_cluster_service"
	"github.com/UFFeScience/akoflow/internal/execution/lifecycle/channel"
	"github.com/UFFeScience/akoflow/internal/infrastructure/config"
)

var FLAG_ID_WORKER_STOP_LISTENING = -1

type Worker struct {
	State           WorkerState
	activityChannel <-chan channel.DataChannel
	runner          ActivityRunner
}

type ActivityRunner interface {
	Run(activityID int) error
}

func New() *Worker {
	return &Worker{
		State:           WorkerState{},
		activityChannel: channel.GetInstance().WorfklowChannel,
		runner:          run_activity_in_cluster_service.New(),
	}
}

func NewWithDependencies(activityChannel <-chan channel.DataChannel, runner ActivityRunner) *Worker {
	return &Worker{State: WorkerState{}, activityChannel: activityChannel, runner: runner}
}

func (w *Worker) StartWorker() {
	for {

		result, open := <-w.activityChannel
		if !open {
			break
		}

		if result.Id == FLAG_ID_WORKER_STOP_LISTENING {
			break
		}

		if err := w.runner.Run(result.Id); err != nil {
			config.App().Logger.Error("Worker: Activity failed", result.Id, err)
			continue
		}

		config.App().Logger.Info("Worker: Activity dispatched", result.Id)
	}
}
