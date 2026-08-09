package runtime_repository

import (
	"path/filepath"
	"testing"
)

func setup(t *testing.T) IRuntimeRepository {
	t.Helper()
	t.Setenv("AKOFLOW_DATABASE_PATH", filepath.Join(t.TempDir(), "db.sqlite"))
	return New()
}

func TestRuntimeRepositoryLifecycle(t *testing.T) {
	r := setup(t)
	r.CreateOrUpdate("k8s", STATUS_READY, map[string]string{"REGION": "us"})
	got, err := r.GetByName("k8s")
	if err != nil || got == nil || got.Status != STATUS_READY || got.Metadata["REGION"] != "us" {
		t.Fatalf("unexpected runtime: %+v %v", got, err)
	}
	all, err := r.GetAll()
	if err != nil || len(all) != 1 {
		t.Fatalf("unexpected list: %v %v", all, err)
	}
	r.CreateOrUpdate("k8s", STATUS_NOT_READY, nil)
	got, _ = r.GetByName("k8s")
	if got.Status != STATUS_NOT_READY {
		t.Fatal("update failed")
	}
	if err := r.UpdateStatus(got, STATUS_READY); err != nil {
		t.Fatal(err)
	}
	got, _ = r.GetByName("k8s")
	if got.Status != STATUS_READY {
		t.Fatal("status update failed")
	}
	if _, err := r.GetByName("missing"); err == nil {
		t.Fatal("missing runtime must fail")
	}
}
