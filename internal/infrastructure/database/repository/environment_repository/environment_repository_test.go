package environment_repository

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/UFFeScience/akoflow/internal/domain"
)

func setupRepository(t *testing.T) *Repository {
	t.Helper()
	t.Setenv("AKOFLOW_DATABASE_PATH", filepath.Join(t.TempDir(), "db.sqlite"))
	return New().(*Repository)
}

func TestEnvironmentDefinitionCreate(t *testing.T) {
	repository := setupRepository(t)
	definition := Definition{
		Environment: domain.Environment{ID: "env", Name: "hybrid", Description: "test"},
		Version:     domain.EnvironmentVersion{ID: "v1", Version: 1, Status: domain.EnvironmentVersionPublished, NetworkModel: "real", InterferenceModel: "none", CostModel: "aws", StorageModel: "shared", ConfigurationHash: "hash"},
		Runtimes:    []domain.EnvironmentRuntime{{RuntimeID: "k8s", Role: "cloud", Configuration: map[string]any{"region": "us"}}},
		Resources:   []domain.Resource{{ID: "r1", RuntimeID: "k8s", Type: domain.ResourceCloudVM, Name: "vm", ProviderID: "provider", CPUCores: 2, CPUCapacity: 2, MemoryBytes: 1024, Schedulable: true, Metadata: map[string]any{"tier": "cloud"}}},
		Links:       []domain.NetworkLink{{ID: "l1", SourceResourceID: "r1", TargetResourceID: "r1", BandwidthBitsPerSecond: 500e6, LatencySeconds: .1, Bidirectional: true, Metadata: map[string]any{"kind": "loop"}}},
		Profiles:    []domain.ActivityResourceProfile{{ID: "p1", ActivityTypeID: "type", ResourceID: "r1", RuntimeSeconds: 2, Source: "measured", Metadata: map[string]any{"samples": 3}}},
		Connections: []domain.EnvironmentConnection{{ID: "c1", Name: "cluster", Type: domain.ConnectionSSH, Endpoint: "login.example", Username: "user", CredentialRef: "keychain:test", Configuration: map[string]any{"port": float64(22)}}},
	}
	if err := repository.Create(definition); err != nil {
		t.Fatal(err)
	}
	if err := repository.Create(definition); err == nil {
		t.Fatal("duplicate environment must fail")
	}
	connections, err := repository.ListConnections(context.Background(), "env")
	if err != nil || len(connections) != 1 || connections[0].CredentialRef != "keychain:test" {
		t.Fatalf("connections=%+v err=%v", connections, err)
	}
	connection := connections[0]
	connection.Endpoint = "new-login.example"
	if err := repository.UpsertConnection(context.Background(), connection); err != nil {
		t.Fatal(err)
	}
	connections, _ = repository.ListConnections(context.Background(), "env")
	if connections[0].Endpoint != "new-login.example" {
		t.Fatal("connection upsert failed")
	}
	if err := repository.UpdateStatus(context.Background(), "env", domain.EnvironmentReady); err != nil {
		t.Fatal(err)
	}
	if err := repository.UpdateStatus(context.Background(), "missing", domain.EnvironmentReady); err == nil {
		t.Fatal("updating a missing environment must fail")
	}
}
