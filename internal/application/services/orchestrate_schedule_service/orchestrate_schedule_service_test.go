package orchestrate_schedule_service

import (
	"errors"
	"plugin"
	"strings"
	"testing"

	"github.com/UFFeScience/akoflow/internal/application/services/resource_current_metrics_service"
	"github.com/UFFeScience/akoflow/internal/domain"
	schedule_entity "github.com/UFFeScience/akoflow/internal/domain/planning/schedule"
	workflow_activity_entity "github.com/UFFeScience/akoflow/internal/domain/workflow/activity"
	workflow_entity "github.com/UFFeScience/akoflow/internal/domain/workflow/definition"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/activity_repository"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/resource_repository"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/schedule_repository"
)

type scheduleRepositoryFake struct {
	schedule_repository.IScheduleRepository
	schedule schedule_entity.ScheduleEntity
	err      error
}

func (f scheduleRepositoryFake) GetScheduleByName(string) (schedule_entity.ScheduleEntity, error) {
	return f.schedule, f.err
}

type scheduleActivityRepositoryFake struct {
	activity_repository.IActivityRepository
	scheduled   map[int]bool
	isErr       error
	persistErr  error
	resourceIDs map[int]string
}

func (f *scheduleActivityRepositoryFake) IsActivityScheduled(_ int, activityID int) (bool, error) {
	return f.scheduled[activityID], f.isErr
}

func (f *scheduleActivityRepositoryFake) SetActivitySchedule(_ int, activityID int, resourceID, _ string, _, _ float64, _ string) error {
	if f.persistErr != nil {
		return f.persistErr
	}
	f.resourceIDs[activityID] = resourceID
	return nil
}

type scheduleResourceRepositoryFake struct {
	resource_repository.IRepository
	resources []domain.Resource
	err       error
}

func (f scheduleResourceRepositoryFake) ListByRuntime(_, _ string) ([]domain.Resource, error) {
	return f.resources, f.err
}

type metricsFake struct {
	values map[string]*resource_current_metrics_service.Metrics
	err    error
}

func (f metricsFake) Get(resourceID string, _ []workflow_activity_entity.WorkflowActivities) (*resource_current_metrics_service.Metrics, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.values[resourceID], nil
}

func schedulingFixture(score ScoreRunner) (*OrchestrateScheduleService, *scheduleActivityRepositoryFake) {
	activities := &scheduleActivityRepositoryFake{scheduled: map[int]bool{}, resourceIDs: map[int]string{}}
	service := NewWithDependencies(
		scheduleRepositoryFake{schedule: schedule_entity.ScheduleEntity{Name: "prism"}},
		activities,
		scheduleResourceRepositoryFake{resources: []domain.Resource{{ID: "slow", Name: "slow"}, {ID: "fast", Name: "fast"}}},
		metricsFake{values: map[string]*resource_current_metrics_service.Metrics{
			"slow": {CPUTotal: 4, MemoryTotal: 4096},
			"fast": {CPUTotal: 8, MemoryTotal: 8192},
		}},
		score,
	)
	workflow := workflow_entity.Workflow{Id: 7, Spec: workflow_entity.WorkflowSpec{Schedule: "prism", Runtime: "k8s", EnvironmentVersionID: "env-1"}}
	service.SetWorkflow(workflow).SetReadyToRunActivities([]workflow_activity_entity.WorkflowActivities{{Id: 11, WorkflowId: 7, Name: "task", CpuLimit: "2", MemoryLimit: "512Mi"}})
	return &service, activities
}

func TestOrchestrateSelectsHighestScoringResourceAndPersistsAssignment(t *testing.T) {
	service, activities := schedulingFixture(func(_ string, input map[string]any) (float64, error) {
		if input["machine_type"] == "fast" {
			return 9, nil
		}
		return 3, nil
	})

	ready, err := service.Orchestrate()

	if err != nil || len(ready) != 1 || activities.resourceIDs[11] != "fast" {
		t.Fatalf("ready=%v assignments=%v error=%v", ready, activities.resourceIDs, err)
	}
}

func TestOrchestrateSkipsAlreadyScheduledAndZeroScoreActivities(t *testing.T) {
	t.Run("already scheduled", func(t *testing.T) {
		service, activities := schedulingFixture(func(string, map[string]any) (float64, error) { return 5, nil })
		activities.scheduled[11] = true
		ready, err := service.Orchestrate()
		if err != nil || len(ready) != 0 || len(activities.resourceIDs) != 0 {
			t.Fatalf("ready=%v assignments=%v error=%v", ready, activities.resourceIDs, err)
		}
	})

	t.Run("zero score", func(t *testing.T) {
		service, activities := schedulingFixture(func(string, map[string]any) (float64, error) { return 0, nil })
		ready, err := service.Orchestrate()
		if err != nil || len(ready) != 0 || len(activities.resourceIDs) != 0 {
			t.Fatalf("ready=%v assignments=%v error=%v", ready, activities.resourceIDs, err)
		}
	})

	t.Run("no resources", func(t *testing.T) {
		service, activities := schedulingFixture(func(string, map[string]any) (float64, error) { return 5, nil })
		service.resourceRepository = scheduleResourceRepositoryFake{resources: nil}
		ready, err := service.Orchestrate()
		if err != nil || len(ready) != 0 || len(activities.resourceIDs) != 0 {
			t.Fatalf("ready=%v assignments=%v error=%v", ready, activities.resourceIDs, err)
		}
	})
}

