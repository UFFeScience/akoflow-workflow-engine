package workflow

import (
	"testing"

	"github.com/UFFeScience/akoflow/internal/cli/api/server_connector/server_connector_workflow"
	"github.com/UFFeScience/akoflow/internal/infrastructure/system/utils/utils_create_file"
	"github.com/stretchr/testify/assert"
)

type connectorStub struct {
	workflow server_connector_workflow.Client
}

func (s connectorStub) Workflow() server_connector_workflow.Client { return s.workflow }

type workflowStub struct {
	create func(string, string, string) error
}

func (s workflowStub) Create(host, port, content string) error { return s.create(host, port, content) }

func TestRunner_Run(t *testing.T) {

	connector := connectorStub{workflow: workflowStub{create: func(string, string, string) error { return nil }}}

	file := utils_create_file.New().CreateTempFile("content")

	runner := New(connector)
	runner.SetHost("localhost")
	runner.SetPort("8080")
	runner.SetFile(file)

	assert.NoError(t, runner.Run())

}

func TestRunner_SetHost(t *testing.T) {

	runner := New(connectorStub{})
	runner.SetHost("localhost")

	assert.Equal(t, "localhost", runner.GetHost())
}

func TestRunner_SetPort(t *testing.T) {

	runner := New(connectorStub{})
	runner.SetPort("8080")

	assert.Equal(t, "8080", runner.GetPort())
}

func TestRunner_SetFile(t *testing.T) {

	runner := New(connectorStub{})
	runner.SetFile("file")

	assert.Equal(t, "file", runner.GetFile())
}

func TestRunner_GetBase64FileContent(t *testing.T) {

	runner := New(connectorStub{})

	file := utils_create_file.New().CreateTempFile("content")

	base64FileContent := runner.getBase64FileContent(file)

	assert.Equal(t, "Y29udGVudA==", base64FileContent)
}

func TestRunner_GetFileContent(t *testing.T) {

	runner := New(connectorStub{})

	file := utils_create_file.New().CreateTempFile("content")

	fileContent := runner.getFileContent(file)

	assert.Equal(t, "content", fileContent)
}
