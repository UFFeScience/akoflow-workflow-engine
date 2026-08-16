package config

import (
	"os"
	"testing"
)

func TestVersionAndEnvironmentCollection(t *testing.T) {
	t.Setenv("AKOFLOW_SERVER_VERSION", "1.2.3")
	if GetVersion() != "1.2.3" {
		t.Fatal("version env")
	}
	if err := os.Unsetenv("AKOFLOW_SERVER_VERSION"); err != nil {
		t.Fatal(err)
	}
	if GetVersion() != "dev-env" {
		t.Fatal("default version")
	}
	t.Setenv("K8S_SAMPLE_VALUE", "x")
	all, byRuntime := GetEnvVars()
	if all["K8S_SAMPLE_VALUE"] != "x" || byRuntime["k8s"]["K8S_SAMPLE_VALUE"] != "x" {
		t.Fatal("environment grouping")
	}
}

func TestSetupEnvAndContainerSingleton(t *testing.T) {
	t.Setenv("K8S_API_SERVER_TOKEN", "token")
	t.Setenv("KUBERNETES_SERVICE_HOST", "cluster.local")
	SetupEnv()
	if os.Getenv("K8S_API_SERVER_HOST") != "cluster.local" {
		t.Fatal("host")
	}
	original := appContainer
	t.Cleanup(func() { appContainer = original })
	custom := AppContainer{DefaultNamespace: "custom"}
	SetAppContainer(custom)
	if App().DefaultNamespace != "custom" {
		t.Fatal("singleton")
	}
}

func TestMakeAppContainer(t *testing.T) {
	c := MakeAppContainer()
	if c.DefaultNamespace != DEFAULT_NAMESPACE || c.Logger == nil || c.Repository.WorkflowRepository == nil || c.Connector.K8sConnector == nil || c.HttpHelper.WriteJson == nil {
		t.Fatal("incomplete container")
	}
}
