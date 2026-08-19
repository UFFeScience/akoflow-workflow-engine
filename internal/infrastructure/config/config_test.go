package config

import (
	"testing"
	"time"
)

func TestVersionAndEnvironmentCollection(t *testing.T) {
	t.Setenv("AKOFLOW_SERVER_VERSION", "1.2.3")
	if GetVersion() != "1.2.3" {
		t.Fatal("version env")
	}
	t.Setenv("AKOFLOW_SERVER_VERSION", "")
	if GetVersion() != "dev-env" {
		t.Fatal("default version")
	}
}

func TestLoadSettings(t *testing.T) {
	t.Setenv("AKOFLOW_HTTP_ADDRESS", ":9090")
	t.Setenv("AKOFLOW_NAMESPACE", "science")
	t.Setenv("K8S_API_SERVER_HOST", "kind-control-plane:6443")
	t.Setenv("K8S_API_SERVER_TOKEN", "token")
	t.Setenv("K8S_API_SERVER_CA_FILE", "/tmp/kind-ca.crt")
	t.Setenv("K8S_API_SERVER_INSECURE_SKIP_TLS_VERIFY", "true")
	t.Setenv("AKOFLOW_KUBERNETES_HISTORY_CLEANUP_ENABLED", "true")
	t.Setenv("AKOFLOW_KUBERNETES_HISTORY_CLEANUP_INTERVAL", "5m")
	t.Setenv("AKOFLOW_KUBERNETES_HISTORY_RETENTION", "12h")
	t.Setenv("AKOFLOW_SLURM_SCRIPT_DIRECTORY", "/shared/akoflow/scripts")
	settings := Load()
	if settings.HTTPAddress != ":9090" || settings.DefaultNamespace != "science" {
		t.Fatalf("settings=%+v", settings)
	}
	if settings.KubernetesAPIServer != "kind-control-plane:6443" ||
		settings.KubernetesToken != "token" ||
		settings.KubernetesCAFile != "/tmp/kind-ca.crt" ||
		!settings.KubernetesInsecureSkipTLS ||
		!settings.KubernetesCleanupEnabled ||
		settings.KubernetesCleanupInterval != 5*time.Minute ||
		settings.KubernetesHistoryRetention != 12*time.Hour ||
		settings.SlurmScriptDirectory != "/shared/akoflow/scripts" {
		t.Fatalf("Kubernetes settings=%+v", settings)
	}
}
