package environment

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/UFFeScience/akoflow/internal/application/ports"
	"github.com/UFFeScience/akoflow/internal/domain"
)

type ConnectionMonitor struct {
	store   connectionCatalog
	probers map[domain.ConnectionType]ports.ConnectionProber
	mu      sync.Mutex
}

type connectionCatalog interface {
	ports.EnvironmentCatalog
	ports.ConnectionStore
}

var _ ports.ConnectionHealthMonitor = (*ConnectionMonitor)(nil)

func NewConnectionMonitor(
	store connectionCatalog,
	probers map[domain.ConnectionType]ports.ConnectionProber,
) *ConnectionMonitor {
	return &ConnectionMonitor{store: store, probers: probers}
}

func (m *ConnectionMonitor) Check(ctx context.Context, connectionID string) (domain.ConnectionCheck, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	connection, err := m.store.FindConnection(ctx, connectionID)
	if err != nil {
		return domain.ConnectionCheck{}, err
	}
	if connection == nil {
		return domain.ConnectionCheck{}, fmt.Errorf("environment connection %q was not found", connectionID)
	}
	started := time.Now()
	health := ports.ConnectionHealth{Message: "no connection prober is configured"}
	if prober := m.probers[connection.Type]; prober != nil {
		health = prober.Probe(ctx, *connection)
	}
	checkedAt := time.Now().UTC()
	status := domain.ConnectionOffline
	environmentStatus := domain.EnvironmentUnreachable
	if health.Healthy {
		status = domain.ConnectionOnline
		environmentStatus = domain.EnvironmentConnected
	}
	check := domain.ConnectionCheck{
		ID: fmt.Sprintf("connection-check-%d", checkedAt.UnixNano()), ConnectionID: connection.ID,
		Status: status, Message: health.Message,
		LatencyMS: float64(time.Since(started).Microseconds()) / 1000, CheckedAt: checkedAt,
		Metadata: map[string]any{"endpoint": connection.Endpoint, "connectionType": connection.Type},
	}
	if err := m.store.SaveConnectionCheck(ctx, check); err != nil {
		return domain.ConnectionCheck{}, err
	}
	if err := m.store.UpdateStatus(ctx, connection.EnvironmentID, environmentStatus); err != nil {
		return domain.ConnectionCheck{}, err
	}
	return check, nil
}

func (m *ConnectionMonitor) History(
	ctx context.Context,
	connectionID string,
	limit int,
) ([]domain.ConnectionCheck, error) {
	return m.store.ListConnectionChecks(ctx, connectionID, limit)
}

func (m *ConnectionMonitor) CheckAll(ctx context.Context) {
	definitions, err := m.store.List(ctx)
	if err != nil {
		return
	}
	for _, definition := range definitions {
		for _, connection := range definition.Connections {
			if ctx.Err() != nil {
				return
			}
			_, _ = m.Check(ctx, connection.ID)
		}
	}
}

func (m *ConnectionMonitor) Run(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	m.CheckAll(ctx)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.CheckAll(ctx)
		}
	}
}
