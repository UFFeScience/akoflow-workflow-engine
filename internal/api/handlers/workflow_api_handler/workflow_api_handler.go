package workflow_api_handler

import (
	"net/http"
	"strconv"

	"github.com/UFFeScience/akoflow/internal/api/requests"
	"github.com/UFFeScience/akoflow/internal/application/services/find_workflow_api_service"
	"github.com/UFFeScience/akoflow/internal/application/services/list_workflows_api_service"
	"github.com/UFFeScience/akoflow/internal/application/services/provenance_graph_service"
	"github.com/UFFeScience/akoflow/internal/infrastructure/config"
)

type WorkflowApiHandler struct {
	listWorkflowApiService interface {
		ListAllWorkflows() ([]types_api.ApiWorkflowType, error)
	}
	findWorkflowApiService interface {
		FindWorkflowById(int) (types_api.ApiWorkflowType, error)
	}
	provenanceGraphService interface {
		BuildGraph(int) (*provenance_graph_service.ProvenanceGraph, error)
	}
}

func New() *WorkflowApiHandler {
	return &WorkflowApiHandler{
		listWorkflowApiService: list_workflows_api_service.New(),
		findWorkflowApiService: find_workflow_api_service.New(),
		provenanceGraphService: provenance_graph_service.New(
			config.App().Repository.StoragesRepository,
			config.App().Repository.ActivityRepository,
		),
	}
}

func NewWithDependencies(list interface {
	ListAllWorkflows() ([]types_api.ApiWorkflowType, error)
}, find interface {
	FindWorkflowById(int) (types_api.ApiWorkflowType, error)
}, graph interface {
	BuildGraph(int) (*provenance_graph_service.ProvenanceGraph, error)
}) *WorkflowApiHandler {
	return &WorkflowApiHandler{listWorkflowApiService: list, findWorkflowApiService: find, provenanceGraphService: graph}
}

func (h *WorkflowApiHandler) ListAllWorkflows(w http.ResponseWriter, r *http.Request) {

	workflows, err := h.listWorkflowApiService.ListAllWorkflows()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	config.App().HttpHelper.WriteJson(w, workflows)
}

func (h *WorkflowApiHandler) GetWorkflow(w http.ResponseWriter, r *http.Request) {
	workflowIdStr := config.App().HttpHelper.GetUrlParam(r, "workflowId")
	workflowId, err := strconv.Atoi(workflowIdStr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	workflow, err := h.findWorkflowApiService.FindWorkflowById(workflowId)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	config.App().HttpHelper.WriteJson(w, workflow)

}

func (h *WorkflowApiHandler) ListStorageFiles(w http.ResponseWriter, r *http.Request) {
	workflowIdStr := config.App().HttpHelper.GetUrlParam(r, "workflowId")
	workflowId, _ := strconv.Atoi(workflowIdStr)

	graph, err := h.provenanceGraphService.BuildGraph(workflowId)
	if err != nil {
		http.Error(w, "Erro ao montar grafo de proveniência", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	config.App().HttpHelper.WriteJson(w, graph)
}
