package config

import (
	"os"
	"strconv"
	"time"
)

type Settings struct {
	HTTPAddress                string
	DefaultNamespace           string
	KubernetesAPIServer        string
	KubernetesToken            string
	KubernetesCAFile           string
	KubernetesInsecureSkipTLS  bool
	KubernetesCleanupEnabled   bool
	KubernetesCleanupInterval  time.Duration
	KubernetesHistoryRetention time.Duration
	SlurmScriptDirectory       string
	SimulationBackend          string
	SimGridBinaryPath          string
	SimGridWorkspace           string
	SimGridMaxConcurrent       int
	SimGridTimeout             time.Duration
	SimGridReferenceFLOPS      float64
}

func Load() Settings {
	settings := Settings{
		HTTPAddress: ":8080", DefaultNamespace: "akoflow",
		KubernetesCleanupEnabled: true, KubernetesCleanupInterval: 15 * time.Minute,
		KubernetesHistoryRetention: 24 * time.Hour,
		SlurmScriptDirectory:       "storage/slurm/scripts",
		SimulationBackend:          "simgrid",
		SimGridBinaryPath:          "akoflow-simgrid-runner",
		SimGridWorkspace:           "storage/simgrid",
		SimGridMaxConcurrent:       2,
		SimGridTimeout:             30 * time.Minute,
		SimGridReferenceFLOPS:      1e9,
	}
	if value := os.Getenv("AKOFLOW_HTTP_ADDRESS"); value != "" {
		settings.HTTPAddress = value
	}
	if value := os.Getenv("AKOFLOW_NAMESPACE"); value != "" {
		settings.DefaultNamespace = value
	}
	settings.KubernetesAPIServer = os.Getenv("K8S_API_SERVER_HOST")
	settings.KubernetesToken = os.Getenv("K8S_API_SERVER_TOKEN")
	settings.KubernetesCAFile = os.Getenv("K8S_API_SERVER_CA_FILE")
	settings.KubernetesInsecureSkipTLS = os.Getenv("K8S_API_SERVER_INSECURE_SKIP_TLS_VERIFY") == "true"
	if value := os.Getenv("AKOFLOW_SLURM_SCRIPT_DIRECTORY"); value != "" {
		settings.SlurmScriptDirectory = value
	}
	if value := os.Getenv("AKOFLOW_SIMULATION_BACKEND"); value != "" {
		settings.SimulationBackend = value
	}
	if value := os.Getenv("AKOFLOW_SIMGRID_BINARY"); value != "" {
		settings.SimGridBinaryPath = value
	}
	if value := os.Getenv("AKOFLOW_SIMGRID_WORKSPACE"); value != "" {
		settings.SimGridWorkspace = value
	}
	settings.SimGridMaxConcurrent = integerOrDefault(
		os.Getenv("AKOFLOW_SIMGRID_MAX_CONCURRENT"), settings.SimGridMaxConcurrent,
	)
	settings.SimGridReferenceFLOPS = floatOrDefault(
		os.Getenv("AKOFLOW_SIMGRID_REFERENCE_FLOPS"), settings.SimGridReferenceFLOPS,
	)
	settings.SimGridTimeout = durationOrDefault(
		os.Getenv("AKOFLOW_SIMGRID_TIMEOUT"), settings.SimGridTimeout,
	)
	if value := os.Getenv("AKOFLOW_KUBERNETES_HISTORY_CLEANUP_ENABLED"); value != "" {
		settings.KubernetesCleanupEnabled = value == "true"
	}
	settings.KubernetesCleanupInterval = durationOrDefault(
		os.Getenv("AKOFLOW_KUBERNETES_HISTORY_CLEANUP_INTERVAL"), settings.KubernetesCleanupInterval,
	)
	settings.KubernetesHistoryRetention = durationOrDefault(
		os.Getenv("AKOFLOW_KUBERNETES_HISTORY_RETENTION"), settings.KubernetesHistoryRetention,
	)
	return settings
}

func integerOrDefault(value string, fallback int) int {
	parsed, err := strconv.Atoi(value)
	if value == "" || err != nil || parsed < 1 {
		return fallback
	}
	return parsed
}

func floatOrDefault(value string, fallback float64) float64 {
	parsed, err := strconv.ParseFloat(value, 64)
	if value == "" || err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func durationOrDefault(value string, fallback time.Duration) time.Duration {
	parsed, err := time.ParseDuration(value)
	if value == "" || err != nil || parsed < 0 {
		return fallback
	}
	return parsed
}

func GetVersion() string {

	versionEnv := os.Getenv("AKOFLOW_SERVER_VERSION")
	if versionEnv != "" {
		return versionEnv
	}
	return "dev-env"
}
