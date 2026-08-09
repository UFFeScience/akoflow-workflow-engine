package workflow_engine_api_handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/UFFeScience/akoflow/internal/domain"
	executionservice "github.com/UFFeScience/akoflow/internal/execution/simulation"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/environment_repository"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/execution_run_repository"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/schedule_plan_repository"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/workflow_definition_repository"
	"github.com/UFFeScience/akoflow/internal/infrastructure/plugins/planning"
)

type PlanValidator interface {
	Validate(domain.SchedulePlan, domain.WorkflowVersion, []domain.Resource) error
}

type Simulator interface {
	Execute(context.Context, executionservice.Request) (domain.ExecutionTrace, error)
}

type Dependencies struct {
	Environments environment_repository.IRepository
	Workflows    workflow_definition_repository.IRepository
	Plans        schedule_plan_repository.IRepository
	Runs         execution_run_repository.IRepository
	Validator    PlanValidator
	Simulator    Simulator
}

type Handler struct {
	environments environment_repository.IRepository
	workflows    workflow_definition_repository.IRepository
	plans        schedule_plan_repository.IRepository
	runs         execution_run_repository.IRepository
	validator    PlanValidator
	simulator    Simulator
}

func New() *Handler {
	return NewWithDependencies(Dependencies{
		Environments: environment_repository.New(), Workflows: workflow_definition_repository.New(),
		Plans: schedule_plan_repository.New(), Runs: execution_run_repository.New(),
		Validator: planning.NewValidatePlanService(), Simulator: executionservice.NewSimulationExecutor(),
	})
}

func NewWithDependencies(dependencies Dependencies) *Handler {
	return &Handler{
		environments: dependencies.Environments, workflows: dependencies.Workflows,
		plans: dependencies.Plans, runs: dependencies.Runs,
		validator: dependencies.Validator, simulator: dependencies.Simulator,
	}
}

func (h *Handler) CreateEnvironment(w http.ResponseWriter, r *http.Request) {
	var definition environment_repository.Definition
	if !decode(w, r, &definition) {
		return
	}
	if err := h.environments.Create(definition); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusCreated, definition)
}

func (h *Handler) CreateWorkflow(w http.ResponseWriter, r *http.Request) {
	var definition workflow_definition_repository.Definition
	if !decode(w, r, &definition) {
		return
	}
	if err := h.workflows.Create(definition); err != nil {
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
	if err := h.plans.Save(request.Plan); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusCreated, request.Plan)
}

func (h *Handler) GetPlan(w http.ResponseWriter, r *http.Request) {
	plan, err := h.plans.Find(r.PathValue("planId"))
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

func (h *Handler) Simulate(w http.ResponseWriter, r *http.Request) {
	var request executionservice.Request
	if !decode(w, r, &request) {
		return
	}
	request.Run.Mode = domain.ExecutionModeSimulation
	request.Run.SchedulePlanID = request.Plan.ID
	request.Run.Status = domain.ExecutionRunRunning
	if err := h.runs.Create(request.Run); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err)
		return
	}
	trace, err := h.simulator.Execute(r.Context(), request)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err)
		return
	}
	if err := h.runs.Complete(trace); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusCreated, trace)
}

func decode(w http.ResponseWriter, r *http.Request, target any) bool {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return false
	}
	return true
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
