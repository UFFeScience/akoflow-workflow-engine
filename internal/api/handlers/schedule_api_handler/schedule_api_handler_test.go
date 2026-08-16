package schedule_api_handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/UFFeScience/akoflow/internal/api/requests"
	"github.com/UFFeScience/akoflow/internal/infrastructure/config"
)

type listFake struct {
	values []types_api.ApiScheduleType
	err    error
}

func (f listFake) ListAllSchedules() ([]types_api.ApiScheduleType, error) { return f.values, f.err }

type createFake struct {
	value types_api.ApiScheduleType
	err   error
}

func (f createFake) CreateSchedule(string, string, string) (types_api.ApiScheduleType, error) {
	return f.value, f.err
}

type getFake struct {
	value *types_api.ApiScheduleType
	err   error
}

func (f getFake) GetScheduleByName(string) (*types_api.ApiScheduleType, error) { return f.value, f.err }
func configureHTTP() {
	container := config.MakeAppContainer()
	container.HttpHelper.ReadJson = func(r *http.Request, v interface{}) error { return json.NewDecoder(r.Body).Decode(v) }
	container.HttpHelper.GetUrlParam = func(*http.Request, string) string { return "prism" }
	config.SetAppContainer(container)
}
func TestScheduleHandlersSuccess(t *testing.T) {
	configureHTTP()
	value := types_api.ApiScheduleType{Name: "prism"}
	h := NewWithDependencies(listFake{values: []types_api.ApiScheduleType{value}}, createFake{value: value}, getFake{value: &value})
	for _, tc := range []struct {
		call    func(http.ResponseWriter, *http.Request)
		request *http.Request
		code    int
	}{{h.ListAllSchedules, httptest.NewRequest(http.MethodGet, "/", nil), 200}, {h.CreateSchedule, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"prism","type":"go","code":"YQ=="}`)), 201}, {h.GetSchedule, httptest.NewRequest(http.MethodGet, "/", nil), 200}} {
		rec := httptest.NewRecorder()
		tc.call(rec, tc.request)
		if rec.Code != tc.code || !strings.Contains(rec.Body.String(), "prism") {
			t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
		}
	}
}
func TestScheduleHandlersValidationAndServiceErrors(t *testing.T) {
	configureHTTP()
	h := NewWithDependencies(listFake{err: errors.New("db")}, createFake{err: errors.New("create")}, getFake{err: errors.New("get")})
	rec := httptest.NewRecorder()
	h.ListAllSchedules(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != 500 {
		t.Fatal(rec.Code)
	}
	rec = httptest.NewRecorder()
	h.CreateSchedule(rec, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{`)))
	if rec.Code != 400 {
		t.Fatal(rec.Code)
	}
	rec = httptest.NewRecorder()
	h.CreateSchedule(rec, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":""}`)))
	if rec.Code != 400 {
		t.Fatal(rec.Code)
	}
	rec = httptest.NewRecorder()
	h.CreateSchedule(rec, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"a","type":"b","code":"c"}`)))
	if rec.Code != 500 {
		t.Fatal(rec.Code)
	}
	rec = httptest.NewRecorder()
	h.GetSchedule(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != 500 {
		t.Fatal(rec.Code)
	}
}
func TestNewInitializesServices(t *testing.T) {
	h := New()
	if h.listApiSchedulesService == nil || h.createApiScheduleService == nil || h.getApiScheduleService == nil {
		t.Fatal("incomplete")
	}
}
