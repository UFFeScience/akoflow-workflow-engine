package storages_repository

import (
	"github.com/UFFeScience/akoflow/internal/application/ports"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/schema"
)

type StorageRepository struct {
	tableName string
}

var TableName = "storages"
var Columns = "(id INTEGER PRIMARY KEY AUTOINCREMENT, workflow_id INTEGER, activity_id INTEGER, pvc_name TEXT,  namespace TEXT, status INTEGER, storage_mount_path TEXT, storage_class TEXT, storage_size TEXT, initial_file_list TEXT, end_file_list TEXT, initial_disk_spec TEXT, end_disk_spec TEXT, keep_storage_after_finish INTEGER, detached DATETIME,  created_at DATETIME DEFAULT CURRENT_TIMESTAMP)"

type StorageDatabase = ports.Storage

const StatusNotCreated = ports.StorageStatusNotCreated
const StatusCreated = ports.StorageStatusCreated
const StatusCompleted = ports.StorageStatusCompleted

func New() ports.StorageRepository {

	database := repository.Database{}
	c := database.Connect()
	defer c.Close()
	if err := schema.Apply(c); err != nil {
		return nil
	}

	return &StorageRepository{
		tableName: TableName,
	}
}

type IStorageRepository = ports.StorageRepository
