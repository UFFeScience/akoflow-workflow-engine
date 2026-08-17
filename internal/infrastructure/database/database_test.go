package database

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConnectUsesConfiguredPathAndCreatesParentDirectories(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "database", "akoflow.db")
	t.Setenv("AKOFLOW_DATABASE_PATH", path)
	db := (&Database{}).Connect()
	if _, err := db.Exec("CREATE TABLE sample(id INTEGER PRIMARY KEY)"); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("database was not created: %v", err)
	}
}

func TestCreateDirectoryIsIdempotent(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "a", "b")
	createDirectoryIfNotExists(directory)
	createDirectoryIfNotExists(directory)
	if info, err := os.Stat(directory); err != nil || !info.IsDir() {
		t.Fatalf("directory info=%v err=%v", info, err)
	}
}
