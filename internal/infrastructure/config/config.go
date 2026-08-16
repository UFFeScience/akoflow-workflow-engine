package config

import (
	"os"
)

type Settings struct {
	HTTPAddress      string
	DefaultNamespace string
}

func Load() Settings {
	settings := Settings{HTTPAddress: ":8080", DefaultNamespace: "akoflow"}
	if value := os.Getenv("AKOFLOW_HTTP_ADDRESS"); value != "" {
		settings.HTTPAddress = value
	}
	if value := os.Getenv("AKOFLOW_NAMESPACE"); value != "" {
		settings.DefaultNamespace = value
	}
	return settings
}

func GetVersion() string {

	versionEnv := os.Getenv("AKOFLOW_SERVER_VERSION")
	if versionEnv != "" {
		return versionEnv
	}
	return "dev-env"
}
