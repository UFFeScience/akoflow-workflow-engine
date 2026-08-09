package get_schedule_api_service

import (
	"github.com/UFFeScience/akoflow/internal/api/requests"
	"github.com/UFFeScience/akoflow/internal/infrastructure/config"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/schedule_repository"
)

type GetScheduleApiService struct {
	scheduleRepository schedule_repository.IScheduleRepository
}

func New() *GetScheduleApiService {
	return NewWithRepository(config.App().Repository.ScheduleRepository)
}

func NewWithRepository(repository schedule_repository.IScheduleRepository) *GetScheduleApiService {
	return &GetScheduleApiService{scheduleRepository: repository}
}

func (h *GetScheduleApiService) GetScheduleByName(scheduleId string) (*types_api.ApiScheduleType, error) {
	scheduleEngine, err := h.scheduleRepository.GetScheduleByName(scheduleId)

	if err != nil {
		return nil, err
	}

	scheduleApi := &types_api.ApiScheduleType{
		ID:        scheduleEngine.ID,
		Type:      scheduleEngine.Type,
		Code:      scheduleEngine.Code,
		Name:      scheduleEngine.Name,
		CreatedAt: "",
		UpdatedAt: "",
	}

	return scheduleApi, nil
}
