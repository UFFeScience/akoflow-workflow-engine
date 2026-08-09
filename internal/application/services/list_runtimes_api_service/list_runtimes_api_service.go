package list_runtimes_api_service

import (
	"github.com/UFFeScience/akoflow/internal/api/mapper/mapper_engine_api"
	"github.com/UFFeScience/akoflow/internal/api/requests"
	"github.com/UFFeScience/akoflow/internal/infrastructure/config"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/runtime_repository"
)

type ListRuntimesApiService struct {
	runtimeRepository runtime_repository.IRuntimeRepository
}

func New() *ListRuntimesApiService {
	return NewWithRepository(config.App().Repository.RuntimeRepository)
}

func NewWithRepository(repository runtime_repository.IRuntimeRepository) *ListRuntimesApiService {
	return &ListRuntimesApiService{runtimeRepository: repository}
}

func (h *ListRuntimesApiService) ListAllRuntimes() ([]types_api.ApiRuntimeType, error) {
	runtimesEngine, err := h.runtimeRepository.GetAll()

	if err != nil {
		return nil, err
	}

	runtimeApi := mapper_engine_api.MapEngineRuntimeEntityToApiRuntimeEntityList(runtimesEngine)
	return runtimeApi, nil
}
