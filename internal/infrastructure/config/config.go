package config

import (
	"os"
	"time"
)

type Settings struct {
	HTTPAddress                string
	DefaultNamespace           string
	KubernetesAPIServer        string
	KubernetesToken            string
	KubernetesCAFile           string
	KubernetesInsecureSkipTLS  bool
	KubernetesObserverImage    string
	KubernetesCleanupEnabled   bool
	KubernetesCleanupInterval  time.Duration
	KubernetesHistoryRetention time.Duration
}

func Load() Settings {
	settings := Settings{
		HTTPAddress: ":8080", DefaultNamespace: "akoflow",
		KubernetesCleanupEnabled: true, KubernetesCleanupInterval: 15 * time.Minute,
		KubernetesHistoryRetention: 24 * time.Hour,
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
	settings.KubernetesObserverImage = os.Getenv("AKOFLOW_KUBERNETES_OBSERVER_IMAGE")
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
