package workflow_handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	workflow_entity "github.com/UFFeScience/akoflow/internal/domain/workflow/definition"
)

type creatorFake struct {
	id  int
	err error
}

func (f creatorFake) Create(workflow_entity.Workflow) (int, error) { return f.id, f.err }
func TestCreateWorkflowSuccessAndFailures(t *testing.T) {
	payload := `{"workflow":"bmFtZTogd2YKc3BlYzoKICBydW50aW1lOiBsb2NhbAo="}`
	rec := httptest.NewRecorder()
	NewWithDependencies(creatorFake{id: 7}).Create(rec, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(payload)))
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"workflow_id":7`) {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
	rec = httptest.NewRecorder()
	NewWithDependencies(creatorFake{}).Create(rec, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{`)))
	if rec.Code != 400 {
		t.Fatal(rec.Code)
	}
	rec = httptest.NewRecorder()
	NewWithDependencies(creatorFake{err: errors.New("db")}).Create(rec, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(payload)))
	if rec.Code != 500 {
		t.Fatal(rec.Code)
	}
}
func TestNewInitializesCreator(t *testing.T) {
	if New().create_workflow_service == nil {
		t.Fatal("missing creator")
	}
}
