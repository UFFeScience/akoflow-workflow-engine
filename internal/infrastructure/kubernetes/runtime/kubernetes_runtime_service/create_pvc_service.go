package kubernetes_runtime_service

import (
	"github.com/UFFeScience/akoflow/internal/application/ports"
	"github.com/UFFeScience/akoflow/internal/infrastructure/config"
	"github.com/UFFeScience/akoflow/internal/infrastructure/kubernetes/connector"
	"github.com/UFFeScience/akoflow/internal/infrastructure/kubernetes/connector/connector_pvc_k8s"

	"github.com/UFFeScience/akoflow/internal/domain/workflow/activity"
	"github.com/UFFeScience/akoflow/internal/domain/workflow/definition"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/runtime_repository"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/storages_repository"
)

type CreatePVCService struct {
	connector         connector_k8s.IConnector
	storageRepository ports.StorageRepository

	runtimeRepository runtime_repository.IRuntimeRepository
}

func NewCreatePVCService() CreatePVCService {
	return CreatePVCService{
		connector:         config.App().Connector.K8sConnector,
		storageRepository: config.App().Repository.StoragesRepository,

		runtimeRepository: config.App().Repository.RuntimeRepository,
	}
}

func (c *CreatePVCService) GetOrCreatePersistentVolumeClainByActivity(wf workflow_entity.Workflow, wfa workflow_activity_entity.WorkflowActivities, namespace string) (string, error) {

	var err error
	var pvc connector_pvc_k8s.ResponseGetPersistentVolumeClain

	runtime, err := c.runtimeRepository.GetByName(wfa.GetRuntimeId())

	if err != nil {
		println("Runtime not found")
		return "", err
	}

	if wf.IsStoragePolicyStandalone() {
		pvc, err = c.connector.PersistentVolumeClain(runtime).GetPersistentVolumeClain(wfa.GetVolumeName(), namespace)
	} else {
		pvc, err = c.connector.PersistentVolumeClain(runtime).GetPersistentVolumeClain(wf.MakeVolumeNameDistributed(), namespace)
	}

	if err != nil {
		println("Persistent volume not found")
		return c.handleCreatePersistentVolumeClain(wf, wfa, namespace)
	}

	err = c.storageRepository.Update(storages_repository.ParamsStorageUpdate{
		PVCName:    wfa.GetVolumeName(),
		Status:     storages_repository.StatusCreated,
		ActivityID: wfa.Id,
	})

	return pvc.Metadata.Name, nil
}

func (c *CreatePVCService) handleCreatePersistentVolumeClain(wf workflow_entity.Workflow, wfa workflow_activity_entity.WorkflowActivities, namespace string) (string, error) {

	var pv connector_pvc_k8s.ResponseCreatePersistentVolumeClain
	var err error

	runtime, err := c.runtimeRepository.GetByName(wfa.GetRuntimeId())
	if err != nil {
		println("Runtime not found")
		return "", err
	}

	if wf.IsStoragePolicyStandalone() {
		pv, err = c.connector.
			PersistentVolumeClain(runtime).
			CreatePersistentVolumeClain(
				wfa.GetVolumeName(),
				namespace,
				wf.Spec.StoragePolicy.StorageSize,
				wf.Spec.StoragePolicy.StorageClassName,
			)
	} else {
		pv, err = c.connector.
			PersistentVolumeClain(runtime).
			CreatePersistentVolumeClain(
				wf.MakeVolumeNameDistributed(),
				namespace,
				wf.Spec.StoragePolicy.StorageSize,
				wf.MakeStorageClassNameDistributed(),
			)
	}

	if err != nil {
		println("Error creating persistent volume")
		return "", err
	}

	if pv.Metadata.Name == "" {
		println("Error creating persistent volume")
		return "", err
	}

	err = c.storageRepository.Update(storages_repository.ParamsStorageUpdate{
		PVCName:    wfa.GetVolumeName(),
		Status:     storages_repository.StatusCreated,
		ActivityID: wfa.Id,
	})

	if err != nil {
		return "", err
	}

	return pv.Metadata.Name, nil
}
