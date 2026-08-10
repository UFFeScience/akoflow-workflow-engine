package activity_repository

import (
	"testing"

	"github.com/UFFeScience/akoflow/internal/application/ports"
	workflow_activity_entity "github.com/UFFeScience/akoflow/internal/domain/workflow/activity"
	workflow_entity "github.com/UFFeScience/akoflow/internal/domain/workflow/definition"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/workflow_repository"
)

func activityTestRepository(t *testing.T) (ports.ActivityRepository, int) {
	t.Helper()
	t.Setenv("AKOFLOW_DATABASE_PATH", t.TempDir()+"/akoflow.db")
	workflows := workflow_repository.New()
	workflow := workflow_entity.Workflow{Name: "wf", Spec: workflow_entity.WorkflowSpec{Runtime: "local", Image: "busybox", MountPath: "/data"}}
	id, err := workflows.Create("lab", workflow)
	if err != nil {
		t.Fatal(err)
	}
	return New(), id
}

func TestActivityRepositoryLifecycleDependenciesAndSchedule(t *testing.T) {
	repository, workflowID := activityTestRepository(t)
	workflow := workflow_entity.Workflow{Id: workflowID, Spec: workflow_entity.WorkflowSpec{Runtime: "local", Image: "busybox", MountPath: "/data"}}
	activities := []workflow_activity_entity.WorkflowActivities{
		{Name: "root", Run: "echo root", Runtime: "local"},
		{Name: "child", Run: "echo child", Runtime: "local", DependsOn: []string{"root"}},
	}
	if err := repository.Create("lab", workflow, activities); err != nil {
		t.Fatal(err)
	}
	stored, err := repository.GetByWorkflowId(workflowID)
	if err != nil || len(stored) != 2 {
		t.Fatalf("stored=%v err=%v", stored, err)
	}
	root, child := stored[0], stored[1]
	if root.Name != "root" {
		root, child = child, root
	}
	if err = repository.UpdateStatus(root.Id, ports.ActivityStatusRunning); err != nil {
		t.Fatal(err)
	}
	if err = repository.UpdateProcID(root.Id, "pid-1"); err != nil {
		t.Fatal(err)
	}
	if err = repository.UpdateResourceSelector(root.Id, "tier=edge"); err != nil {
		t.Fatal(err)
	}
	found, err := repository.Find(root.Id)
	if err != nil || found.Status != ports.ActivityStatusRunning || found.ProcId != "pid-1" || found.ResourceSelector != "tier=edge" {
		t.Fatalf("found=%+v err=%v", found, err)
	}
	dependencies, err := repository.GetWfaDependencies(workflowID)
	if err != nil || len(dependencies) != 1 || dependencies[0].ActivityId != child.Id || dependencies[0].DependsOnId != root.Id {
		t.Fatalf("dependencies=%v err=%v", dependencies, err)
	}

	if err = repository.SetActivitySchedule(workflowID, root.Id, "resource-1", "prism", 2, 512, `{}`); err != nil {
		t.Fatal(err)
	}
	scheduled, err := repository.IsActivityScheduled(workflowID, root.Id)
	if err != nil || !scheduled {
		t.Fatalf("scheduled=%v err=%v", scheduled, err)
	}
	assignment, err := repository.GetActivityScheduleByActivityId(root.Id)
	if err != nil || assignment.ResourceID != "resource-1" {
		t.Fatalf("assignment=%+v err=%v", assignment, err)
	}
	byResource, err := repository.GetActivitySchedulesByResourceID("resource-1")
	if err != nil || len(byResource) != 1 {
		t.Fatalf("byResource=%v err=%v", byResource, err)
	}
	if err = repository.UpdateStatus(root.Id, ports.ActivityStatusFinished); err != nil {
		t.Fatal(err)
	}
	running, err := repository.GetAllRunningActivities()
	if err != nil || len(running) != 0 {
		t.Fatalf("running=%v err=%v", running, err)
	}
}

func TestActivityRepositoryEmptyScheduleAndUnknownActivity(t *testing.T) {
	repository, workflowID := activityTestRepository(t)
	assignment, err := repository.GetActivityScheduleByActivityId(999)
	if err != nil || assignment.ActivityID != 0 {
		t.Fatalf("assignment=%+v err=%v", assignment, err)
	}
	scheduled, err := repository.IsActivityScheduled(workflowID, 999)
	if err != nil || scheduled {
		t.Fatalf("scheduled=%v err=%v", scheduled, err)
	}
}
