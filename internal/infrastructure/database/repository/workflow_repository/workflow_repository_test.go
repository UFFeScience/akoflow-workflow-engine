package workflow_repository

import (
	"testing"

	"github.com/UFFeScience/akoflow/internal/application/ports"
	workflow_entity "github.com/UFFeScience/akoflow/internal/domain/workflow/definition"
)

func workflowTestRepository(t *testing.T) ports.WorkflowRepository {
	t.Helper()
	t.Setenv("AKOFLOW_DATABASE_PATH", t.TempDir()+"/akoflow.db")
	repository := New()
	if repository == nil {
		t.Fatal("New() returned nil")
	}
	return repository
}

func TestWorkflowRepositoryLifecycleAndPagination(t *testing.T) {
	repository := workflowTestRepository(t)
	first := workflow_entity.Workflow{Name: "first", Spec: workflow_entity.WorkflowSpec{Runtime: "local", Namespace: "lab"}}
	second := workflow_entity.Workflow{Name: "second", Spec: workflow_entity.WorkflowSpec{Runtime: "k8s", Namespace: "lab"}}
	id1, err := repository.Create("lab", first)
	if err != nil {
		t.Fatal(err)
	}
	id2, err := repository.Create("lab", second)
	if err != nil {
		t.Fatal(err)
	}

	found, err := repository.Find(id1)
	if err != nil || found.Id != id1 || found.Name != "first" {
		t.Fatalf("found=%+v err=%v", found, err)
	}
	pending, err := repository.GetPendingWorkflows("lab")
	if err != nil || len(pending) != 2 {
		t.Fatalf("pending=%v err=%v", pending, err)
	}
	if err = repository.UpdateStatus(id1, ports.WorkflowStatusFinished); err != nil {
		t.Fatal(err)
	}
	pending, _ = repository.GetPendingWorkflows("lab")
	if len(pending) != 1 || pending[0].Id != id2 {
		t.Fatalf("pending after finish=%v", pending)
	}

	page, perPage := 0, 1
	listed, err := repository.ListAllWorkflows(&ports.WorkflowListOptions{Page: &page, PerPage: &perPage})
	if err != nil || len(listed) != 1 || listed[0].Id != id2 {
		t.Fatalf("listed=%v err=%v", listed, err)
	}
	all, err := repository.ListAllWorkflows(nil)
	if err != nil || len(all) != 2 {
		t.Fatalf("all=%v err=%v", all, err)
	}
}

func TestWorkflowRepositoryReturnsNotFound(t *testing.T) {
	repository := workflowTestRepository(t)
	if _, err := repository.Find(999); err == nil {
		t.Fatal("expected not found error")
	}
}
