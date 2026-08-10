package metrics_repository

import (
	"testing"

	"github.com/UFFeScience/akoflow/internal/application/ports"
	base "github.com/UFFeScience/akoflow/internal/infrastructure/database/repository"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/schema"
)

func TestMetricsRepositoryPersistsActivityMetric(t *testing.T) {
	t.Setenv("AKOFLOW_DATABASE_PATH", t.TempDir()+"/akoflow.db")
	db := (&base.Database{}).Connect()
	defer db.Close()
	if err := schema.Apply(db); err != nil {
		t.Fatal(err)
	}
	db.Exec("INSERT INTO workflows(id, namespace, runtime, name, raw_workflow, status) VALUES(1,'','','','',0)")
	db.Exec("INSERT INTO activities(id, workflow_id, namespace, name, image, runtime, resource_k8s_base64, status) VALUES(2,1,'','','','','',0)")
	repository := New()
	metric := ports.ActivityMetric{ActivityID: 2, CPU: "250m", Memory: "64Mi", Window: "1s", Timestamp: "now"}
	if err := repository.Create(metric); err != nil {
		t.Fatal(err)
	}
	var cpu, memory string
	if err := db.QueryRow("SELECT cpu, memory FROM metrics WHERE activity_id=2").Scan(&cpu, &memory); err != nil || cpu != "250m" || memory != "64Mi" {
		t.Fatalf("cpu=%q memory=%q err=%v", cpu, memory, err)
	}
}
