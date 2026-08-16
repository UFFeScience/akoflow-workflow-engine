package orchestrator

import (
	"context"
	"testing"

	"github.com/UFFeScience/akoflow/internal/domain"
)

type adapterFake struct{ mode domain.ExecutionMode }

func (a adapterFake) Modes() []domain.ExecutionMode { return []domain.ExecutionMode{a.mode} }
func (adapterFake) Start(context.Context, domain.ActivityExecutionContext) (domain.ActivityHandle, error) {
	return domain.ActivityHandle{}, nil
}
func (adapterFake) Inspect(_ context.Context, h domain.ActivityHandle) (domain.ActivityHandle, error) {
	return h, nil
}
func (adapterFake) Stop(context.Context, domain.ActivityHandle) error { return nil }

func TestRuntimeRegistryResolvesExactAndFallbackAdapters(t *testing.T) {
	registry := NewRuntimeRegistry()
	if err := registry.Register("*", adapterFake{mode: domain.ExecutionModeSimulation}); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.Resolve(domain.ExecutionModeSimulation, "simgrid"); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register("local", adapterFake{mode: domain.ExecutionModeReal}); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.Resolve(domain.ExecutionModeReal, "local"); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.Resolve(domain.ExecutionModeReal, "missing"); err == nil {
		t.Fatal("missing runtime must fail")
	}
	if err := registry.Register("local", adapterFake{mode: domain.ExecutionModeReal}); err == nil {
		t.Fatal("duplicate registration must fail")
	}
}
