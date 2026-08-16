package health_check_runtime_check_service

import (
	"fmt"

	"github.com/UFFeScience/akoflow/internal/execution/real/runtimes"
	"github.com/UFFeScience/akoflow/internal/infrastructure/config"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/runtime_repository"
)

type HealthCheckRuntimeCheckService struct {
	State             int
	runtimeRepository runtime_repository.IRuntimeRepository
	runtimeResolver   RuntimeResolver
}

type RuntimeHealth interface{ HealthCheck() bool }
type RuntimeResolver func(string) RuntimeHealth

func New() *HealthCheckRuntimeCheckService {
	return &HealthCheckRuntimeCheckService{
		State: 0, runtimeResolver: func(id string) RuntimeHealth { return runtimes.GetRuntimeInstance(id) },
	}
}
func NewHealthCheckRuntimeCheckService() *HealthCheckRuntimeCheckService {
	return &HealthCheckRuntimeCheckService{
		State:             0,
		runtimeRepository: config.App().Repository.RuntimeRepository,
		runtimeResolver:   func(id string) RuntimeHealth { return runtimes.GetRuntimeInstance(id) },
	}
}

func NewWithDependencies(resolver RuntimeResolver) *HealthCheckRuntimeCheckService {
	return &HealthCheckRuntimeCheckService{runtimeResolver: resolver}
}

func (w *HealthCheckRuntimeCheckService) Handle(runtimeName string) error {
	runtime := w.runtimeResolver(runtimeName)
	if runtime == nil {
		return fmt.Errorf("runtime %s not found", runtimeName)
	}
	if !runtime.HealthCheck() {
		return fmt.Errorf("runtime %s is unhealthy", runtimeName)
	}
	return nil
}
