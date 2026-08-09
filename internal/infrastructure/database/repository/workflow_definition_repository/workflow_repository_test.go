package workflow_definition_repository

import (
	"path/filepath"
	"testing"

	"github.com/UFFeScience/akoflow/internal/domain"
)

func setupRepository(t *testing.T) IRepository {
	t.Helper()
	t.Setenv("AKOFLOW_DATABASE_PATH", filepath.Join(t.TempDir(), "db.sqlite"))
	return New()
}

func TestWorkflowDefinitionCreateAndFind(t *testing.T) {
	repository := setupRepository(t)
	definition := Definition{ID: "workflow", ExternalID: "external", Name: "WF", Namespace: "science", Types: []domain.ActivityType{{ID: "type", Name: "compute", Metadata: map[string]any{"kind": "cpu"}}}, Version: domain.WorkflowVersion{ID: "version", Version: 1, DefinitionHash: "hash", Activities: []domain.Activity{{ID: "a", ActivityTypeID: "type", ExternalID: "A", Name: "first", Command: "run", Metadata: map[string]any{"x": "y"}}, {ID: "b", ActivityTypeID: "type", ExternalID: "B", Name: "second"}}, Dependencies: []domain.ActivityDependency{{ActivityID: "b", DependsOnActivityID: "a", Type: "control"}}}}
	if err := repository.Create(definition); err != nil {
		t.Fatal(err)
	}
	got, err := repository.FindVersion("version")
	if err != nil || got == nil {
		t.Fatalf("find failed: %+v %v", got, err)
	}
	if got.WorkflowID != "workflow" || len(got.Activities) != 2 || len(got.Dependencies) != 1 || got.Activities[0].Metadata["x"] != "y" {
		t.Fatalf("unexpected workflow: %+v", got)
	}
	missing, err := repository.FindVersion("missing")
	if err != nil || missing != nil {
		t.Fatal("missing version must return nil")
	}
}

func TestWorkflowDefinitionDuplicateFails(t *testing.T) {
	repository := setupRepository(t)
	definition := Definition{ID: "same", Version: domain.WorkflowVersion{ID: "v"}}
	if err := repository.Create(definition); err != nil {
		t.Fatal(err)
	}
	if err := repository.Create(definition); err == nil {
		t.Fatal("duplicate definition must fail")
	}
}
