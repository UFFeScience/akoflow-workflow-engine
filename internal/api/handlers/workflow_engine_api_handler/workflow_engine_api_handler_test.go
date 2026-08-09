package workflow_engine_api_handler

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/UFFeScience/akoflow/internal/domain"
	executionservice "github.com/UFFeScience/akoflow/internal/execution/simulation"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/environment_repository"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/workflow_definition_repository"
	"github.com/stretchr/testify/require"
)

type environmentRepositoryStub struct{ err error }

func (s environmentRepositoryStub) Create(environment_repository.Definition) error { return s.err }

type workflowRepositoryStub struct{ err error }

func (s workflowRepositoryStub) Create(workflow_definition_repository.Definition) error { return s.err }
func (s workflowRepositoryStub) FindVersion(string) (*domain.WorkflowVersion, error)    { return nil, nil }

type planRepositoryStub struct {
	plan *domain.SchedulePlan
	err  error
}

func (s planRepositoryStub) Save(domain.SchedulePlan) error            { return s.err }
func (s planRepositoryStub) Find(string) (*domain.SchedulePlan, error) { return s.plan, s.err }

type runRepositoryStub struct{ err error }

func (s runRepositoryStub) Create(domain.ExecutionRun) error     { return s.err }
func (s runRepositoryStub) Complete(domain.ExecutionTrace) error { return s.err }

type validatorStub struct{ err error }

func (s validatorStub) Validate(domain.SchedulePlan, domain.WorkflowVersion, []domain.Resource) error {
	return s.err
}

type simulatorStub struct {
	trace domain.ExecutionTrace
	err   error
}

func (s simulatorStub) Execute(context.Context, executionservice.Request) (domain.ExecutionTrace, error) {
	return s.trace, s.err
}

func newTestHandler() *Handler {
	return NewWithDependencies(Dependencies{
		Environments: environmentRepositoryStub{}, Workflows: workflowRepositoryStub{},
		Plans: planRepositoryStub{}, Runs: runRepositoryStub{}, Validator: validatorStub{},
		Simulator: simulatorStub{trace: domain.ExecutionTrace{RunID: "run"}},
	})
}

func TestCreateEnvironment(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"environment":{"id":"env"},"version":{"id":"env-1"}}`))
	newTestHandler().CreateEnvironment(recorder, request)
	require.Equal(t, http.StatusCreated, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"id":"env"`)
}

func TestCreateWorkflowRejectsUnknownJSONField(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"unknown":true}`))
	newTestHandler().CreateWorkflow(recorder, request)
	require.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestCreatePlanReturnsValidationError(t *testing.T) {
	handler := newTestHandler()
	handler.validator = validatorStub{err: errors.New("invalid plan")}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"plan":{},"workflow":{},"resources":[]}`))
	handler.CreatePlan(recorder, request)
	require.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
	require.Contains(t, recorder.Body.String(), "invalid plan")
}

func TestGetPlanReturnsPlanAndNotFound(t *testing.T) {
	plan := &domain.SchedulePlan{ID: "plan"}
	handler := newTestHandler()
	handler.plans = planRepositoryStub{plan: plan}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/plans/plan", nil)
	request.SetPathValue("planId", "plan")
	handler.GetPlan(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code)

	handler.plans = planRepositoryStub{}
	recorder = httptest.NewRecorder()
	handler.GetPlan(recorder, request)
	require.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestSimulatePersistsAndReturnsTrace(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"run":{"id":"run"},"plan":{"id":"plan"},"workflow":{},"resources":[],"networkLinks":[],"activityProfiles":[]}`))
	newTestHandler().Simulate(recorder, request)
	require.Equal(t, http.StatusCreated, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"runId":"run"`)
}
