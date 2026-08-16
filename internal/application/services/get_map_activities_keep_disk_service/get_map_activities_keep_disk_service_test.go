package get_map_activities_keep_disk_service

import (
	"errors"
	"testing"

	"github.com/UFFeScience/akoflow/internal/application/ports"
	workflow_activity_entity "github.com/UFFeScience/akoflow/internal/domain/workflow/activity"
)

type activityFake struct {
	ports.ActivityRepository
	values ports.ActivitiesByWorkflow
	err    error
}

func (f activityFake) GetActivitiesByWorkflowIds([]int) (ports.ActivitiesByWorkflow, error) {
	return f.values, f.err
}

type dependencyFake struct {
	values workflow_activity_entity.MapActivityDependencies
}

func (f dependencyFake) GetActivityDependencies(int) workflow_activity_entity.MapActivityDependencies {
	return f.values
}

func TestKeepDiskDefaultsTrueAndDisablesReusableDependency(t *testing.T) {
	service := NewWithDependencies(activityFake{values: ports.ActivitiesByWorkflow{1: {{Id: 1}, {Id: 2}}}}, dependencyFake{values: workflow_activity_entity.MapActivityDependencies{2: {{Id: 1, KeepDisk: false}}}})
	result, err := service.GetMapActivitiesKeepDisk(1)
	if err != nil || result[1] || !result[2] {
		t.Fatalf("result=%v err=%v", result, err)
	}
}
func TestKeepDiskPropagatesActivityError(t *testing.T) {
	service := NewWithDependencies(activityFake{err: errors.New("db")}, dependencyFake{})
	if _, err := service.GetMapActivitiesKeepDisk(1); err == nil {
		t.Fatal("expected error")
	}
}
func TestNewInitializesDependencies(t *testing.T) {
	s := New()
	if s.activityRepository == nil || s.getActivityDependeciesService == nil {
		t.Fatalf("incomplete: %+v", s)
	}
}
