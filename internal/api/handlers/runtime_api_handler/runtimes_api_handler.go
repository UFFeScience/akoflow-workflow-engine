package runtime_api_handler

import (
	"net/http"

	"github.com/UFFeScience/akoflow/internal/infrastructure/config"
	// "github.com/UFFeScience/akoflow/internal/application/services/find_runtime_api_service"
	"github.com/UFFeScience/akoflow/internal/application/services/list_runtimes_api_service"
)

type RuntimeApiHandler struct {
	listRuntimeApiService *list_runtimes_api_service.ListRuntimesApiService
	// findRuntimeApiService *find_runtime_api_service.FindRuntimeApiService
}

func New() *RuntimeApiHandler {
	return &RuntimeApiHandler{
		listRuntimeApiService: list_runtimes_api_service.New(),
		// findRuntimeApiService: find_runtime_api_service.New(),
	}
}

func (h *RuntimeApiHandler) ListAllRuntimes(w http.ResponseWriter, r *http.Request) {

	runtimes, err := h.listRuntimeApiService.ListAllRuntimes()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	config.App().HttpHelper.WriteJson(w, runtimes)
}
