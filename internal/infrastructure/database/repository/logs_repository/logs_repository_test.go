package logs_repository

import (
	"testing"

	"github.com/UFFeScience/akoflow/internal/application/ports"
	base "github.com/UFFeScience/akoflow/internal/infrastructure/database/repository"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/schema"
)

func TestLogsRepositoryPersistsActivityLog(t *testing.T) {
	t.Setenv("AKOFLOW_DATABASE_PATH", t.TempDir()+"/akoflow.db")
	db := (&base.Database{}).Connect()
	defer db.Close()
	if err := schema.Apply(db); err != nil {
		t.Fatal(err)
	}
	db.Exec("INSERT INTO workflows(id, namespace, runtime, name, raw_workflow, status) VALUES(1,'','','','',0)")
	db.Exec("INSERT INTO activities(id, workflow_id, namespace, name, image, runtime, resource_k8s_base64, status) VALUES(2,1,'','','','','',0)")
	repository := New()
	if err := repository.Create(ports.ActivityLog{ActivityID: 2, Logs: "finished"}); err != nil {
		t.Fatal(err)
	}
	var value string
	if err := db.QueryRow("SELECT logs FROM logs WHERE activity_id=2").Scan(&value); err != nil || value != "finished" {
		t.Fatalf("value=%q err=%v", value, err)
	}
}
