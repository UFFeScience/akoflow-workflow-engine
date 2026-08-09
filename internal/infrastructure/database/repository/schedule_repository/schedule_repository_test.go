package schedule_repository

import (
	"path/filepath"
	"testing"
)

func setup(t *testing.T) IScheduleRepository {
	t.Helper()
	t.Setenv("AKOFLOW_DATABASE_PATH", filepath.Join(t.TempDir(), "db.sqlite"))
	return New()
}

func TestScheduleRepositoryLifecycle(t *testing.T) {
	r := setup(t)
	created, err := r.CreateSchedule("prism", "beam", "code", "/plugin.so")
	if err != nil || created.ID == 0 {
		t.Fatalf("create failed: %+v %v", created, err)
	}
	found, err := r.GetScheduleByName("prism")
	if err != nil || found.Name != "prism" || found.PluginSoPath != "/plugin.so" {
		t.Fatalf("find failed: %+v %v", found, err)
	}
	list, err := r.ListAllSchedules()
	if err != nil || len(list) != 1 || list[0].Name != "prism" {
		t.Fatalf("list failed: %+v %v", list, err)
	}
	if _, err := r.GetScheduleByName("missing"); err == nil {
		t.Fatal("missing schedule must fail")
	}
}
