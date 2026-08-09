package health_check_runtime_check_service

import (
	"github.com/UFFeScience/akoflow/internal/execution/real/runtimes"
	"github.com/UFFeScience/akoflow/internal/infrastructure/config"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/runtime_repository"
)

type HealthCheckRuntimeCheckService struct {
	State             int
	runtimeRepository runtime_repository.IRuntimeRepository
}

func New() *HealthCheckRuntimeCheckService {
	return &HealthCheckRuntimeCheckService{
		State: 0,
	}
}
func NewHealthCheckRuntimeCheckService() *HealthCheckRuntimeCheckService {
	return &HealthCheckRuntimeCheckService{
		State:             0,
		runtimeRepository: config.App().Repository.RuntimeRepository,
	}
}

func (w *HealthCheckRuntimeCheckService) Handle(runtimeName string) {

	runtimes.
		GetRuntimeInstance(runtimeName).
		HealthCheck()

}
