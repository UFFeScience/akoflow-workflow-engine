package monitor

import (
	"time"

	"github.com/UFFeScience/akoflow/internal/application/services/monitor_change_workflow_service"
	"github.com/UFFeScience/akoflow/internal/application/services/monitor_collect_metrics_service"
	"github.com/UFFeScience/akoflow/internal/infrastructure/config"
)

const TimeToUpdateSeconds = 1

func StartMonitor() {
	for {
		handleMonitor()
		time.Sleep(TimeToUpdateSeconds * time.Second)
		config.App().Logger.Info("Monitor is running")

	}
}

func handleMonitor() {
	monitorChangeWorkflowService := monitor_change_workflow_service.New()
	monitorChangeWorkflowService.MonitorChangeWorkflow()

	monitorCollectMetricsService := monitor_collect_metrics_service.New()
	monitorCollectMetricsService.CollectMetrics()
}
