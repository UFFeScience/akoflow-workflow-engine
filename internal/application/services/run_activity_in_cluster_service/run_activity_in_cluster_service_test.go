package run_activity_in_cluster_service

import (
	"errors"
	"strings"
	"testing"

	"github.com/UFFeScience/akoflow/internal/domain"
	workflow_activity_entity "github.com/UFFeScience/akoflow/internal/domain/workflow/activity"
	workflow_entity "github.com/UFFeScience/akoflow/internal/domain/workflow/definition"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/activity_repository"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/resource_repository"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/workflow_repository"
)

type activityRepositoryFake struct {
	activity_repository.IActivityRepository
	activity        workflow_activity_entity.WorkflowActivities
	findErr         error
	assignment      workflow_activity_entity.ActivitySchedule
	assignmentErr   error
	findCalls       int
	assignmentCalls int
}

func (f *activityRepositoryFake) Find(id int) (workflow_activity_entity.WorkflowActivities, error) {
	f.findCalls++
	return f.activity, f.findErr
}

func (f *activityRepositoryFake) GetActivityScheduleByActivityId(id int) (workflow_activity_entity.ActivitySchedule, error) {
	f.assignmentCalls++
	return f.assignment, f.assignmentErr
}

type workflowRepositoryFake struct {
	workflow_repository.IWorkflowRepository
	workflow workflow_entity.Workflow
	err      error
	calls    int
}

func (f *workflowRepositoryFake) Find(id int) (workflow_entity.Workflow, error) {
	f.calls++
	return f.workflow, f.err
}

type resourceRepositoryFake struct {
	resource_repository.IRepository
	resource *domain.Resource
	err      error
	calls    int
}

func (f *resourceRepositoryFake) FindByID(id string) (*domain.Resource, error) {
	f.calls++
	return f.resource, f.err
}

type runtimeFake struct {
	accepted   bool
	calls      int
	workflowID int
	activityID int
}

func (f *runtimeFake) ApplyJob(workflowID, activityID int) bool {
	f.calls++
	f.workflowID = workflowID
	f.activityID = activityID
	return f.accepted
}

type fixture struct {
	activities *activityRepositoryFake
	workflows  *workflowRepositoryFake
	resources  *resourceRepositoryFake
	runtime    *runtimeFake
	resolvedID string
	service    *RunActivityInClusterService
}

func newFixture() *fixture {
	f := &fixture{
		activities: &activityRepositoryFake{
			activity:   workflow_activity_entity.WorkflowActivities{Id: 42, WorkflowId: 7},
			assignment: workflow_activity_entity.ActivitySchedule{ActivityID: 42, ResourceID: "node-1"},
		},
		workflows: &workflowRepositoryFake{workflow: workflow_entity.Workflow{Id: 7}},
		resources: &resourceRepositoryFake{resource: &domain.Resource{ID: "node-1", RuntimeID: "k8s://cluster-a"}},
		runtime:   &runtimeFake{accepted: true},
	}
	f.service = NewWithDependencies(f.workflows, f.activities, f.resources, func(runtimeID string) RuntimeExecutor {
		f.resolvedID = runtimeID
		return f.runtime
	})
	return f
}

func TestRunDispatchesScheduledActivityToAssignedRuntime(t *testing.T) {
	f := newFixture()

	if err := f.service.Run(42); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if f.resolvedID != "k8s://cluster-a" {
		t.Fatalf("resolved runtime = %q", f.resolvedID)
	}
	if f.runtime.calls != 1 || f.runtime.workflowID != 7 || f.runtime.activityID != 42 {
		t.Fatalf("ApplyJob calls=%d workflow=%d activity=%d", f.runtime.calls, f.runtime.workflowID, f.runtime.activityID)
	}
}

func TestRunStopsAtEachInvalidBoundary(t *testing.T) {
	tests := []struct {
		name       string
		configure  func(*fixture)
		wantError  string
		wantWF     int
		wantAssign int
		wantRes    int
	}{
		{"activity missing", func(f *fixture) { f.activities.findErr = errors.New("missing") }, "find activity 42", 0, 0, 0},
		{"workflow missing", func(f *fixture) { f.workflows.err = errors.New("missing") }, "find workflow 7", 1, 0, 0},
		{"assignment lookup fails", func(f *fixture) { f.activities.assignmentErr = errors.New("db down") }, "find resource assignment", 1, 1, 0},
		{"assignment is empty", func(f *fixture) { f.activities.assignment.ResourceID = "" }, "has no resource assignment", 1, 1, 0},
		{"resource lookup fails", func(f *fixture) { f.resources.err = errors.New("db down") }, "find resource node-1", 1, 1, 1},
		{"resource missing", func(f *fixture) { f.resources.resource = nil }, "resource node-1 not found", 1, 1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture()
			tt.configure(f)
			err := f.service.Run(42)
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("Run() error = %v, want containing %q", err, tt.wantError)
			}
			if f.workflows.calls != tt.wantWF || f.activities.assignmentCalls != tt.wantAssign || f.resources.calls != tt.wantRes {
				t.Fatalf("calls workflow=%d assignment=%d resource=%d", f.workflows.calls, f.activities.assignmentCalls, f.resources.calls)
			}
			if f.runtime.calls != 0 {
				t.Fatalf("runtime called %d times after failure", f.runtime.calls)
			}
		})
	}
}

func TestRunHandlesUnavailableAndRejectingRuntime(t *testing.T) {
	t.Run("runtime unavailable", func(t *testing.T) {
		f := newFixture()
		f.service.runtimeResolver = func(string) RuntimeExecutor { return nil }
		err := f.service.Run(42)
		if err == nil || !strings.Contains(err.Error(), "runtime k8s://cluster-a not found") {
			t.Fatalf("Run() error = %v", err)
		}
	})

	t.Run("runtime rejects job", func(t *testing.T) {
		f := newFixture()
		f.runtime.accepted = false
		err := f.service.Run(42)
		if err == nil || !strings.Contains(err.Error(), "rejected activity 42") {
			t.Fatalf("Run() error = %v", err)
		}
		if f.runtime.calls != 1 {
			t.Fatalf("ApplyJob calls = %d", f.runtime.calls)
		}
	})
}
