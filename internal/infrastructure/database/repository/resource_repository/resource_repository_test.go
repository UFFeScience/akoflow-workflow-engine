package resource_repository

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/UFFeScience/akoflow/internal/domain"
)

func testRepository(t *testing.T) IRepository {
	t.Helper()
	t.Setenv("AKOFLOW_DATABASE_PATH", filepath.Join(t.TempDir(), "db.sqlite"))
	return New()
}

func TestResourceRepositoryLifecycle(t *testing.T) {
	repository := testRepository(t)
	parent := "parent"
	resource := domain.Resource{ID: "r1", EnvironmentVersionID: "env", RuntimeID: "k8s", ParentResourceID: &parent, Type: domain.ResourceCloudVM, Name: "one", ProviderID: "provider", CPUCores: 4, CPUCapacity: 4.5, MemoryBytes: 1000, StorageBytes: 2000, ComputeSpeedup: 2, PricePerSecond: .1, Schedulable: true, Metadata: map[string]any{"zone": "a"}}
	if err := repository.Upsert(resource); err != nil {
		t.Fatal(err)
	}
	found, err := repository.FindByID("r1")
	if err != nil || found == nil || found.Name != "one" || found.ParentResourceID == nil || found.Metadata["zone"] != "a" {
		t.Fatalf("unexpected resource: %+v %v", found, err)
	}
	byProvider, err := repository.FindByProviderID("env", "provider")
	if err != nil || byProvider == nil {
		t.Fatal("provider lookup failed")
	}
	if list, err := repository.ListByRuntime("env", "k8s"); err != nil || len(list) != 1 {
		t.Fatalf("runtime list: %v %v", list, err)
	}
	if list, err := repository.ListSchedulable("env"); err != nil || len(list) != 1 {
		t.Fatalf("schedulable list: %v %v", list, err)
	}
	resource.Name = "updated"
	resource.Schedulable = false
	if err := repository.Upsert(resource); err != nil {
		t.Fatal(err)
	}
	found, _ = repository.FindByID("r1")
	if found.Name != "updated" || found.Schedulable {
		t.Fatal("upsert update failed")
	}
	if missing, err := repository.FindByID("missing"); err != nil || missing != nil {
		t.Fatal("missing resource must be nil")
	}
	if missing, err := repository.FindByProviderID("env", "missing"); err != nil || missing != nil {
		t.Fatal("missing provider must be nil")
	}
}

func TestResourceSnapshots(t *testing.T) {
	repository := testRepository(t)
	if snapshot, err := repository.LatestSnapshot("missing"); err != nil || snapshot != nil {
		t.Fatal("missing snapshot must be nil")
	}
	when := time.Now().UTC().Truncate(time.Second)
	snapshot := domain.ResourceSnapshot{ID: "s1", ResourceID: "r1", CapturedAt: when, Available: true, CPUUsed: 2, MemoryUsedBytes: 3, NetworkInBPS: 4, NetworkOutBPS: 5, DiskReadBPS: 6, DiskWriteBPS: 7, QueueLength: 8, Metadata: map[string]any{"source": "test"}}
	if err := repository.CreateSnapshot(snapshot); err != nil {
		t.Fatal(err)
	}
	got, err := repository.LatestSnapshot("r1")
	if err != nil || got == nil || got.ID != "s1" || got.Metadata["source"] != "test" {
		t.Fatalf("unexpected snapshot: %+v %v", got, err)
	}
}

func TestResourceRepositoryRejectsUnserializableMetadata(t *testing.T) {
	repository := testRepository(t)
	if err := repository.Upsert(domain.Resource{Metadata: map[string]any{"bad": make(chan int)}}); err == nil {
		t.Fatal("resource metadata must fail")
	}
	if err := repository.CreateSnapshot(domain.ResourceSnapshot{Metadata: map[string]any{"bad": make(chan int)}}); err == nil {
		t.Fatal("snapshot metadata must fail")
	}
}
