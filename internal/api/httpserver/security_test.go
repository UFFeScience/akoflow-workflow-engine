package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSecureAPIRequiresBearerToken(t *testing.T) {
	handler := SecureAPI(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}), SecurityOptions{BearerToken: "test-token"})
	request := httptest.NewRequest(http.MethodGet, "/akoflow-api/storages/x/entries/", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("got %d, want 401", response.Code)
	}
	request.Header.Set("Authorization", "Bearer test-token")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("got %d, want 204", response.Code)
	}
}
