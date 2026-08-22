package environment

import (
	"context"
	"testing"

	"github.com/UFFeScience/akoflow/internal/application/ports"
	"github.com/UFFeScience/akoflow/internal/domain"
)

type monitorStore struct {
	connection domain.EnvironmentConnection
	checks     []domain.ConnectionCheck
	status     domain.EnvironmentStatus
}

func (s *monitorStore) Create(context.Context, domain.EnvironmentDefinition) error { return nil }
func (s *monitorStore) List(context.Context) ([]domain.EnvironmentDefinition, error) {
	return []domain.EnvironmentDefinition{{Connections: []domain.EnvironmentConnection{s.connection}}}, nil
}
func (s *monitorStore) Find(context.Context, string) (*domain.EnvironmentDefinition, error) {
	return nil, nil
}
func (s *monitorStore) UpdateStatus(_ context.Context, _ string, status domain.EnvironmentStatus) error {
	s.status = status
	return nil
}
func (s *monitorStore) UpsertConnection(context.Context, domain.EnvironmentConnection) error {
	return nil
}
func (s *monitorStore) ListConnections(context.Context, string) ([]domain.EnvironmentConnection, error) {
	return []domain.EnvironmentConnection{s.connection}, nil
}
func (s *monitorStore) FindConnection(context.Context, string) (*domain.EnvironmentConnection, error) {
	return &s.connection, nil
}
func (s *monitorStore) SaveConnectionCheck(_ context.Context, check domain.ConnectionCheck) error {
	s.checks = append(s.checks, check)
	return nil
}
func (s *monitorStore) ListConnectionChecks(context.Context, string, int) ([]domain.ConnectionCheck, error) {
	return s.checks, nil
}

type proberStub struct{ health ports.ConnectionHealth }

func (p proberStub) Probe(context.Context, domain.EnvironmentConnection) ports.ConnectionHealth {
	return p.health
}

func TestConnectionMonitorPersistsHealthAndEnvironmentStatus(t *testing.T) {
	store := &monitorStore{connection: domain.EnvironmentConnection{ID: "kind", EnvironmentID: "env", Type: domain.ConnectionKubernetes}}
	monitor := NewConnectionMonitor(store, map[domain.ConnectionType]ports.ConnectionProber{
		domain.ConnectionKubernetes: proberStub{health: ports.ConnectionHealth{Healthy: true, Message: "reachable"}},
	})
	check, err := monitor.Check(context.Background(), "kind")
	if err != nil || check.Status != domain.ConnectionOnline || store.status != domain.EnvironmentConnected || len(store.checks) != 1 {
		t.Fatalf("check=%+v status=%q checks=%d err=%v", check, store.status, len(store.checks), err)
	}
}

func TestConnectionMonitorRecordsUnsupportedConnectionOffline(t *testing.T) {
	store := &monitorStore{connection: domain.EnvironmentConnection{ID: "ssh", EnvironmentID: "env", Type: domain.ConnectionSSH}}
	monitor := NewConnectionMonitor(store, nil)
	check, err := monitor.Check(context.Background(), "ssh")
	if err != nil || check.Status != domain.ConnectionOffline || store.status != domain.EnvironmentUnreachable {
		t.Fatalf("check=%+v status=%q err=%v", check, store.status, err)
	}
}
