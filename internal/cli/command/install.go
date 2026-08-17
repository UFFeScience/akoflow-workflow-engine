package command

import (
	"flag"
	"os"

	"github.com/UFFeScience/akoflow/internal/cli/install"
	clissh "github.com/UFFeScience/akoflow/internal/cli/ssh"
	"github.com/UFFeScience/akoflow/internal/infrastructure/system/utils/utils_parser_params_ssh_client"
)

type Install struct{}

func (i *Install) Run() {

	hostsStr := flag.String("hosts", "<host1>,<host2>", "Hosts to install the CLI service")
	identityFile := flag.String("identity", "~/.ssh/id_rsa", "Identity file")

	flag.CommandLine.Parse(os.Args[2:])

	hosts := utils_parser_params_ssh_client.
		New().
		SetIdentityFile(*identityFile).
		Parse(*hostsStr)

	connection := clissh.New()
	for _, host := range hosts {
		connection.AddHost(host)
	}
	install.NewClusterInstaller().SetConnection(connection).Install()

}
