package workflow_engine_api_handler

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/UFFeScience/akoflow/internal/domain"
	domainqueue "github.com/UFFeScience/akoflow/internal/domain/queue"
	"github.com/stretchr/testify/require"
)

type environmentRepositoryStub struct{ err error }

func (s environmentRepositoryStub) Create(context.Context, domain.EnvironmentDefinition) error {
	return s.err
}
func (s environmentRepositoryStub) UpdateStatus(context.Context, string, domain.EnvironmentStatus) error {
	return s.err
}
func (s environmentRepositoryStub) UpsertConnection(context.Context, domain.EnvironmentConnection) error {
	return s.err
}
func (s environmentRepositoryStub) ListConnections(context.Context, string) ([]domain.EnvironmentConnection, error) {
	return nil, s.err
}

type workflowRepositoryStub struct{ err error }

func (s workflowRepositoryStub) Create(context.Context, domain.WorkflowDefinition) error {
	return s.err
}
func (s workflowRepositoryStub) FindVersion(context.Context, string) (*domain.WorkflowVersion, error) {
	return nil, nil
}

type planRepositoryStub struct {
	plan *domain.SchedulePlan
	err  error
}

func (s planRepositoryStub) Save(context.Context, domain.SchedulePlan) error { return s.err }
func (s planRepositoryStub) Find(context.Context, string) (*domain.SchedulePlan, error) {
	return s.plan, s.err
}

type eventPublisherStub struct {
	jobs []domainqueue.Job
	err  error
}

func (s *eventPublisherStub) Publish(_ context.Context, job domainqueue.Job) (domainqueue.Job, error) {
	s.jobs = append(s.jobs, job)
	return job, s.err
}

type validatorStub struct{ err error }

func (s validatorStub) Validate(domain.SchedulePlan, domain.WorkflowVersion, []domain.Resource) error {
	return s.err
}

func newTestHandler() *Handler {
	handler, err := New(Dependencies{
		Environments: environmentRepositoryStub{}, Workflows: workflowRepositoryStub{},
		Plans: planRepositoryStub{}, Events: &eventPublisherStub{}, Validator: validatorStub{},
	})
	if err != nil {
		panic(err)
	}
	return handler
}

func TestNewRejectsIncompleteDependencies(t *testing.T) {
	if _, err := New(Dependencies{}); err == nil {
		t.Fatal("missing dependencies must fail")
	}
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

func TestCreateExecutionPublishesPersistentCommand(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"run":{"id":"run","mode":"simulation"},"plan":{"id":"plan"},"workflow":{"id":"workflow"},"resources":[],"networkLinks":[],"activityProfiles":[]}`))
	newTestHandler().CreateExecution(recorder, request)
	require.Equal(t, http.StatusAccepted, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"eventType":"execution.run.requested"`)
}
