package database

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/UFFeScience/akoflow/internal/infrastructure/database/schema"
	_ "github.com/mattn/go-sqlite3"
)

func TestOpenCreatesParentDirectories(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "database", "akoflow.db")
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("database was not created: %v", err)
	}
}

func TestBootstrapInstallsAndValidatesCanonicalSchema(t *testing.T) {
	db := memoryDatabase(t)
	ctx := context.Background()
	if err := Bootstrap(ctx, db); err != nil {
		t.Fatal(err)
	}
	if err := Bootstrap(ctx, db); err != nil {
		t.Fatalf("valid schema should be reusable: %v", err)
	}
	var version int
	var checksum string
	if err := db.QueryRow(`SELECT version, checksum FROM schema_metadata`).Scan(&version, &checksum); err != nil {
		t.Fatal(err)
	}
	if version != schema.Version || checksum != schemaChecksum() {
		t.Fatalf("metadata = version %d checksum %q", version, checksum)
	}
}

func TestBootstrapRejectsPartialDatabase(t *testing.T) {
	db := memoryDatabase(t)
	if _, err := db.Exec(`CREATE TABLE stray(id INTEGER PRIMARY KEY)`); err != nil {
		t.Fatal(err)
	}
	err := Bootstrap(context.Background(), db)
	if !errors.Is(err, ErrIncompatibleSchema) {
		t.Fatalf("expected incompatible schema, got %v", err)
	}
}

func TestBootstrapRejectsChangedMetadata(t *testing.T) {
	db := memoryDatabase(t)
	ctx := context.Background()
	if err := Bootstrap(ctx, db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE schema_metadata SET checksum='changed'`); err != nil {
		t.Fatal(err)
	}
	if err := Validate(ctx, db); !errors.Is(err, ErrIncompatibleSchema) {
		t.Fatalf("expected incompatible schema, got %v", err)
	}
}

func TestValidateRejectsMissingCanonicalTable(t *testing.T) {
	db := memoryDatabase(t)
	ctx := context.Background()
	if err := Bootstrap(ctx, db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`DROP TABLE discovery_runs`); err != nil {
		t.Fatal(err)
	}
	if err := Validate(ctx, db); !errors.Is(err, ErrIncompatibleSchema) {
		t.Fatalf("expected incompatible schema, got %v", err)
	}
}

func memoryDatabase(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	return db
}
