package run_schedule_service

import (
	"errors"
	"plugin"
	"strings"
	"testing"

	schedule_entity "github.com/UFFeScience/akoflow/internal/domain/planning/schedule"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/schedule_repository"
)

type repositoryFake struct {
	schedule_repository.IScheduleRepository
	schedule schedule_entity.ScheduleEntity
	err      error
	name     string
}

func (f *repositoryFake) GetScheduleByName(name string) (schedule_entity.ScheduleEntity, error) {
	f.name = name
	return f.schedule, f.err
}

type lookupFake struct {
	symbol plugin.Symbol
	err    error
}

func (f lookupFake) Lookup(string) (plugin.Symbol, error) { return f.symbol, f.err }
func TestStartRunningScheduleSuccess(t *testing.T) {
	repo := &repositoryFake{schedule: schedule_entity.ScheduleEntity{Name: "prism", PluginSoPath: "plugin.so"}}
	service := NewWithDependencies(repo, func(string) (PluginLookup, error) {
		return lookupFake{symbol: func(input any) float64 { return input.(map[string]any)["score"].(float64) }}, nil
	})
	result, err := service.StartRunningSchedule("prism", map[string]any{"score": 4.5})
	if err != nil || result != 4.5 || repo.name != "prism" {
		t.Fatalf("result=%v name=%q err=%v", result, repo.name, err)
	}
}
func TestStartRunningScheduleErrors(t *testing.T) {
	repo := &repositoryFake{err: errors.New("db")}
	service := NewWithDependencies(repo, nil)
	if _, err := service.StartRunningSchedule("x", nil); err == nil {
		t.Fatal("repo")
	}
	repo = &repositoryFake{schedule: schedule_entity.ScheduleEntity{PluginSoPath: "x"}}
	cases := []PluginOpener{func(string) (PluginLookup, error) { return nil, errors.New("open") }, func(string) (PluginLookup, error) { return lookupFake{err: errors.New("lookup")}, nil }, func(string) (PluginLookup, error) { return lookupFake{symbol: "bad"}, nil }}
	for i, opener := range cases {
		service := NewWithDependencies(repo, opener)
		_, err := service.StartRunningSchedule("x", nil)
		if err == nil {
			t.Fatalf("case %d", i)
		}
		if i == 2 && !strings.Contains(err.Error(), "invalid AkoScore") {
			t.Fatal(err)
		}
	}
}
func TestNewInitializesDependencies(t *testing.T) {
	s := New()
	if s.scheduleRepository == nil || s.pluginOpener == nil {
		t.Fatalf("incomplete: %+v", s)
	}
}
