package kubernetes_runtime_service

import (
	"fmt"
	"time"

	"github.com/UFFeScience/akoflow/internal/application/ports"
	"github.com/UFFeScience/akoflow/internal/domain"
	"github.com/UFFeScience/akoflow/internal/domain/workflow/activity"
	"github.com/UFFeScience/akoflow/internal/domain/workflow/definition"
	"github.com/UFFeScience/akoflow/internal/infrastructure/config"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/resource_repository"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/runtime_repository"
	"github.com/UFFeScience/akoflow/internal/infrastructure/kubernetes/connector"
)

type MonitorGetMetricsActivityService struct {
	namespace          string
	logsRepository     ports.LogsRepository
	metricsRepository  ports.MetricsRepository
	resourceRepository resource_repository.IRepository
	workflowRepository ports.WorkflowRepository
	activityRepository ports.ActivityRepository
	runtimeRepository  runtime_repository.IRuntimeRepository

	connector connector_k8s.IConnector
}

func NewMonitorGetMetricsActivityService() *MonitorGetMetricsActivityService {
	return &MonitorGetMetricsActivityService{
		namespace:          "akoflow",
		logsRepository:     config.App().Repository.LogsRepository,
		metricsRepository:  config.App().Repository.MetricsRepository,
		workflowRepository: config.App().Repository.WorkflowRepository,
		activityRepository: config.App().Repository.ActivityRepository,
		resourceRepository: config.App().Repository.ResourceRepository,

		runtimeRepository: config.App().Repository.RuntimeRepository,

		connector: config.App().Connector.K8sConnector,
	}
}

func (m *MonitorGetMetricsActivityService) GetMetrics(wf int, wfa int) {
	m.handleGetMetricsByActivity(wf, wfa)
}

func (m *MonitorGetMetricsActivityService) handleGetMetricsByActivity(wfID int, wfaID int) {

	wfa, err := m.activityRepository.Find(wfaID)
	wf, _ := m.workflowRepository.Find(wfID)
	if err != nil {
		config.App().Logger.Infof("WORKER: Activity not found %d", wfa.Id)
		return
	}

	fmt.Println("Activity: ", wfa.WorkflowId, wfa.Id)

	nameJob := wfa.GetNameJob()

	runtime, err := m.runtimeRepository.GetByName(wfa.GetRuntimeId())
	if err != nil {
		return
	}

	job, err := m.connector.Pod(runtime).GetPodByJob(m.namespace, nameJob)
	if err != nil {
		return
	}

	podName, err := job.GetPodName()
	if err != nil {
		return
	}

	m.retrieveMetricsInDatabase(wf, wfa, podName)
}

func (m *MonitorGetMetricsActivityService) retrieveMetricsInDatabase(_ workflow_entity.Workflow, wfa workflow_activity_entity.WorkflowActivities, podName string) {

	runtime, err := m.runtimeRepository.GetByName(wfa.GetRuntimeId())
	if err != nil {
		return
	}

	metric, err := m.connector.Metrics(runtime).GetPodMetrics(m.namespace, podName)
	if err != nil {
		return
	}

	_ = m.metricsRepository.Create(ports.ActivityMetric{
		ActivityID: wfa.Id,
		CPU:        metric.Containers[0].Usage.Cpu,
		Memory:     metric.Containers[0].Usage.Memory,
		Window:     metric.Window,
		Timestamp:  metric.Timestamp.String(),
	})

	config.App().Logger.Infof("WORKER: Metrics collected for activity %d - CPU: %s - Memory: %s", wfa.Id, metric.Containers[0].Usage.Cpu, metric.Containers[0].Usage.Memory)

	resourceMetrics, err := m.connector.Metrics(runtime).GetNodeMetrics()
	if err != nil {
		return
	}

	environmentVersionID := runtime.GetCurrentRuntimeMetadata("ENVIRONMENT_VERSION_ID")
	for _, providerMetric := range resourceMetrics {

		resource, err := m.resourceRepository.FindByProviderID(environmentVersionID, providerMetric.Name)
		if err != nil || resource == nil {
			continue
		}
		_ = m.resourceRepository.CreateSnapshot(domain.ResourceSnapshot{
			ID:         fmt.Sprintf("%s:%d", resource.ID, time.Now().UnixNano()),
			ResourceID: resource.ID, CapturedAt: metric.Timestamp, Available: true,
			CPUUsed: providerMetric.GetCpuUsage(), MemoryUsedBytes: int64(providerMetric.GetMemoryUsage()),
		})
	}

}
