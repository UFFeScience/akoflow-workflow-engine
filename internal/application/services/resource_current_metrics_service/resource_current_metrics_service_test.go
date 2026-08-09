package resource_current_metrics_service

import (
	"errors"
	"testing"

	"github.com/UFFeScience/akoflow/internal/domain"
	workflow_activity_entity "github.com/UFFeScience/akoflow/internal/domain/workflow/activity"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/activity_repository"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/resource_repository"
)

type activityFake struct {
	activity_repository.IActivityRepository
	schedules   []workflow_activity_entity.ActivitySchedule
	running     []workflow_activity_entity.WorkflowActivities
	scheduleErr error
	runningErr  error
}

func (f activityFake) GetActivitySchedulesByResourceID(string) ([]workflow_activity_entity.ActivitySchedule, error) {
	return f.schedules, f.scheduleErr
}
func (f activityFake) GetAllRunningActivities() ([]workflow_activity_entity.WorkflowActivities, error) {
	return f.running, f.runningErr
}

type resourceFake struct {
	resource_repository.IRepository
	resource *domain.Resource
	err      error
}

func (f resourceFake) FindByID(string) (*domain.Resource, error) { return f.resource, f.err }

func TestMetricsGetCountsOnlyRunningAndCurrentAssignments(t *testing.T) {
	s := Service{
		resources: resourceFake{resource: &domain.Resource{ID: "r", CPUCapacity: 8, MemoryBytes: 16000}},
		activities: activityFake{
			schedules: []workflow_activity_entity.ActivitySchedule{{ActivityID: 1, CpuRequired: 2, MemoryRequired: 1000}, {ActivityID: 2, CpuRequired: 3, MemoryRequired: 2000}, {ActivityID: 9, CpuRequired: 7, MemoryRequired: 9000}},
			running:   []workflow_activity_entity.WorkflowActivities{{Id: 1}},
		},
	}
	metrics, err := s.Get("r", []workflow_activity_entity.WorkflowActivities{{Id: 2}})
	if err != nil {
		t.Fatal(err)
	}
	if metrics.CPUUsage != 5 || metrics.MemoryUsage != 3000 || metrics.CPUFree() != 3 || metrics.MemoryFree() != 13000 {
		t.Fatalf("unexpected metrics: %+v", metrics)
	}
}

func TestMetricsGetErrors(t *testing.T) {
	cases := []Service{
		{resources: resourceFake{}},
		{resources: resourceFake{err: errors.New("find")}},
		{resources: resourceFake{resource: &domain.Resource{}}, activities: activityFake{scheduleErr: errors.New("schedules")}},
		{resources: resourceFake{resource: &domain.Resource{}}, activities: activityFake{runningErr: errors.New("running")}},
	}
	for i, service := range cases {
		if _, err := service.Get("missing", nil); err == nil {
			t.Fatalf("case %d expected error", i)
		}
	}
}

func TestNewMetricsService(t *testing.T) { _ = New() }
