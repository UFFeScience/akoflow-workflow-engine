package main

import (
	"github.com/UFFeScience/akoflow/internal/api/handlers/workflow_engine_api_handler"
	"github.com/UFFeScience/akoflow/internal/application/ports"
	appstorage "github.com/UFFeScience/akoflow/internal/application/storage"
	"github.com/UFFeScience/akoflow/internal/domain"
	"github.com/UFFeScience/akoflow/internal/infrastructure/credentials/sshkey"
	planningplugin "github.com/UFFeScience/akoflow/internal/infrastructure/plugins/planning"
	"github.com/UFFeScience/akoflow/internal/provider"
	filesystem "github.com/UFFeScience/akoflow/internal/provider/storage/filesystem"
	s3 "github.com/UFFeScience/akoflow/internal/provider/storage/s3"
	sshfilesystem "github.com/UFFeScience/akoflow/internal/provider/storage/sshfilesystem"
)

func buildAPI(
	storage persistence,
	connections ports.ConnectionHealthMonitor,
	discovery ports.EnvironmentDiscovery,
	console ports.ConsoleCommands,
	terminal ports.InteractiveConsole,
	sshKeys *sshkey.Manager,
) (*workflow_engine_api_handler.Handler, error) {
	fs, err := filesystem.New(domain.StorageLocal, "/")
	if err != nil {
		return nil, err
	}
	ssh := sshfilesystem.New(storage.environments, provider.OSCommandExecutor{})
	browsers := appstorage.Registry{domain.StorageLocal: fs, domain.StoragePVC: fs, domain.StorageNFS: fs, domain.StorageLustre: fs, domain.StorageSSH: ssh, domain.StorageS3: s3.New(nil, nil), domain.StorageMinIO: s3.New(nil, nil)}
	return workflow_engine_api_handler.New(workflow_engine_api_handler.Dependencies{
		Environments: storage.environments,
		Workflows:    storage.workflows,
		Plans:        storage.plans,
		Events:       storage.events,
		Validator:    planningplugin.NewValidator(),
		Executions:   storage.executions,
		Topologies:   storage.topologies,
		Scopes:       storage.topologies,
		Data:         storage.data,
		Resources:    storage.resources,
		Instance:     storage.instance,
		Connections:  connections,
		Discovery:    discovery,
		SSHKeys:      sshKeys,
		Audit:        storage.audit,
		Console:      console,
		Terminal:     terminal,
		Storage:      appstorage.NewBrowserCoordinator(storage.storage, browsers),
	})
}
