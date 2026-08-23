package schema

import (
	"strings"
	"testing"
)

func TestCanonicalSchemaIsEmbedded(t *testing.T) {
	if Version != 5 {
		t.Fatalf("schema version = %d", Version)
	}
	for _, definition := range []string{
		"CREATE TABLE schema_metadata",
		"CREATE TABLE environments",
		"CREATE TABLE environment_connection_checks",
		"CREATE TABLE audit_events",
		"CREATE TABLE console_commands",
		"CREATE TABLE console_session_logs",
		"CREATE TABLE workflow_definitions",
		"CREATE TABLE execution_runs",
		"CREATE TABLE queue_jobs",
	} {
		if !strings.Contains(SQL, definition) {
			t.Errorf("canonical schema does not contain %q", definition)
		}
	}
}
