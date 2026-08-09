package resource_current_metrics_service

import (
	"fmt"

	"github.com/UFFeScience/akoflow/internal/domain/workflow/activity"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/activity_repository"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/resource_repository"
)

type Metrics struct {
	CPUUsage    float64 `json:"cpuUsage"`
	MemoryUsage float64 `json:"memoryUsage"`
	CPUTotal    float64 `json:"cpuTotal"`
	MemoryTotal float64 `json:"memoryTotal"`
}

func (m Metrics) CPUFree() float64    { return m.CPUTotal - m.CPUUsage }
func (m Metrics) MemoryFree() float64 { return m.MemoryTotal - m.MemoryUsage }

type Service struct {
	activities activity_repository.IActivityRepository
	resources  resource_repository.IRepository
}

func New() Service {
	return Service{activities: activity_repository.New(), resources: resource_repository.New()}
}

func (s Service) Get(resourceID string, current []workflow_activity_entity.WorkflowActivities) (*Metrics, error) {
	resource, err := s.resources.FindByID(resourceID)
	if err != nil || resource == nil {
		return nil, fmt.Errorf("resource %q not found", resourceID)
	}
	result := &Metrics{CPUTotal: resource.CPUCapacity, MemoryTotal: float64(resource.MemoryBytes)}
	schedules, err := s.activities.GetActivitySchedulesByResourceID(resourceID)
	if err != nil {
		return nil, err
	}
	running, err := s.activities.GetAllRunningActivities()
	if err != nil {
		return nil, err
	}
	runningIDs := make(map[int]bool, len(running)+len(current))
	for _, activity := range running {
		runningIDs[activity.Id] = true
	}
	for _, activity := range current {
		runningIDs[activity.Id] = true
	}
	for _, schedule := range schedules {
		if runningIDs[schedule.ActivityID] {
			result.CPUUsage += schedule.CpuRequired
			result.MemoryUsage += schedule.MemoryRequired
		}
	}
	return result, nil
}
