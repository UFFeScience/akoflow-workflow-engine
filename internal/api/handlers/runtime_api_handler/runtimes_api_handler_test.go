package runtime_api_handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/UFFeScience/akoflow/internal/api/requests"
	"github.com/UFFeScience/akoflow/internal/infrastructure/config"
)

type runtimeListerFake struct {
	values []types_api.ApiRuntimeType
	err    error
}

func (f runtimeListerFake) ListAllRuntimes() ([]types_api.ApiRuntimeType, error) {
	return f.values, f.err
}
func TestListAllRuntimesSuccessAndError(t *testing.T) {
	config.SetAppContainer(config.MakeAppContainer())
	rec := httptest.NewRecorder()
	NewWithDependencies(runtimeListerFake{values: []types_api.ApiRuntimeType{{Name: "local"}}}).ListAllRuntimes(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "local") {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
	rec = httptest.NewRecorder()
	NewWithDependencies(runtimeListerFake{err: errors.New("db")}).ListAllRuntimes(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != 500 {
		t.Fatalf("code=%d", rec.Code)
	}
}
func TestNewInitializesLister(t *testing.T) {
	if New().listRuntimeApiService == nil {
		t.Fatal("missing lister")
	}
}
