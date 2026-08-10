package storages_repository

import (
	"fmt"
	"time"

	"github.com/UFFeScience/akoflow/internal/application/ports"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository"
)

type ParamsStorageUpdate = ports.UpdateStorageParams

func (s *StorageRepository) Update(params ports.UpdateStorageParams) error {

	database := repository.Database{}
	c := database.Connect()

	defer c.Close()
	if params.Status <= 0 || params.PVCName == "" || params.ActivityID <= 0 {
		return fmt.Errorf("invalid storage update")
	}

	_, err := c.Exec("UPDATE "+s.tableName+" SET status = ?, pvc_name = ? WHERE activity_id = ?", params.Status, params.PVCName, params.ActivityID)
	return err

}

func (s *StorageRepository) UpdateInitialFileListDisk(activityId int, fileDisk string) error {

	database := repository.Database{}
	c := database.Connect()

	defer c.Close()
	_, err := c.Exec("UPDATE "+s.tableName+" SET initial_file_list = ? WHERE activity_id = ?", fileDisk, activityId)

	if err != nil {
		return err
	}

	return nil
}

func (s *StorageRepository) UpdateEndFileListDisk(activityId int, fileDisk string) error {

	database := repository.Database{}
	c := database.Connect()

	defer c.Close()
	_, err := c.Exec("UPDATE "+s.tableName+" SET end_file_list = ? WHERE activity_id = ?", fileDisk, activityId)

	if err != nil {
		return err
	}

	return nil
}

func (s *StorageRepository) UpdateInitialDiskSpec(activityId int, fileSpec string) error {

	database := repository.Database{}
	c := database.Connect()

	defer c.Close()
	_, err := c.Exec("UPDATE "+s.tableName+" SET initial_disk_spec = ? WHERE activity_id = ?", fileSpec, activityId)

	if err != nil {
		return err
	}

	return nil
}

func (s *StorageRepository) UpdateEndDiskSpec(activityId int, fileSpec string) error {

	database := repository.Database{}
	c := database.Connect()

	defer c.Close()
	_, err := c.Exec("UPDATE "+s.tableName+" SET end_disk_spec = ? WHERE activity_id = ?", fileSpec, activityId)
	if err != nil {
		return err
	}

	return nil
}

func (s *StorageRepository) UpdateDetached(activityId int) error {

	database := repository.Database{}
	c := database.Connect()

	now := time.Now()

	defer c.Close()
	_, err := c.Exec("UPDATE "+s.tableName+" SET detached = ? WHERE activity_id = ?", now.Format("2006-01-02 15:04:05"), activityId)
	if err != nil {
		return err
	}

	return nil
}
