package list_schedules_api_service

import (
	"github.com/UFFeScience/akoflow/internal/api/requests"
	"github.com/UFFeScience/akoflow/internal/infrastructure/config"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/schedule_repository"
)

type ListSchedulesApiService struct {
	scheduleRepository schedule_repository.IScheduleRepository
}

func New() *ListSchedulesApiService {
	return NewWithRepository(config.App().Repository.ScheduleRepository)
}

func NewWithRepository(repository schedule_repository.IScheduleRepository) *ListSchedulesApiService {
	return &ListSchedulesApiService{scheduleRepository: repository}
}

func (h *ListSchedulesApiService) ListAllSchedules() ([]types_api.ApiScheduleType, error) {
	schedulesEngine, err := h.scheduleRepository.ListAllSchedules()

	if err != nil {
		return nil, err
	}

	schedulesApi := make([]types_api.ApiScheduleType, 0, len(schedulesEngine))
	for _, schedule := range schedulesEngine {
		schedulesApi = append(schedulesApi, types_api.ApiScheduleType{
			ID:        schedule.ID,
			Type:      schedule.Type,
			Code:      schedule.Code,
			Name:      schedule.Name,
			CreatedAt: schedule.CreatedAt,
			UpdatedAt: schedule.UpdatedAt,
		})
	}

	return schedulesApi, nil
}
