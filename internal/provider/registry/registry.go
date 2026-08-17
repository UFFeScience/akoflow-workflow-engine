package registry

import (
	"fmt"
	"sync"

	"github.com/UFFeScience/akoflow/internal/application/ports"
	"github.com/UFFeScience/akoflow/internal/domain"
)

type Registry struct {
	mu       sync.RWMutex
	runtimes map[string]ports.RuntimeAdapter
}

func New() *Registry {
	return &Registry{runtimes: make(map[string]ports.RuntimeAdapter)}
}

func (r *Registry) Register(runtimeID string, adapter ports.RuntimeAdapter) error {
	if runtimeID == "" || adapter == nil {
		return fmt.Errorf("runtime id and adapter are required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	modes := adapter.Modes()
	if len(modes) == 0 {
		return fmt.Errorf("runtime %q does not declare execution modes", runtimeID)
	}
	for _, mode := range modes {
		key := runtimeKey(mode, runtimeID)
		if _, exists := r.runtimes[key]; exists {
			return fmt.Errorf("runtime %q is already registered for mode %q", runtimeID, mode)
		}
	}
	for _, mode := range modes {
		r.runtimes[runtimeKey(mode, runtimeID)] = adapter
	}
	return nil
}

func (r *Registry) Resolve(mode domain.ExecutionMode, runtimeID string) (ports.RuntimeAdapter, error) {
	r.mu.RLock()
	adapter := r.runtimes[runtimeKey(mode, runtimeID)]
	if adapter == nil {
		adapter = r.runtimes[runtimeKey(mode, "*")]
	}
	r.mu.RUnlock()
	if adapter == nil {
		return nil, fmt.Errorf("runtime %q is not registered for mode %q", runtimeID, mode)
	}
	return adapter, nil
}

func runtimeKey(mode domain.ExecutionMode, runtimeID string) string {
	return string(mode) + "\x00" + runtimeID
}
