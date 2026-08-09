package planning

import (
	"context"
	"errors"
	"testing"

	"github.com/UFFeScience/akoflow/internal/domain"
	"github.com/stretchr/testify/require"
)

type pluginStub struct {
	plan domain.SchedulePlan
	err  error
}

func (p pluginStub) Plan(context.Context, domain.PlanningRequest) (domain.SchedulePlan, error) {
	return p.plan, p.err
}

func TestPluginSourceMarksPlanOrigin(t *testing.T) {
	plan, err := (PluginSource{Plugin: pluginStub{plan: domain.SchedulePlan{ID: "plan"}}}).Build(context.Background(), domain.PlanningRequest{})
	require.NoError(t, err)
	require.Equal(t, domain.PlanningSourcePlugin, plan.Source)
}

func TestPluginSourcePropagatesErrorWithoutChangingResult(t *testing.T) {
	expected := errors.New("planner unavailable")
	_, err := (PluginSource{Plugin: pluginStub{err: expected}}).Build(context.Background(), domain.PlanningRequest{})
	require.ErrorIs(t, err, expected)
}

func TestImportedSourceMarksPlanOrigin(t *testing.T) {
	plan, err := (ImportedSource{Plan: domain.SchedulePlan{ID: "plan"}}).Build(context.Background(), domain.PlanningRequest{})
	require.NoError(t, err)
	require.Equal(t, domain.PlanningSourceImported, plan.Source)
}

func TestValidatePlanAcceptsCompleteCompatiblePlan(t *testing.T) {
	workflow, resources, plan := validPlanFixture()
	require.NoError(t, NewValidatePlanService().Validate(plan, workflow, resources))
}

func TestValidatePlanRejectsInvalidCases(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*domain.WorkflowVersion, *[]domain.Resource, *domain.SchedulePlan)
		want   string
	}{
		{"workflow mismatch", func(_ *domain.WorkflowVersion, _ *[]domain.Resource, p *domain.SchedulePlan) {
			p.WorkflowVersionID = "other"
		}, "does not match"},
		{"unknown activity", func(_ *domain.WorkflowVersion, _ *[]domain.Resource, p *domain.SchedulePlan) {
			p.Assignments[0].ActivityID = "missing"
		}, "unknown activity"},
		{"duplicate activity", func(_ *domain.WorkflowVersion, _ *[]domain.Resource, p *domain.SchedulePlan) {
			p.Assignments[1].ActivityID = "a"
		}, "more than one assignment"},
		{"unknown resource", func(_ *domain.WorkflowVersion, _ *[]domain.Resource, p *domain.SchedulePlan) {
			p.Assignments[0].ResourceID = "missing"
		}, "unknown resource"},
		{"wrong environment", func(_ *domain.WorkflowVersion, r *[]domain.Resource, _ *domain.SchedulePlan) {
			(*r)[0].EnvironmentVersionID = "other"
		}, "another environment"},
		{"not schedulable", func(_ *domain.WorkflowVersion, r *[]domain.Resource, _ *domain.SchedulePlan) {
			(*r)[0].Schedulable = false
		}, "not schedulable"},
		{"insufficient cpu", func(w *domain.WorkflowVersion, _ *[]domain.Resource, _ *domain.SchedulePlan) {
			w.Activities[0].CPURequired = 3
		}, "lacks CPU"},
		{"insufficient memory", func(w *domain.WorkflowVersion, _ *[]domain.Resource, _ *domain.SchedulePlan) {
			w.Activities[0].MemoryRequiredBytes = 3
		}, "lacks memory"},
		{"negative interval", func(_ *domain.WorkflowVersion, _ *[]domain.Resource, p *domain.SchedulePlan) {
			p.Assignments[0].PredictedStartAt = 2
			p.Assignments[0].PredictedFinishAt = 1
		}, "finishes before"},
		{"incomplete", func(w *domain.WorkflowVersion, _ *[]domain.Resource, _ *domain.SchedulePlan) {
			w.Activities = append(w.Activities, domain.Activity{ID: "c"})
		}, "covers 2 of 3"},
		{"self dependency", func(w *domain.WorkflowVersion, _ *[]domain.Resource, _ *domain.SchedulePlan) {
			w.Dependencies = []domain.ActivityDependency{{ActivityID: "a", DependsOnActivityID: "a"}}
		}, "depends on itself"},
		{"cycle", func(w *domain.WorkflowVersion, _ *[]domain.Resource, _ *domain.SchedulePlan) {
			w.Dependencies = append(w.Dependencies, domain.ActivityDependency{ActivityID: "a", DependsOnActivityID: "b"})
		}, "contain a cycle"},
		{"invalid lane order", func(_ *domain.WorkflowVersion, _ *[]domain.Resource, p *domain.SchedulePlan) {
			p.Assignments[0].ResourceID = "r2"
			p.Assignments[0].OrderOnResource = 2
			p.Assignments[1].OrderOnResource = 1
		}, "before predecessor"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			workflow, resources, plan := validPlanFixture()
			test.mutate(&workflow, &resources, &plan)
			err := NewValidatePlanService().Validate(plan, workflow, resources)
			require.ErrorContains(t, err, test.want)
		})
	}
}

func validPlanFixture() (domain.WorkflowVersion, []domain.Resource, domain.SchedulePlan) {
	workflow := domain.WorkflowVersion{
		ID: "wf", Activities: []domain.Activity{
			{ID: "a", CPURequired: 1, MemoryRequiredBytes: 1},
			{ID: "b", CPURequired: 1, MemoryRequiredBytes: 1},
		}, Dependencies: []domain.ActivityDependency{{ActivityID: "b", DependsOnActivityID: "a"}},
	}
	resources := []domain.Resource{
		{ID: "r1", EnvironmentVersionID: "env", CPUCapacity: 2, MemoryBytes: 2, Schedulable: true},
		{ID: "r2", EnvironmentVersionID: "env", CPUCapacity: 2, MemoryBytes: 2, Schedulable: true},
	}
	plan := domain.SchedulePlan{WorkflowVersionID: "wf", EnvironmentVersionID: "env", Assignments: []domain.PlanAssignment{
		{ID: "aa", ActivityID: "a", ResourceID: "r1", PredictedFinishAt: 1},
		{ID: "ab", ActivityID: "b", ResourceID: "r2", PredictedStartAt: 1, PredictedFinishAt: 2},
	}}
	return workflow, resources, plan
}
