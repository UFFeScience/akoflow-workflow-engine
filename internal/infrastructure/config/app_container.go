package config

import (
	"net/http"
	"os"
	"strings"

	"github.com/UFFeScience/akoflow/internal/infrastructure/config/http_helper"
	"github.com/UFFeScience/akoflow/internal/infrastructure/config/logger"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/activity_repository"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/logs_repository"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/metrics_repository"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/resource_repository"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/runtime_repository"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/schedule_repository"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/storages_repository"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/workflow_execution_repository"
	"github.com/UFFeScience/akoflow/internal/infrastructure/database/repository/workflow_repository"
	"github.com/UFFeScience/akoflow/internal/infrastructure/kubernetes/connector"
	"github.com/UFFeScience/akoflow/internal/infrastructure/slurm/connector"
	"github.com/UFFeScience/akoflow/internal/infrastructure/ssh/local_connector"
	"github.com/UFFeScience/akoflow/internal/infrastructure/ssh/singularity_connector"
)

const DEFAULT_NAMESPACE = "akoflow"
const LOG_FILE_PATH = "akoflow.log"

type AppContainer struct {
	Repository       AppContainerRepository
	Connector        AppContainerConnector
	DefaultNamespace string
	HttpHelper       AppContainerHttpHelper
	Logger           *logger.Logger
	EnvVars          EnvVars
}
type EnvVars struct {
	EnvVars         map[string]string
	EnvVarByRuntime map[string]map[string]string
}

type AppContainerRepository struct {
	WorkflowRepository          workflow_repository.IWorkflowRepository
	ActivityRepository          activity_repository.IActivityRepository
	LogsRepository              logs_repository.ILogsRepository
	MetricsRepository           metrics_repository.IMetricsRepository
	StoragesRepository          storages_repository.IStorageRepository
	RuntimeRepository           runtime_repository.IRuntimeRepository
	ResourceRepository          resource_repository.IRepository
	WorkflowExecutionRepository workflow_execution_repository.IWorkflowExecutionRepository
	ScheduleRepository          schedule_repository.IScheduleRepository
}

type AppContainerConnector struct {
	K8sConnector         connector_k8s.IConnector
	SingularityConnector connector_singularity.IConnectorSingularity
	HPCRuntimeConnector  connector_hpc.IConnectorHPCRuntime
	LocalConnector       connector_local.IConnectorLocal
}

type AppContainerHttpHelper struct {
	WriteJson   func(w http.ResponseWriter, data interface{})
	GetUrlParam func(r *http.Request, key string) string
	ReadJson    func(r *http.Request, data interface{}) error
}

// GetEnvVars returns the environment variables as a map
func GetEnvVars() (map[string]string, map[string]map[string]string) {
	envVars := make(map[string]string)
	envVarByRuntime := make(map[string]map[string]string)

	runtimes_avaibles := []string{"k8s", "singularity", "hpc"}

	for _, v := range os.Environ() {
		splitted := strings.Split(v, "=")
		for _, runtime := range runtimes_avaibles {

			envVar := os.Getenv(splitted[0])
			envVars[splitted[0]] = envVar

			if strings.Contains(strings.ToLower(splitted[0]), runtime) {

				currentRuntime := strings.ToLower(splitted[0])
				stringRuntimeSplitted := strings.Split(currentRuntime, "_")
				runtime := stringRuntimeSplitted[0]
				runtime = strings.ToLower(runtime)

				if envVarByRuntime[runtime] == nil {
					envVarByRuntime[runtime] = make(map[string]string)
					envVarByRuntime[runtime][splitted[0]] = envVar
				} else {
					envVarByRuntime[runtime][splitted[0]] = envVar
				}

			}
		}
	}
	return envVars, envVarByRuntime
}

func MakeAppContainer() AppContainer {

	// Create the repository instances
	workflowRepository := workflow_repository.New()
	activityRepository := activity_repository.New()
	logsRepository := logs_repository.New()
	metricsRepository := metrics_repository.New()
	storagesRepository := storages_repository.New()
	runtimeRepository := runtime_repository.New()
	resourceRepository := resource_repository.New()
	workflowExecutionRepository := workflow_execution_repository.New()
	scheduleRepository := schedule_repository.New()

	// create the Connector instances
	k8sConnector := connector_k8s.New()
	singularityConnector := connector_singularity.New()
	hpcConnector := connector_hpc.New()
	localConnector := connector_local.New()

	logger, _ := logger.NewLogger(LOG_FILE_PATH)

	envVars, envVarByRuntime := GetEnvVars()

	appContainer := AppContainer{
		DefaultNamespace: DEFAULT_NAMESPACE,
		Repository: AppContainerRepository{
			WorkflowRepository:          workflowRepository,
			ActivityRepository:          activityRepository,
			LogsRepository:              logsRepository,
			MetricsRepository:           metricsRepository,
			StoragesRepository:          storagesRepository,
			RuntimeRepository:           runtimeRepository,
			ResourceRepository:          resourceRepository,
			WorkflowExecutionRepository: workflowExecutionRepository,
			ScheduleRepository:          scheduleRepository,
		},
		Connector: AppContainerConnector{
			K8sConnector:         k8sConnector,
			SingularityConnector: singularityConnector,
			HPCRuntimeConnector:  hpcConnector,
			LocalConnector:       localConnector,
		},
		HttpHelper: AppContainerHttpHelper{
			WriteJson:   http_helper.WriteJson,
			GetUrlParam: http_helper.GetUrlPathParam,
			ReadJson:    http_helper.ReadJson,
		},
		Logger: logger,
		EnvVars: EnvVars{
			EnvVars:         envVars,
			EnvVarByRuntime: envVarByRuntime,
		},
	}
	return appContainer
}

// singleton appContainer
var appContainer AppContainer

func App() AppContainer {
	if appContainer.DefaultNamespace == "" {
		appContainer = MakeAppContainer()
	}

	return appContainer
}

func SetAppContainer(container AppContainer) {
	appContainer = container
}
