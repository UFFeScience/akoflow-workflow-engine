package schema

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestApplyCreatesCanonicalTables(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := Apply(db); err != nil {
		t.Fatalf("apply schema: %v", err)
	}

	for _, table := range []string{
		"runtimes", "workflows", "activities", "activities_dependencies",
		"pre_activities", "activities_schedules", "storages", "logs",
		"metrics", "workflow_executions", "schedules", "environments",
		"resources", "schedule_plans", "execution_runs", "data_transfers",
	} {
		var name string
		err := db.QueryRow(
			"SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?",
			table,
		).Scan(&name)
		if err != nil {
			t.Errorf("table %s was not created: %v", table, err)
		}
	}
}

func TestApplyIsIdempotent(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := Apply(db); err != nil {
		t.Fatal(err)
	}
	if err := Apply(db); err != nil {
		t.Fatalf("second schema application must be safe: %v", err)
	}
}
