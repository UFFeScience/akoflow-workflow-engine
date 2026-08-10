package storages_repository

import (
	"testing"

	"github.com/UFFeScience/akoflow/internal/application/ports"
	base "github.com/UFFeScience/akoflow/internal/infrastructure/database/repository"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/schema"
)

func storageTestRepository(t *testing.T) ports.StorageRepository {
	t.Helper()
	t.Setenv("AKOFLOW_DATABASE_PATH", t.TempDir()+"/akoflow.db")
	db := (&base.Database{}).Connect()
	if err := schema.Apply(db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO workflows(id, namespace, runtime, name, raw_workflow, status) VALUES(1,'lab','local','wf','',0)"); err != nil {
		t.Fatal(err)
	}
	for _, id := range []int{10, 20} {
		if _, err := db.Exec("INSERT INTO activities(id, workflow_id, namespace, name, image, runtime, resource_k8s_base64, status) VALUES(?,1,'lab',?,'','local','',0)", id, id); err != nil {
			t.Fatal(err)
		}
	}
	db.Close()
	return New()
}

func TestStorageRepositoryLifecycle(t *testing.T) {
	repository := storageTestRepository(t)
	err := repository.Create(ports.CreateStorageParams{WorkflowID: 1, Namespace: "lab", Status: ports.StorageStatusCreated, StorageMountPath: "/data", StorageClass: "fast", StorageSize: "10Gi", ActivitiesKeepingDisk: map[int]bool{10: true, 20: false}})
	if err != nil {
		t.Fatal(err)
	}
	storages, err := repository.FindByWorkflow(1)
	if err != nil || len(storages) != 2 {
		t.Fatalf("storages=%v err=%v", storages, err)
	}
	first, err := repository.Find(storages[0].ID)
	if err != nil || first.ActivityID == 0 || first.StorageMountPath != "/data" {
		t.Fatalf("first=%+v err=%v", first, err)
	}
	if err = repository.Update(ports.UpdateStorageParams{ActivityID: 10, Status: ports.StorageStatusCompleted, PVCName: "pvc-10"}); err != nil {
		t.Fatal(err)
	}
	if err = repository.UpdateInitialFileListDisk(10, `{"before":true}`); err != nil {
		t.Fatal(err)
	}
	if err = repository.UpdateEndFileListDisk(10, `{"after":true}`); err != nil {
		t.Fatal(err)
	}
	if err = repository.UpdateInitialDiskSpec(10, "initial"); err != nil {
		t.Fatal(err)
	}
	if err = repository.UpdateEndDiskSpec(10, "end"); err != nil {
		t.Fatal(err)
	}
	if err = repository.UpdateDetached(10); err != nil {
		t.Fatal(err)
	}
	updated, _ := repository.Find(first.ID)
	if updated.PVCName == nil || *updated.PVCName != "pvc-10" || updated.Detached == nil || updated.InitialDiskSpec != "initial" || updated.EndDiskSpec != "end" {
		t.Fatalf("updated=%+v", updated)
	}
	created, err := repository.GetCreatedStorages("lab")
	if err != nil || len(created) != 1 || created[0].ActivityID != 20 {
		t.Fatalf("created=%v err=%v", created, err)
	}
}

func TestStorageRepositoryRejectsInvalidUpdateAndMissingRecord(t *testing.T) {
	repository := storageTestRepository(t)
	if err := repository.Update(ports.UpdateStorageParams{}); err == nil {
		t.Fatal("expected validation error")
	}
	if _, err := repository.Find(999); err == nil {
		t.Fatal("expected not found error")
	}
}
