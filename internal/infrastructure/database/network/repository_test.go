package network

import (
	"context"
	"database/sql"
	"testing"

	"github.com/UFFeScience/akoflow/internal/domain"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func TestRepositoryPersistsFederatedTopology(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	defer db.Close()
	require.NoError(t, database.Bootstrap(context.Background(), db))
	statements := []string{
		`INSERT INTO runtimes(name) VALUES ('runtime')`,
		`INSERT INTO environments(id,name) VALUES ('hpc','HPC'),('cloud','Cloud')`,
		`INSERT INTO environment_versions(id,environment_id,version,status,network_model,interference_model,cost_model,configuration_hash) VALUES ('hpc-v1','hpc',1,'published','','','','h'),('cloud-v1','cloud',1,'published','','','','c')`,
		`INSERT INTO resources(id,environment_version_id,runtime_id,type,name,provider_id) VALUES ('hpc-node','hpc-v1','runtime','hpc_machine','HPC','hpc'),('cloud-vm','cloud-v1','runtime','cloud_vm','Cloud','cloud')`,
	}
	for _, statement := range statements {
		_, err := db.Exec(statement)
		require.NoError(t, err)
	}
	repository := New(db)
	topology := domain.NetworkTopology{ID: "federated-v1", Name: "HPC Cloud", Version: 1, Scope: "federated",
		Links: []domain.NetworkLink{{ID: "hpc-cloud", SourceResourceID: "hpc-node",
			TargetResourceID: "cloud-vm", BandwidthBitsPerSecond: 500e6,
			LatencySeconds: .1, Bidirectional: true, Metadata: map[string]any{"carrier": "internet"}}}}
	require.NoError(t, repository.Create(context.Background(), topology))
	stored, err := repository.Find(context.Background(), topology.ID)
	require.NoError(t, err)
	require.Equal(t, "federated", stored.Scope)
	require.Equal(t, "cloud-vm", stored.Links[0].TargetResourceID)
	require.Equal(t, "internet", stored.Links[0].Metadata["carrier"])
}
