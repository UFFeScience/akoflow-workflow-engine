package workflow_api_handler

import (
	"encoding/json"
	"errors"
	"github.com/UFFeScience/akoflow/internal/api/requests"
	"github.com/UFFeScience/akoflow/internal/application/services/provenance_graph_service"
	"github.com/UFFeScience/akoflow/internal/infrastructure/config"
	"net/http"
	"net/http/httptest"
	"testing"
)

type listFake struct{ err error }

func (f listFake) ListAllWorkflows() ([]types_api.ApiWorkflowType, error) {
	return []types_api.ApiWorkflowType{{Name: "wf"}}, f.err
}

type findFake struct{ err error }

func (f findFake) FindWorkflowById(int) (types_api.ApiWorkflowType, error) {
	return types_api.ApiWorkflowType{Name: "wf"}, f.err
}

type graphFake struct{ err error }

func (f graphFake) BuildGraph(int) (*provenance_graph_service.ProvenanceGraph, error) {
	return &provenance_graph_service.ProvenanceGraph{}, f.err
}
func installHTTPHelpers(t *testing.T) {
	old := config.App()
	t.Cleanup(func() { config.SetAppContainer(old) })
	old.HttpHelper.WriteJson = func(w http.ResponseWriter, v interface{}) { _ = json.NewEncoder(w).Encode(v) }
	old.HttpHelper.GetUrlParam = func(*http.Request, string) string { return "7" }
	config.SetAppContainer(old)
}
func TestWorkflowAPIHandlers(t *testing.T) {
	installHTTPHelpers(t)
	h := NewWithDependencies(listFake{}, findFake{}, graphFake{})
	for _, call := range []func(http.ResponseWriter, *http.Request){h.ListAllWorkflows, h.GetWorkflow, h.ListStorageFiles} {
		r := httptest.NewRecorder()
		call(r, httptest.NewRequest("GET", "/", nil))
		if r.Code != 200 {
			t.Fatalf("status %d", r.Code)
		}
	}
}
func TestWorkflowAPIErrors(t *testing.T) {
	installHTTPHelpers(t)
	req := httptest.NewRequest("GET", "/", nil)
	for _, h := range []*WorkflowApiHandler{NewWithDependencies(listFake{errors.New("db")}, findFake{}, graphFake{}), NewWithDependencies(listFake{}, findFake{errors.New("db")}, graphFake{}), NewWithDependencies(listFake{}, findFake{}, graphFake{errors.New("db")})} {
		r := httptest.NewRecorder()
		if h.listWorkflowApiService.(listFake).err != nil {
			h.ListAllWorkflows(r, req)
		} else if h.findWorkflowApiService.(findFake).err != nil {
			h.GetWorkflow(r, req)
		} else {
			h.ListStorageFiles(r, req)
		}
		if r.Code != 500 {
			t.Fatalf("status %d", r.Code)
		}
	}
	app := config.App()
	app.HttpHelper.GetUrlParam = func(*http.Request, string) string { return "bad" }
	config.SetAppContainer(app)
	r := httptest.NewRecorder()
	NewWithDependencies(listFake{}, findFake{}, graphFake{}).GetWorkflow(r, req)
	if r.Code != 400 {
		t.Fatalf("status %d", r.Code)
	}
}
