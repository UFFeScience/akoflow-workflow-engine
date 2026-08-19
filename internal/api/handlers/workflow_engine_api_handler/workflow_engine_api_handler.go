package workflow_engine_api_handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"time"

	apirequests "github.com/UFFeScience/akoflow/internal/api/requests"
	"github.com/UFFeScience/akoflow/internal/application/ports"
	"github.com/UFFeScience/akoflow/internal/controlplane/eventloop"
	"github.com/UFFeScience/akoflow/internal/domain"
	domainevents "github.com/UFFeScience/akoflow/internal/domain/events"
	domaininstance "github.com/UFFeScience/akoflow/internal/domain/instance"
	domainqueue "github.com/UFFeScience/akoflow/internal/domain/queue"
	"gopkg.in/yaml.v3"
)

const maxRequestBodyBytes = 32 << 20

type ExecutionQuery interface {
	FindRun(context.Context, string) (*domain.ExecutionRun, error)
	ListRuns(context.Context) ([]domain.ExecutionRun, error)
	ListTasks(context.Context, string) ([]domain.TaskExecution, error)
	ListHandles(context.Context, string) ([]domain.ActivityHandle, error)
	ListEvents(context.Context, string) ([]domainevents.Event, error)
}

type Dependencies struct {
	Environments ports.EnvironmentCatalog
	Workflows    ports.WorkflowStore
	Plans        ports.PlanStore
	Events       ports.EventPublisher
	Validator    ports.PlanValidator
	Executions   ExecutionQuery
	Topologies   ports.NetworkTopologyStore
	Scopes       ports.ExecutionScopeStore
	Data         ports.DataCatalog
	Resources    ports.ResourceInventory
	Instance     ports.InstanceStore
}

type Handler struct {
	environments ports.EnvironmentCatalog
	workflows    ports.WorkflowStore
	plans        ports.PlanStore
	events       ports.EventPublisher
	validator    ports.PlanValidator
	executions   ExecutionQuery
	topologies   ports.NetworkTopologyStore
	scopes       ports.ExecutionScopeStore
	data         ports.DataCatalog
	resources    ports.ResourceInventory
	instance     ports.InstanceStore
}

func New(dependencies Dependencies) (*Handler, error) {
	if missingDependencies(dependencies) {
		return nil, fmt.Errorf("workflow API dependencies are required")
	}
	return &Handler{
		environments: dependencies.Environments, workflows: dependencies.Workflows,
		plans: dependencies.Plans, events: dependencies.Events,
		validator: dependencies.Validator, executions: dependencies.Executions,
		topologies: dependencies.Topologies,
		scopes:     dependencies.Scopes,
		data:       dependencies.Data,
		resources:  dependencies.Resources,
		instance:   dependencies.Instance,
	}, nil
}

func missingDependencies(dependencies Dependencies) bool {
	return dependencies.Environments == nil || dependencies.Workflows == nil ||
		dependencies.Plans == nil || dependencies.Events == nil ||
		dependencies.Validator == nil || dependencies.Executions == nil ||
		dependencies.Topologies == nil || dependencies.Scopes == nil || dependencies.Resources == nil ||
		dependencies.Instance == nil
}

func (h *Handler) CreateExecutionScope(w http.ResponseWriter, r *http.Request) {
	var scope domain.ExecutionScope
	if !decode(w, r, &scope) {
		return
	}
	if err := h.scopes.CreateScope(r.Context(), scope); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusCreated, scope)
}

func (h *Handler) ListExecutionScopes(w http.ResponseWriter, r *http.Request) {
	items, err := h.scopes.ListScopes(r.Context())
	writeList(w, items, err)
}

func (h *Handler) GetExecutionScope(w http.ResponseWriter, r *http.Request) {
	item, err := h.scopes.FindScope(r.Context(), r.PathValue("scopeId"))
	writeItem(w, item, err)
}

func (h *Handler) GetInstance(w http.ResponseWriter, r *http.Request) {
	value, err := h.instance.Find(r.Context())
	writeItem(w, value, err)
}

func (h *Handler) SaveInstance(w http.ResponseWriter, r *http.Request) {
	var value domaininstance.Instance
	if !decode(w, r, &value) {
		return
	}
	if strings.TrimSpace(value.ID) == "" || strings.TrimSpace(value.Name) == "" {
		writeError(w, http.StatusUnprocessableEntity, fmt.Errorf("instance id and name are required"))
		return
	}
	if err := h.instance.Save(r.Context(), value); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err)
		return
	}
	stored, err := h.instance.Find(r.Context())
	writeItem(w, stored, err)
}

func (h *Handler) ListEnvironments(w http.ResponseWriter, r *http.Request) {
	items, err := h.environments.List(r.Context())
	writeList(w, items, err)
}

func (h *Handler) GetEnvironment(w http.ResponseWriter, r *http.Request) {
	item, err := h.environments.Find(r.Context(), r.PathValue("environmentId"))
	writeItem(w, item, err)
}

