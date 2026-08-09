package dispatch_to_server_run_workflow_service

import (
	"testing"

	"github.com/UFFeScience/akoflow/internal/cli/api/server_connector/server_connector_workflow"
	"github.com/UFFeScience/akoflow/internal/infrastructure/system/utils/utils_create_file"
	"github.com/stretchr/testify/assert"
)

type connectorStub struct {
	workflow server_connector_workflow.IWorkflow
}

func (s connectorStub) Workflow() server_connector_workflow.IWorkflow { return s.workflow }

type workflowStub struct {
	create func(string, string, string) error
}

func (s workflowStub) Create(host, port, content string) error { return s.create(host, port, content) }

func TestDispatchToServerRunWorkflowService_Run(t *testing.T) {

	connector := connectorStub{workflow: workflowStub{create: func(string, string, string) error { return nil }}}

	file := utils_create_file.New().CreateTempFile("content")

	dispatchToServerRunWorkflowService := New(connector)
	dispatchToServerRunWorkflowService.SetHost("localhost")
	dispatchToServerRunWorkflowService.SetPort("8080")
	dispatchToServerRunWorkflowService.SetFile(file)

	assert.NoError(t, dispatchToServerRunWorkflowService.Run())

}

func TestDispatchToServerRunWorkflowService_SetHost(t *testing.T) {

	dispatchToServerRunWorkflowService := New(connectorStub{})
	dispatchToServerRunWorkflowService.SetHost("localhost")

	assert.Equal(t, "localhost", dispatchToServerRunWorkflowService.GetHost())
}

func TestDispatchToServerRunWorkflowService_SetPort(t *testing.T) {

	dispatchToServerRunWorkflowService := New(connectorStub{})
	dispatchToServerRunWorkflowService.SetPort("8080")

	assert.Equal(t, "8080", dispatchToServerRunWorkflowService.GetPort())
}

func TestDispatchToServerRunWorkflowService_SetFile(t *testing.T) {

	dispatchToServerRunWorkflowService := New(connectorStub{})
	dispatchToServerRunWorkflowService.SetFile("file")

	assert.Equal(t, "file", dispatchToServerRunWorkflowService.GetFile())
}

func TestDispatchToServerRunWorkflowService_GetBase64FileContent(t *testing.T) {

	dispatchToServerRunWorkflowService := New(connectorStub{})

	file := utils_create_file.New().CreateTempFile("content")

	base64FileContent := dispatchToServerRunWorkflowService.getBase64FileContent(file)

	assert.Equal(t, "Y29udGVudA==", base64FileContent)
}

func TestDispatchToServerRunWorkflowService_GetFileContent(t *testing.T) {

	dispatchToServerRunWorkflowService := New(connectorStub{})

	file := utils_create_file.New().CreateTempFile("content")

	fileContent := dispatchToServerRunWorkflowService.getFileContent(file)

	assert.Equal(t, "content", fileContent)
}
