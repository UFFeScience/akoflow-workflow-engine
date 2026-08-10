package storages_repository

import (
	"github.com/UFFeScience/akoflow/internal/application/ports"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository"
)

type ParamsStorageCreate = ports.CreateStorageParams

func (s *StorageRepository) Create(params ports.CreateStorageParams) error {

	database := repository.Database{}
	c := database.Connect()

	defer c.Close()
	for activityId, keepDisk := range params.ActivitiesKeepingDisk {
		_, err := c.Exec(
			"INSERT INTO "+s.tableName+" (workflow_id, activity_id, namespace, status, storage_mount_path, storage_class, storage_size, initial_file_list, end_file_list, initial_disk_spec, end_disk_spec, keep_storage_after_finish) VALUES (?, ? , ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			params.WorkflowID,
			activityId,
			params.Namespace,
			params.Status,
			params.StorageMountPath,
			params.StorageClass,
			params.StorageSize,
			"{}", // initial_file_list
			"{}", // end_file_list
			"{}", // initial_disk_spec
			"{}", // end_disk_spec
			keepDisk,
		)

		if err != nil {
			return err
		}
	}
	return nil
}
