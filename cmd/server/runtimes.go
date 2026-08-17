package main

import (
	"fmt"

	"github.com/UFFeScience/akoflow/internal/application/ports"
	"github.com/UFFeScience/akoflow/internal/infrastructure/config"
	"github.com/UFFeScience/akoflow/internal/provider"
	"github.com/UFFeScience/akoflow/internal/provider/kubernetes"
	"github.com/UFFeScience/akoflow/internal/provider/local"
	"github.com/UFFeScience/akoflow/internal/provider/registry"
	"github.com/UFFeScience/akoflow/internal/provider/simgrid"
	"github.com/UFFeScience/akoflow/internal/provider/slurm"
)

func connectKubernetes(settings config.Settings) (kubernetes.API, error) {
	if settings.KubernetesAPIServer == "" && settings.KubernetesToken == "" {
		return nil, nil
	}
	client, err := kubernetes.NewClient(kubernetes.ClientConfig{
		Endpoint: settings.KubernetesAPIServer, Token: settings.KubernetesToken,
		CAFile:                settings.KubernetesCAFile,
		InsecureSkipTLSVerify: settings.KubernetesInsecureSkipTLS,
	})
	if err != nil {
		return nil, fmt.Errorf("configure Kubernetes API: %w", err)
	}
	return client, nil
}

func buildRuntimes(settings config.Settings, kubernetesAPI kubernetes.API) (*registry.Registry, error) {
	runtimes := registry.New()
	adapters := map[string]ports.RuntimeAdapter{
		"*":          simgrid.NewActivityRuntime(),
		"local":      local.New(),
		"kubernetes": kubernetes.New(kubernetesAPI, settings.DefaultNamespace),
		"slurm":      slurm.New(provider.OSCommandExecutor{}, ""),
	}
	for runtimeID, adapter := range adapters {
		if err := runtimes.Register(runtimeID, adapter); err != nil {
			return nil, err
		}
	}
	return runtimes, nil
}
