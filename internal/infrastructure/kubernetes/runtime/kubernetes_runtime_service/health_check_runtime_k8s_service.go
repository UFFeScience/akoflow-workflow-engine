package kubernetes_runtime_service

import (
	"fmt"
	"time"

	"github.com/UFFeScience/akoflow/internal/domain"
	"github.com/UFFeScience/akoflow/internal/infrastructure/config"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/resource_repository"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/runtime_repository"
	"github.com/UFFeScience/akoflow/internal/infrastructure/kubernetes/connector"
)

type HealthCheckRuntimeK8sService struct {
	k8sConnector      connector_k8s.IConnector
	runtimeRepository runtime_repository.IRuntimeRepository
	resources         resource_repository.IRepository
}

func NewHealthCheckRuntimeK8sService() *HealthCheckRuntimeK8sService {
	return &HealthCheckRuntimeK8sService{
		k8sConnector:      config.App().Connector.K8sConnector,
		runtimeRepository: config.App().Repository.RuntimeRepository,
		resources:         config.App().Repository.ResourceRepository,
	}
}

func (h *HealthCheckRuntimeK8sService) HealthCheck(runtimeID string) bool {
	runtimeEntity, err := h.runtimeRepository.GetByName(runtimeID)
	if err != nil {
		config.App().Logger.Infof("WORKER: Runtime not found %s", runtimeID)
		return false
	}
	response := h.k8sConnector.Healthz(runtimeEntity).Healthz()
	if !response.Success {
		h.runtimeRepository.UpdateStatus(runtimeEntity, runtime_repository.STATUS_NOT_READY)
		return false
	}
	h.runtimeRepository.UpdateStatus(runtimeEntity, runtime_repository.STATUS_READY)
	environmentVersionID := runtimeEntity.GetCurrentRuntimeMetadata("ENVIRONMENT_VERSION_ID")
	if environmentVersionID == "" {
		config.App().Logger.Error(fmt.Sprintf("WORKER: runtime %s has no ENVIRONMENT_VERSION_ID", runtimeID))
		return false
	}
	metrics, err := h.k8sConnector.Metrics(runtimeEntity).GetNodeMetrics()
	if err != nil {
		return false
	}
	for _, providerMetric := range metrics {
		resource, err := h.resources.FindByProviderID(environmentVersionID, providerMetric.Name)
		if err != nil || resource == nil {
			continue
		}
		snapshot := domain.ResourceSnapshot{
			ID: fmt.Sprintf("%s:%d", resource.ID, time.Now().UnixNano()), ResourceID: resource.ID,
			CapturedAt: time.Now().UTC(), Available: true, CPUUsed: providerMetric.GetCpuUsage(),
			MemoryUsedBytes: int64(providerMetric.GetMemoryUsage()),
		}
		if err := h.resources.CreateSnapshot(snapshot); err != nil {
			return false
		}
	}
	return true
}

func (h *HealthCheckRuntimeK8sService) DiscoverResources(runtimeID string) bool {
	runtimeEntity, err := h.runtimeRepository.GetByName(runtimeID)
	if err != nil {
		return false
	}
	environmentVersionID := runtimeEntity.GetCurrentRuntimeMetadata("ENVIRONMENT_VERSION_ID")
	if environmentVersionID == "" {
		config.App().Logger.Error(fmt.Sprintf("WORKER: runtime %s has no ENVIRONMENT_VERSION_ID", runtimeID))
		return false
	}
	response := h.k8sConnector.Nodes(runtimeEntity).ListNodes()
	if !response.Success {
		return false
	}
	for _, providerMachine := range response.Data {
		resourceID := fmt.Sprintf("%s:%s:%s", environmentVersionID, runtimeID, providerMachine.Name)
		resource := domain.Resource{
			ID: resourceID, EnvironmentVersionID: environmentVersionID, RuntimeID: runtimeID,
			Type: domain.ResourceKubernetesMachine, Name: providerMachine.Name,
			ProviderID: providerMachine.Name, CPUCapacity: providerMachine.GetCpuMax(),
			MemoryBytes: int64(providerMachine.GetNodeMemoryMax()), StorageBytes: int64(providerMachine.GetNodeNetworkMax()),
			ComputeSpeedup: 1, Schedulable: true,
		}
		if err := h.resources.Upsert(resource); err != nil {
			config.App().Logger.Error(fmt.Sprintf("WORKER: failed to upsert resource %s: %v", resourceID, err))
			return false
		}
	}
	return true
}
