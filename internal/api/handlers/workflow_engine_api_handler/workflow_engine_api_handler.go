package workflow_engine_api_handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	apirequests "github.com/UFFeScience/akoflow/internal/api/requests"
	"github.com/UFFeScience/akoflow/internal/application/ports"
	"github.com/UFFeScience/akoflow/internal/controlplane/eventloop"
	"github.com/UFFeScience/akoflow/internal/domain"
	domainevents "github.com/UFFeScience/akoflow/internal/domain/events"
	domainqueue "github.com/UFFeScience/akoflow/internal/domain/queue"
	"gopkg.in/yaml.v3"
)

type PlanValidator interface {
	Validate(domain.SchedulePlan, domain.WorkflowVersion, []domain.Resource) error
}

type ExecutionQuery interface {
	FindRun(context.Context, string) (*domain.ExecutionRun, error)
	ListTasks(context.Context, string) ([]domain.TaskExecution, error)
	ListEvents(context.Context, string) ([]domainevents.Event, error)
}

type Dependencies struct {
	Environments ports.EnvironmentCatalog
	Workflows    ports.WorkflowStore
	Plans        ports.PlanStore
	Events       ports.EventPublisher
	Validator    PlanValidator
	Executions   ExecutionQuery
}

type Handler struct {
	environments ports.EnvironmentCatalog
	workflows    ports.WorkflowStore
	plans        ports.PlanStore
	events       ports.EventPublisher
	validator    PlanValidator
	executions   ExecutionQuery
}

func New(dependencies Dependencies) (*Handler, error) {
	if dependencies.Environments == nil || dependencies.Workflows == nil || dependencies.Plans == nil || dependencies.Events == nil || dependencies.Validator == nil || dependencies.Executions == nil {
		return nil, fmt.Errorf("workflow API dependencies are required")
	}
	return &Handler{
		environments: dependencies.Environments, workflows: dependencies.Workflows,
		plans: dependencies.Plans, events: dependencies.Events,
		validator: dependencies.Validator, executions: dependencies.Executions,
	}, nil
}

func (h *Handler) GetExecution(w http.ResponseWriter, r *http.Request) {
	run, err := h.executions.FindRun(r.Context(), r.PathValue("runId"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if run == nil {
		writeError(w, http.StatusNotFound, nil)
		return
	}
	tasks, err := h.executions.ListTasks(r.Context(), run.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	events, err := h.executions.ListEvents(r.Context(), run.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"run": run, "activities": tasks, "events": events})
}

func (h *Handler) CreateEnvironment(w http.ResponseWriter, r *http.Request) {
	var definition domain.EnvironmentDefinition
	if !decode(w, r, &definition) {
		return
	}
	if err := h.environments.Create(r.Context(), definition); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusCreated, definition)
}

func (h *Handler) CreateWorkflow(w http.ResponseWriter, r *http.Request) {
	var request apirequests.Workflow
	if !decode(w, r, &request) {
		return
	}
	definition, err := request.Domain()
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err)
		return
	}
	if err := h.workflows.Create(r.Context(), definition); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusCreated, definition)
}

type CreatePlanRequest struct {
	Plan      domain.SchedulePlan    `json:"plan"`
	Workflow  domain.WorkflowVersion `json:"workflow"`
	Resources []domain.Resource      `json:"resources"`
}

func (h *Handler) CreatePlan(w http.ResponseWriter, r *http.Request) {
	var request CreatePlanRequest
	if !decode(w, r, &request) {
		return
	}
	if err := h.validator.Validate(request.Plan, request.Workflow, request.Resources); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err)
		return
	}
	if err := h.plans.Save(r.Context(), request.Plan); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusCreated, request.Plan)
}

func (h *Handler) GetPlan(w http.ResponseWriter, r *http.Request) {
	plan, err := h.plans.Find(r.Context(), r.PathValue("planId"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if plan == nil {
		writeError(w, http.StatusNotFound, nil)
		return
	}
	writeJSON(w, http.StatusOK, plan)
}

func (h *Handler) CreateExecution(w http.ResponseWriter, r *http.Request) {
	var request ports.ExecutionRequest
	if !decode(w, r, &request) {
		return
	}
	request.Run.SchedulePlanID = request.Plan.ID
	payload, err := json.Marshal(request)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	job, err := domainqueue.New(domainqueue.CategoryExecution, eventloop.EventExecutionRunRequested, payload, time.Now().UTC())
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	job.AggregateType, job.AggregateID = "execution_run", request.Run.ID
	job.IdempotencyKey = "execution-run:" + request.Run.ID
	stored, err := h.events.Publish(r.Context(), job)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusAccepted, stored)
}

func decode(w http.ResponseWriter, r *http.Request, target any) bool {
	if !isYAML(r.Header.Get("Content-Type")) {
		writeError(w, http.StatusUnsupportedMediaType, fmt.Errorf("request Content-Type must be application/yaml"))
		return false
	}
	payload, err := io.ReadAll(io.LimitReader(r.Body, 10<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return false
	}
	var document any
	if err := yaml.Unmarshal(payload, &document); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("decode YAML: %w", err))
		return false
	}
	payload, err = json.Marshal(document)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("normalize YAML: %w", err))
		return false
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return false
	}
	return true
}

func isYAML(contentType string) bool {
	contentType = strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	return contentType == "application/yaml" || contentType == "application/x-yaml" || contentType == "text/yaml"
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, err error) {
	message := http.StatusText(status)
	if err != nil {
		message = err.Error()
	}
	writeJSON(w, status, map[string]string{"error": message})
}
