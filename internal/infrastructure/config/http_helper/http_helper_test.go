package http_helper

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWriteJSONSuccessAndMarshalFailure(t *testing.T) {
	recorder := httptest.NewRecorder()
	WriteJson(recorder, map[string]string{"ok": "yes"})
	if !strings.Contains(recorder.Body.String(), `"data":{"ok":"yes"}`) || !strings.Contains(recorder.Body.String(), `"timestamp"`) {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
	recorder = httptest.NewRecorder()
	WriteJson(recorder, make(chan int))
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("got %d", recorder.Code)
	}
}

func TestReadJSON(t *testing.T) {
	var target struct {
		Name string `json:"name"`
	}
	if err := ReadJson(&http.Request{}, &target); err == nil {
		t.Fatal("nil body must fail")
	}
	request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"name":"test"}`))
	if err := ReadJson(request, &target); err != nil || target.Name != "test" {
		t.Fatalf("decode failed: %v", err)
	}
	request = httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"unknown":1}`))
	if err := ReadJson(request, &target); err == nil {
		t.Fatal("unknown field must fail")
	}
	request = httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{`))
	if err := ReadJson(request, &target); err == nil {
		t.Fatal("malformed JSON must fail")
	}
}

func TestURLPathParameters(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /items/{itemId}/", func(w http.ResponseWriter, r *http.Request) {
		if GetPatternFromRequest(r) == "" {
			t.Error("matched request must expose pattern")
		}
		if got := GetUrlPathParam(r, "itemId"); got != "42" {
			t.Errorf("got %q", got)
		}
		if got := GetUrlPathParam(r, "missing"); got != "" {
			t.Errorf("missing key: %q", got)
		}
	})
	mux.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/items/42/", nil))
	request := httptest.NewRequest(http.MethodGet, "/plain", nil)
	if GetPatternFromRequest(request) != "" || GetUrlPathParam(request, "id") != "" {
		t.Fatal("unmatched request must have no pattern")
	}
}
