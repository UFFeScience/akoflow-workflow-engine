package config

import (
	"os"
)

type Settings struct {
	HTTPAddress               string
	DefaultNamespace          string
	KubernetesAPIServer       string
	KubernetesToken           string
	KubernetesCAFile          string
	KubernetesInsecureSkipTLS bool
}

func Load() Settings {
	settings := Settings{HTTPAddress: ":8080", DefaultNamespace: "akoflow"}
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
	return settings
}

func GetVersion() string {

	versionEnv := os.Getenv("AKOFLOW_SERVER_VERSION")
	if versionEnv != "" {
		return versionEnv
	}
	return "dev-env"
}
