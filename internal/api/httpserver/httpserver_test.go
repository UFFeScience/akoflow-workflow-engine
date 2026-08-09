package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthCheck(t *testing.T) {
	recorder := httptest.NewRecorder()
	HealthCheck(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
	if recorder.Code != http.StatusOK || recorder.Body.String() != "ok" {
		t.Fatalf("unexpected response: %d %q", recorder.Code, recorder.Body.String())
	}
}

func TestAllowCORS(t *testing.T) {
	nextCalls := 0
	handler := AllowCORS(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { nextCalls++; w.WriteHeader(http.StatusCreated) }))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
	if recorder.Code != http.StatusCreated || nextCalls != 1 || recorder.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatal("CORS GET failed")
	}
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodOptions, "/", nil))
	if recorder.Code != http.StatusOK || nextCalls != 1 {
		t.Fatal("OPTIONS must short-circuit")
	}
}
