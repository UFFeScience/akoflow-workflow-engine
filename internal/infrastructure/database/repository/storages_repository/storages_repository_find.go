package storages_repository

import (
	"database/sql"

	"github.com/UFFeScience/akoflow/internal/application/ports"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository"
)

const storageColumns = "id, workflow_id, activity_id, pvc_name, namespace, status, storage_mount_path, storage_class, storage_size, initial_file_list, end_file_list, initial_disk_spec, end_disk_spec, keep_storage_after_finish, detached, created_at"

func scanStorage(scanner interface{ Scan(...any) error }) (ports.Storage, error) {
	var result ports.Storage
	err := scanner.Scan(&result.ID, &result.WorkflowID, &result.ActivityID, &result.PVCName, &result.Namespace, &result.Status, &result.StorageMountPath, &result.StorageClass, &result.StorageSize, &result.InitialFileList, &result.EndFileList, &result.InitialDiskSpec, &result.EndDiskSpec, &result.KeepStorageAfterFinish, &result.Detached, &result.CreatedAt)
	return result, err
}

func (s *StorageRepository) Find(id int) (ports.Storage, error) {
	database := repository.Database{}
	c := database.Connect()

	defer c.Close()
	result, err := scanStorage(c.QueryRow("SELECT "+storageColumns+" FROM "+s.tableName+" WHERE id = ?", id))
	if err == sql.ErrNoRows {
		return ports.Storage{}, err
	}
	return result, err
}

func (s *StorageRepository) FindByWorkflow(workflowID int) ([]ports.Storage, error) {
	database := repository.Database{}
	c := database.Connect()

	defer c.Close()
	rows, err := c.Query("SELECT "+storageColumns+" FROM "+s.tableName+" WHERE workflow_id = ?", workflowID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	storages := []ports.Storage{}
	for rows.Next() {
		result, err := scanStorage(rows)
		if err != nil {
			return nil, err
		}
		storages = append(storages, result)
	}
	return storages, rows.Err()
}

func (s *StorageRepository) GetCreatedStorages(namespace string) ([]ports.Storage, error) {
	database := repository.Database{}
	c := database.Connect()

	defer c.Close()
	rows, err := c.Query("SELECT "+storageColumns+" FROM "+s.tableName+" WHERE namespace = ? AND status = ?", namespace, StatusCreated)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	storages := []ports.Storage{}

	for rows.Next() {
		result, err := scanStorage(rows)
		if err != nil {
			return nil, err
		}

		storages = append(storages, result)
	}

	return storages, rows.Err()
}
