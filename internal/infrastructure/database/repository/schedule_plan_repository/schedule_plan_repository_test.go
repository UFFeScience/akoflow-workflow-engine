package schedule_plan_repository

import (
	"path/filepath"
	"testing"

	"github.com/UFFeScience/akoflow/internal/domain"
)

func setup(t *testing.T) IRepository {
	t.Helper()
	t.Setenv("AKOFLOW_DATABASE_PATH", filepath.Join(t.TempDir(), "db.sqlite"))
	return New()
}

func TestSchedulePlanSaveAndFind(t *testing.T) {
	repository := setup(t)
	plan := domain.SchedulePlan{ID: "p1", WorkflowVersionID: "w1", EnvironmentVersionID: "e1", Source: domain.PlanningSourcePlugin, Algorithm: "prism", AlgorithmVersion: "1", Objective: "time", DeadlineSeconds: 10, Budget: 2, Predicted: domain.PredictedMetrics{MakespanSeconds: 8, Cost: 1, Feasible: true}, Assignments: []domain.PlanAssignment{{ID: "a1", ActivityID: "task", ResourceID: "r1", CoreID: "c", SlotID: "s", OrderOnResource: 1, Priority: 2, PredictedReadyAt: 1, PredictedStartAt: 2, PredictedFinishAt: 3, PredictedRuntimeSeconds: 1, PredictedTransferSeconds: .5, PredictedCost: .2, Metadata: map[string]any{"reason": "best"}}}}
	if err := repository.Save(plan); err != nil {
		t.Fatal(err)
	}
	got, err := repository.Find("p1")
	if err != nil || got == nil {
		t.Fatalf("find failed: %+v %v", got, err)
	}
	if got.Algorithm != "prism" || got.Source != domain.PlanningSourcePlugin || len(got.Assignments) != 1 || got.Assignments[0].PlanID != "p1" || got.Assignments[0].Metadata["reason"] != "best" {
		t.Fatalf("unexpected plan: %+v", got)
	}
	missing, err := repository.Find("missing")
	if err != nil || missing != nil {
		t.Fatal("missing plan must return nil")
	}
}

func TestSchedulePlanSaveErrors(t *testing.T) {
	repository := setup(t)
	bad := domain.SchedulePlan{ID: "bad", Source: domain.PlanningSourcePlugin, Assignments: []domain.PlanAssignment{{ID: "a", Metadata: map[string]any{"channel": make(chan int)}}}}
	if err := repository.Save(bad); err == nil {
		t.Fatal("unserializable assignment metadata must fail")
	}
	valid := domain.SchedulePlan{ID: "duplicate", Source: domain.PlanningSourcePlugin}
	if err := repository.Save(valid); err != nil {
		t.Fatal(err)
	}
	if err := repository.Save(valid); err == nil {
		t.Fatal("duplicate plan must fail")
	}
}
