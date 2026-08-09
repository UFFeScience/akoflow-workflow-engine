package install_cli_service

import (
	"flag"
	"os"

	"github.com/UFFeScience/akoflow/internal/cli/application/install_akoflow_local_service"
	"github.com/UFFeScience/akoflow/internal/cli/application/ssh_connection_service"
	"github.com/UFFeScience/akoflow/internal/infrastructure/system/utils/utils_parser_params_ssh_client"
)

type InstallCliService struct {
}

func New() *InstallCliService {
	return &InstallCliService{}
}

func (i *InstallCliService) Run() {

	hostsStr := flag.String("hosts", "<host1>,<host2>", "Hosts to install the CLI service")
	identityFile := flag.String("identity", "~/.ssh/id_rsa", "Identity file")

	flag.CommandLine.Parse(os.Args[2:])

	hosts := utils_parser_params_ssh_client.
		New().
		SetIdentityFile(*identityFile).
		Parse(*hostsStr)

	sshConnectionService := ssh_connection_service.New()
	for _, host := range hosts {
		sshConnectionService.AddHost(host)
	}

	install_akoflow_local_service := install_akoflow_local_service.New()

	install_akoflow_local_service.
		SetSSHConnectionService(sshConnectionService).
		Install()

}