func TestOrchestrateFallsBackWhenScheduleDoesNotExist(t *testing.T) {
	activities := []workflow_activity_entity.WorkflowActivities{{Id: 5}}
	service := NewWithDependencies(scheduleRepositoryFake{err: errors.New("not found")}, &scheduleActivityRepositoryFake{}, scheduleResourceRepositoryFake{}, metricsFake{}, nil)
	service.SetReadyToRunActivities(activities)
	result, err := service.Orchestrate()
	if err != nil || len(result) != 1 || result[0].Id != 5 {
		t.Fatalf("result=%v error=%v", result, err)
	}
}

func TestOrchestratePropagatesSchedulingBoundaryErrors(t *testing.T) {
	tests := []struct {
		name      string
		configure func(*OrchestrateScheduleService, *scheduleActivityRepositoryFake)
		want      string
	}{
		{"scheduled lookup", func(_ *OrchestrateScheduleService, repo *scheduleActivityRepositoryFake) {
			repo.isErr = errors.New("db")
		}, "checking if activity"},
		{"resources", func(service *OrchestrateScheduleService, _ *scheduleActivityRepositoryFake) {
			service.resourceRepository = scheduleResourceRepositoryFake{err: errors.New("db")}
		}, "getting nodes"},
		{"metrics", func(service *OrchestrateScheduleService, _ *scheduleActivityRepositoryFake) {
			service.resourceMetricsService = metricsFake{err: errors.New("metrics")}
		}, "getting activity schedule"},
		{"score", func(service *OrchestrateScheduleService, _ *scheduleActivityRepositoryFake) {
			service.scoreRunner = func(string, map[string]any) (float64, error) { return 0, errors.New("plugin") }
		}, "score activity 11"},
		{"persist", func(_ *OrchestrateScheduleService, repo *scheduleActivityRepositoryFake) {
			repo.persistErr = errors.New("write")
		}, "persist schedule"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, activities := schedulingFixture(func(string, map[string]any) (float64, error) { return 5, nil })
			tt.configure(service, activities)
			_, err := service.Orchestrate()
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error=%v want containing %q", err, tt.want)
			}
		})
	}
}

func TestBestNodeHandlesEmptyAndChoosesMaximum(t *testing.T) {
	service := OrchestrateScheduleService{}
	if service.getBestNode(nil) != nil {
		t.Fatal("empty candidates should have no best node")
	}
	best := service.getBestNode([]ResponseStartSchedule{{"ako_score": 2.0}, {"ako_score": 7.0}, {"ako_score": 3.0}})
	if best["ako_score"] != 7.0 {
		t.Fatalf("best=%v", best)
	}
}

func TestStartRunScheduleReturnsRepositoryAndPluginOpenErrors(t *testing.T) {
	t.Run("repository", func(t *testing.T) {
		service := NewWithDependencies(scheduleRepositoryFake{err: errors.New("db")}, nil, nil, nil, nil)
		if _, err := service.StartRunSchedule("prism", nil); err == nil {
			t.Fatal("expected repository error")
		}
	})
	t.Run("plugin open", func(t *testing.T) {
		service := NewWithDependencies(scheduleRepositoryFake{schedule: schedule_entity.ScheduleEntity{Name: "prism", PluginSoPath: "/missing/plugin.so"}}, nil, nil, nil, nil)
		if _, err := service.StartRunSchedule("prism", nil); err == nil {
			t.Fatal("expected plugin open error")
		}
	})
}

type pluginLookupFake struct {
	symbol plugin.Symbol
	err    error
}

func (f pluginLookupFake) Lookup(string) (plugin.Symbol, error) { return f.symbol, f.err }

func TestStartRunScheduleLoadsAndValidatesScoreSymbol(t *testing.T) {
	original := openPlugin
	t.Cleanup(func() { openPlugin = original })
	repository := scheduleRepositoryFake{schedule: schedule_entity.ScheduleEntity{Name: "prism", PluginSoPath: "plugin.so"}}
	service := NewWithDependencies(repository, nil, nil, nil, nil)

	t.Run("lookup error", func(t *testing.T) {
		openPlugin = func(string) (pluginSymbolLookup, error) {
			return pluginLookupFake{err: errors.New("symbol missing")}, nil
		}
		if _, err := service.StartRunSchedule("prism", nil); err == nil {
			t.Fatal("expected lookup error")
		}
	})

	t.Run("invalid symbol", func(t *testing.T) {
		openPlugin = func(string) (pluginSymbolLookup, error) { return pluginLookupFake{symbol: "not-a-function"}, nil }
		if _, err := service.StartRunSchedule("prism", nil); err == nil || !strings.Contains(err.Error(), "invalid AkoScore") {
			t.Fatalf("error=%v", err)
		}
	})

	t.Run("valid symbol", func(t *testing.T) {
		openPlugin = func(string) (pluginSymbolLookup, error) {
			return pluginLookupFake{symbol: func(input any) float64 {
				return input.(map[string]any)["score"].(float64)
			}}, nil
		}
		result, err := service.StartRunSchedule("prism", map[string]any{"score": 8.5})
		if err != nil || result != 8.5 {
			t.Fatalf("result=%v error=%v", result, err)
		}
	})
}

func TestNewInitializesProductionDependencies(t *testing.T) {
	service := New()
	if service.scheduleRepository == nil || service.activityRepository == nil || service.resourceRepository == nil || service.resourceMetricsService == nil || service.scoreRunner == nil {
		t.Fatalf("New() returned incomplete service: %+v", service)
	}
}
