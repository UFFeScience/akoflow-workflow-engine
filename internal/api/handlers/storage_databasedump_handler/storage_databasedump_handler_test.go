package storage_databasedump_handler

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestDatabaseDumpMissingAndExisting(t *testing.T) {
	original := PATH
	defer func() { PATH = original }()
	PATH = filepath.Join(t.TempDir(), "missing.db")
	handler := New()
	recorder := httptest.NewRecorder()
	handler.DatabaseDumpHandler(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
	if recorder.Body.String() != `{"status": "error", "message": "File not found"}` || handler.fileDumpExists() {
		t.Fatal("missing response failed")
	}
	if err := os.WriteFile(PATH, []byte("database"), 0o600); err != nil {
		t.Fatal(err)
	}
	recorder = httptest.NewRecorder()
	handler.DatabaseDumpHandler(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
	if !handler.fileDumpExists() || recorder.Body.String() != "database" || recorder.Header().Get("Content-Disposition") == "" {
		t.Fatalf("download failed: %q", recorder.Body.String())
	}
}
