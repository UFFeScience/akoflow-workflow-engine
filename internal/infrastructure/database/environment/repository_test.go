package environment

import (
	"context"
	"database/sql"
	"testing"

	"github.com/UFFeScience/akoflow/internal/domain"
	database "github.com/UFFeScience/akoflow/internal/infrastructure/database"
	_ "github.com/mattn/go-sqlite3"
)

func setupRepository(t *testing.T) *Repository {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	if err := database.Bootstrap(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	return New(db)
}

func TestEnvironmentDefinitionCreate(t *testing.T) {
	repository := setupRepository(t)
	definition := Definition{
		Environment: domain.Environment{ID: "env", Name: "hybrid", Description: "test"},
		Version:     domain.EnvironmentVersion{ID: "v1", Version: 1, Status: domain.EnvironmentVersionPublished, NetworkModel: "real", InterferenceModel: "none", CostModel: "aws", ConfigurationHash: "hash"},
		Runtimes:    []domain.EnvironmentRuntime{{RuntimeID: "k8s", Role: "cloud", Configuration: map[string]any{"region": "us"}}},
		Storages: []domain.StorageResource{{ID: "shared", Name: "shared", Type: domain.StorageNFS,
			Endpoint: "nfs:/akoflow", Shared: true, RuntimeBindings: []domain.StorageRuntimeBinding{{
				RuntimeID: "k8s", Default: true, HostPath: "/shared/akoflow"}}}},
		Resources: []domain.Resource{
			{ID: "cluster", RuntimeID: "k8s", Type: domain.ResourceCluster, Name: "cluster", ProviderID: "cluster"},
			{ID: "r1", RuntimeID: "k8s", Type: domain.ResourceCloudVM, Name: "vm", ProviderID: "provider", CPUCores: 2, CPUCapacity: 2, MemoryBytes: 1024, Schedulable: true, Metadata: map[string]any{"tier": "cloud"}},
		},
		Relations: []domain.ResourceRelation{{
			SourceResourceID: "cluster", TargetResourceID: "r1", Type: domain.ResourceRelationContains,
		}},
		Connections: []domain.EnvironmentConnection{{ID: "c1", Name: "cluster", Type: domain.ConnectionSSH, Endpoint: "login.example", Username: "user", CredentialRef: "keychain:test", Configuration: map[string]any{"port": float64(22)}}},
	}
	if err := repository.Create(context.Background(), definition); err != nil {
		t.Fatal(err)
	}
	found, err := repository.Find(context.Background(), "env")
	if err != nil || found == nil || len(found.Relations) != 1 {
		t.Fatalf("resource relations were not loaded: %+v %v", found, err)
	}
	if err := repository.Create(context.Background(), definition); err == nil {
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
	storage, err := repository.FindDefaultRuntimeStorage(context.Background(), "v1", "k8s")
	if err != nil || storage.ID != "shared" || storage.RuntimeBindings[0].ContainerPath != "/akoflow/data" {
		t.Fatalf("storage=%+v err=%v", storage, err)
	}
}

func TestEnvironmentRejectsTwoDefaultStoragesForRuntime(t *testing.T) {
	repository := setupRepository(t)
	definition := Definition{
		Environment: domain.Environment{ID: "env", Name: "cluster"},
		Version: domain.EnvironmentVersion{ID: "v1", Version: 1, Status: domain.EnvironmentVersionPublished,
			NetworkModel: "real", InterferenceModel: "none", CostModel: "free"},
		Runtimes: []domain.EnvironmentRuntime{{RuntimeID: "slurm"}},
	}
	binding := []domain.StorageRuntimeBinding{{RuntimeID: "slurm", Default: true}}
	definition.Storages = []domain.StorageResource{
		{ID: "one", Name: "one", Type: domain.StorageLustre, RuntimeBindings: binding},
		{ID: "two", Name: "two", Type: domain.StorageNFS, RuntimeBindings: binding},
	}
	if err := repository.Create(context.Background(), definition); err == nil {
		t.Fatal("two default storages for the same runtime must fail")
	}
}
