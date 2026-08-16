package httpserver

import (
	"github.com/UFFeScience/akoflow/internal/api/handlers/workflow_engine_api_handler"
	"github.com/UFFeScience/akoflow/internal/infrastructure/config"
	"github.com/UFFeScience/akoflow/internal/infrastructure/config/http_config"

	"net/http"
)

func HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func StartServer() {
	workflowEngine := workflow_engine_api_handler.New()

	http.HandleFunc("GET /", http_config.KernelHandler(HealthCheck))

	http.HandleFunc("POST /akoflow-api/environments/", http_config.KernelHandler(workflowEngine.CreateEnvironment))
	http.HandleFunc("POST /akoflow-api/workflow-definitions/", http_config.KernelHandler(workflowEngine.CreateWorkflow))
	http.HandleFunc("POST /akoflow-api/schedule-plans/", http_config.KernelHandler(workflowEngine.CreatePlan))
	http.HandleFunc("GET /akoflow-api/schedule-plans/{planId}/", http_config.KernelHandler(workflowEngine.GetPlan))
	http.HandleFunc("POST /akoflow-api/execution-runs/", http_config.KernelHandler(workflowEngine.Simulate))

	handler := AllowCORS(http.DefaultServeMux)
	err := http.ListenAndServe(config.PORT_SERVER, handler)
	if err != nil {
		println("Error starting server", err)
		panic(err)
	}

}
