package workflow

import (
	"encoding/base64"

	"github.com/UFFeScience/akoflow/internal/cli/api/server_connector"
	"github.com/UFFeScience/akoflow/internal/infrastructure/system/utils/utils_read_file"
)

type Runner struct {
	host            string
	port            string
	file            string
	serverConnector server_connector.Connector
}

func New(serverConnector server_connector.Connector) *Runner {
	return &Runner{
		serverConnector: serverConnector,
	}
}

func (d *Runner) SetHost(host string) *Runner {
	d.host = host
	return d
}

func (d *Runner) SetPort(port string) *Runner {
	d.port = port
	return d
}

func (d *Runner) SetFile(file string) *Runner {
	d.file = file
	return d
}

func (d *Runner) GetHost() string {
	return d.host
}

func (d *Runner) GetPort() string {
	return d.port
}

func (d *Runner) GetFile() string {
	return d.file
}

func (d *Runner) Run() error {
	base64FileContent := d.getBase64FileContent(d.GetFile())
	return d.sendToServer(base64FileContent)
}

func (d *Runner) getBase64FileContent(filePath string) string {
	fileContent := d.getFileContent(filePath)

	base64FileContent := base64.StdEncoding.EncodeToString([]byte(fileContent))
	return base64FileContent
}

func (d *Runner) getFileContent(filePath string) string {
	return utils_read_file.New().ReadFile(filePath)
}

func (d *Runner) sendToServer(base64FileContent string) error {
	return d.serverConnector.Workflow().Create(d.host, d.port, base64FileContent)
}
