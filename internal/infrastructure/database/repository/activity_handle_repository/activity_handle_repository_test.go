package activity_handle_repository

import (
	"context"
	"database/sql"
	"testing"

	"github.com/UFFeScience/akoflow/internal/domain"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/schema"
	_ "github.com/mattn/go-sqlite3"
)

func TestSaveFindAndUpdateHandle(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if err := schema.Apply(db); err != nil {
		t.Fatal(err)
	}
	repository := New(db)
	handle := domain.ActivityHandle{ID: "h", RunID: "run", ActivityID: "activity",
		ResourceID: "node", RuntimeID: "local", ExternalID: "42",
		Status: domain.HandleRunning, Endpoints: []string{"ssh://node"},
		Metadata: map[string]any{"key": "value"}}
	if err := repository.Save(context.Background(), handle); err != nil {
		t.Fatal(err)
	}
	got, err := repository.Find(context.Background(), "h")
	if err != nil || got == nil || got.ExternalID != "42" || len(got.Endpoints) != 1 {
		t.Fatalf("unexpected handle: %+v %v", got, err)
	}
	handle.Status = domain.HandleCompleted
	exitCode := 0
	handle.ExitCode = &exitCode
	if err := repository.Save(context.Background(), handle); err != nil {
		t.Fatal(err)
	}
	got, err = repository.Find(context.Background(), "h")
	if err != nil || got.Status != domain.HandleCompleted || got.ExitCode == nil {
		t.Fatalf("unexpected update: %+v %v", got, err)
	}
	missing, err := repository.Find(context.Background(), "missing")
	if err != nil || missing != nil {
		t.Fatal("missing handle must return nil")
	}
}