func (h *Handler) ListResources(w http.ResponseWriter, r *http.Request) {
	items, err := h.resources.List(r.Context())
	writeList(w, items, err)
}

func (h *Handler) GetResource(w http.ResponseWriter, r *http.Request) {
	item, err := h.resources.FindByID(r.Context(), r.PathValue("resourceId"))
	writeItem(w, item, err)
}

func (h *Handler) GetResourceSnapshot(w http.ResponseWriter, r *http.Request) {
	item, err := h.resources.LatestSnapshot(r.Context(), r.PathValue("resourceId"))
	writeItem(w, item, err)
}

func (h *Handler) CreateResource(w http.ResponseWriter, r *http.Request) {
	var item domain.Resource
	if !decode(w, r, &item) {
		return
	}
	if err := h.resources.Upsert(r.Context(), item); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (h *Handler) ListNetworkTopologies(w http.ResponseWriter, r *http.Request) {
	items, err := h.topologies.List(r.Context())
	writeList(w, items, err)
}

func (h *Handler) ListWorkflows(w http.ResponseWriter, r *http.Request) {
	items, err := h.workflows.List(r.Context())
	writeList(w, items, err)
}

func (h *Handler) GetWorkflow(w http.ResponseWriter, r *http.Request) {
	item, err := h.workflows.Find(r.Context(), r.PathValue("workflowId"))
	writeItem(w, item, err)
}

func (h *Handler) ListPlans(w http.ResponseWriter, r *http.Request) {
	items, err := h.plans.List(r.Context())
	writeList(w, items, err)
}

func (h *Handler) ListExecutions(w http.ResponseWriter, r *http.Request) {
	runs, err := h.executions.ListRuns(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	items := make([]map[string]any, 0, len(runs))
	for _, run := range runs {
		items = append(items, map[string]any{"run": run})
	}
	writeJSON(w, http.StatusOK, items)
}

func writeList(w http.ResponseWriter, value any, err error) {
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func writeItem(w http.ResponseWriter, value any, err error) {
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if isNil(value) {
		writeError(w, http.StatusNotFound, nil)
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func isNil(value any) bool {
	if value == nil {
		return true
	}
	reflectValue := reflect.ValueOf(value)
	return (reflectValue.Kind() == reflect.Ptr || reflectValue.Kind() == reflect.Interface) && reflectValue.IsNil()
}

func (h *Handler) CreateNetworkTopology(w http.ResponseWriter, r *http.Request) {
	var topology domain.NetworkTopology
	if !decode(w, r, &topology) {
		return
	}
	if err := h.topologies.Create(r.Context(), topology); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusCreated, topology)
}

func (h *Handler) GetNetworkTopology(w http.ResponseWriter, r *http.Request) {
	topology, err := h.topologies.Find(r.Context(), r.PathValue("topologyId"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if topology == nil {
		writeError(w, http.StatusNotFound, nil)
		return
	}
	writeJSON(w, http.StatusOK, topology)
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
	handles, err := h.executions.ListHandles(r.Context(), run.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	events, err := h.executions.ListEvents(r.Context(), run.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	response := map[string]any{
		"run": run, "activities": tasks, "handles": handles, "events": events,
	}
	if h.data != nil {
		instances, dataErr := h.data.ListInstances(r.Context(), run.ID)
		if dataErr != nil {
			writeError(w, http.StatusInternalServerError, dataErr)
			return
		}
		locations, locationErr := h.data.ListLocations(r.Context(), run.ID)
		if locationErr != nil {
			writeError(w, http.StatusInternalServerError, locationErr)
			return
		}
		response["dataObjects"], response["dataLocations"] = instances, locations
	}
	writeJSON(w, http.StatusOK, response)
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
	Plan            domain.SchedulePlan    `json:"plan"`
	Workflow        domain.WorkflowVersion `json:"workflow"`
	Resources       []domain.Resource      `json:"resources"`
	ExecutionScope  domain.ExecutionScope  `json:"executionScope"`
	NetworkTopology domain.NetworkTopology `json:"networkTopology"`
}

func (h *Handler) CreatePlan(w http.ResponseWriter, r *http.Request) {
	var request CreatePlanRequest
	if !decode(w, r, &request) {
		return
	}
	if request.Plan.NetworkTopologyID == "" {
		request.Plan.NetworkTopologyID = request.NetworkTopology.ID
	}
	if err := h.validator.Validate(request.Plan, request.Workflow, request.Resources, request.ExecutionScope, request.NetworkTopology); err != nil {
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
	contentType := normalizedContentType(r.Header.Get("Content-Type"))
	if contentType != "application/json" && !isYAML(contentType) {
		writeError(w, http.StatusUnsupportedMediaType, fmt.Errorf("request Content-Type must be application/json or application/yaml"))
		return false
	}
	payload, err := io.ReadAll(io.LimitReader(r.Body, maxRequestBodyBytes))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return false
	}
	if isYAML(contentType) {
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
	return contentType == "application/yaml" || contentType == "application/x-yaml" || contentType == "text/yaml"
}

func normalizedContentType(contentType string) string {
	return strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
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
