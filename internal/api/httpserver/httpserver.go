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
	mux.HandleFunc("GET /akoflow-api/instance/", http_config.KernelHandler(workflowEngine.GetInstance))
	mux.HandleFunc("GET /akoflow-api/audit-events/", http_config.KernelHandler(workflowEngine.ListAuditEvents))
	mux.HandleFunc("GET /akoflow-api/console-commands/", http_config.KernelHandler(workflowEngine.ListConsoleCommands))
	mux.HandleFunc("POST /akoflow-api/console-commands/", http_config.KernelHandler(workflowEngine.ExecuteConsoleCommand))
	mux.HandleFunc("POST /akoflow-api/console-sessions/", http_config.KernelHandler(workflowEngine.OpenConsoleSession))
	mux.HandleFunc("GET /akoflow-api/console-sessions/", http_config.KernelHandler(workflowEngine.ListConsoleSessions))
	mux.HandleFunc("DELETE /akoflow-api/console-sessions/{sessionId}/", http_config.KernelHandler(workflowEngine.CloseConsoleSession))
	mux.HandleFunc("GET /akoflow-api/console-sessions/{sessionId}/log/", http_config.KernelHandler(workflowEngine.ExportConsoleSessionLog))
	mux.HandleFunc("GET /akoflow-api/console-sessions/{sessionId}/stream/", workflowEngine.StreamConsoleSession)
	mux.HandleFunc("GET /akoflow-api/ssh-keys/", http_config.KernelHandler(workflowEngine.ListSSHKeys))
	mux.HandleFunc("POST /akoflow-api/ssh-keys/", http_config.KernelHandler(workflowEngine.GenerateSSHKey))
	mux.HandleFunc("PUT /akoflow-api/instance/", http_config.KernelHandler(workflowEngine.SaveInstance))

	mux.HandleFunc("POST /akoflow-api/environments/", http_config.KernelHandler(workflowEngine.CreateEnvironment))
	mux.HandleFunc("GET /akoflow-api/environments/", http_config.KernelHandler(workflowEngine.ListEnvironments))
	mux.HandleFunc("GET /akoflow-api/environments/{environmentId}/", http_config.KernelHandler(workflowEngine.GetEnvironment))
	mux.HandleFunc("POST /akoflow-api/environment-connections/{connectionId}/health/", http_config.KernelHandler(workflowEngine.CheckEnvironmentConnection))
	mux.HandleFunc("PUT /akoflow-api/environment-connections/{connectionId}/", http_config.KernelHandler(workflowEngine.UpdateEnvironmentConnection))
	mux.HandleFunc("POST /akoflow-api/environment-connections/{connectionId}/discover/", http_config.KernelHandler(workflowEngine.DiscoverEnvironmentConnection))
	mux.HandleFunc("GET /akoflow-api/environment-connections/{connectionId}/history/", http_config.KernelHandler(workflowEngine.ListEnvironmentConnectionHistory))
	mux.HandleFunc("GET /akoflow-api/resources/", http_config.KernelHandler(workflowEngine.ListResources))
	mux.HandleFunc("GET /akoflow-api/resources/{resourceId}/", http_config.KernelHandler(workflowEngine.GetResource))
	mux.HandleFunc("GET /akoflow-api/resources/{resourceId}/snapshot/", http_config.KernelHandler(workflowEngine.GetResourceSnapshot))
	mux.HandleFunc("POST /akoflow-api/resources/", http_config.KernelHandler(workflowEngine.CreateResource))
	mux.HandleFunc("POST /akoflow-api/network-topologies/", http_config.KernelHandler(workflowEngine.CreateNetworkTopology))
	mux.HandleFunc("GET /akoflow-api/network-topologies/", http_config.KernelHandler(workflowEngine.ListNetworkTopologies))
	mux.HandleFunc("GET /akoflow-api/network-topologies/{topologyId}/", http_config.KernelHandler(workflowEngine.GetNetworkTopology))
	mux.HandleFunc("POST /akoflow-api/execution-scopes/", http_config.KernelHandler(workflowEngine.CreateExecutionScope))
	mux.HandleFunc("GET /akoflow-api/execution-scopes/", http_config.KernelHandler(workflowEngine.ListExecutionScopes))
	mux.HandleFunc("GET /akoflow-api/execution-scopes/{scopeId}/", http_config.KernelHandler(workflowEngine.GetExecutionScope))
	mux.HandleFunc("POST /akoflow-api/workflow-definitions/", http_config.KernelHandler(workflowEngine.CreateWorkflow))
	mux.HandleFunc("GET /akoflow-api/workflow-definitions/", http_config.KernelHandler(workflowEngine.ListWorkflows))
	mux.HandleFunc("GET /akoflow-api/workflow-definitions/{workflowId}/", http_config.KernelHandler(workflowEngine.GetWorkflow))
	mux.HandleFunc("POST /akoflow-api/schedule-plans/", http_config.KernelHandler(workflowEngine.CreatePlan))
	mux.HandleFunc("GET /akoflow-api/schedule-plans/", http_config.KernelHandler(workflowEngine.ListPlans))
	mux.HandleFunc("GET /akoflow-api/schedule-plans/{planId}/", http_config.KernelHandler(workflowEngine.GetPlan))
	mux.HandleFunc("POST /akoflow-api/execution-runs/", http_config.KernelHandler(workflowEngine.CreateExecution))
	mux.HandleFunc("GET /akoflow-api/execution-runs/", http_config.KernelHandler(workflowEngine.ListExecutions))
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
