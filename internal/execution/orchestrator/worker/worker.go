package worker

import (
	"github.com/UFFeScience/akoflow/internal/application/services/run_activity_in_cluster_service"
	"github.com/UFFeScience/akoflow/internal/execution/lifecycle/channel"
	"github.com/UFFeScience/akoflow/internal/infrastructure/config"
)

var FLAG_ID_WORKER_STOP_LISTENING = -1

type Worker struct {
	State WorkerState
}

func New() *Worker {
	return &Worker{
		State: WorkerState{},
	}
}

func (w *Worker) StartWorker() {
	for {

		managerChannel := channel.GetInstance()
		result := <-managerChannel.WorfklowChannel

		if result.Id == FLAG_ID_WORKER_STOP_LISTENING {
			break
		}

		runActivityInClusterService := run_activity_in_cluster_service.New()
		runActivityInClusterService.Run(result.Id)

		config.App().Logger.Info("Worker: Activity finished", result.Id)
	}
}
