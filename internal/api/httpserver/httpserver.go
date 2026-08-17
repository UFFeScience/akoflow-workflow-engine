package httpserver

import (
	"context"
	"errors"
	"github.com/UFFeScience/akoflow/internal/api/handlers/workflow_engine_api_handler"
	"github.com/UFFeScience/akoflow/internal/infrastructure/config/http_config"

	"net/http"
	"time"
)

func HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func NewMux(workflowEngine *workflow_engine_api_handler.Handler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", http_config.KernelHandler(HealthCheck))

	mux.HandleFunc("POST /akoflow-api/environments/", http_config.KernelHandler(workflowEngine.CreateEnvironment))
	mux.HandleFunc("POST /akoflow-api/network-topologies/", http_config.KernelHandler(workflowEngine.CreateNetworkTopology))
	mux.HandleFunc("GET /akoflow-api/network-topologies/{topologyId}/", http_config.KernelHandler(workflowEngine.GetNetworkTopology))
	mux.HandleFunc("POST /akoflow-api/workflow-definitions/", http_config.KernelHandler(workflowEngine.CreateWorkflow))
	mux.HandleFunc("POST /akoflow-api/schedule-plans/", http_config.KernelHandler(workflowEngine.CreatePlan))
	mux.HandleFunc("GET /akoflow-api/schedule-plans/{planId}/", http_config.KernelHandler(workflowEngine.GetPlan))
	mux.HandleFunc("POST /akoflow-api/execution-runs/", http_config.KernelHandler(workflowEngine.CreateExecution))
	mux.HandleFunc("GET /akoflow-api/execution-runs/{runId}/", http_config.KernelHandler(workflowEngine.GetExecution))
	return mux
}

func Serve(ctx context.Context, address string, workflowEngine *workflow_engine_api_handler.Handler) error {
	server := &http.Server{Addr: address, Handler: AllowCORS(NewMux(workflowEngine))}
	go shutdownWhenCanceled(ctx, server)
	err := server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func shutdownWhenCanceled(ctx context.Context, server *http.Server) {
	<-ctx.Done()
	shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownContext)
}
