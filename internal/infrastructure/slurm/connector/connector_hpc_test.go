package connector_hpc

import (
	"encoding/base64"
	"strings"
	"testing"

	runtime_entity "github.com/UFFeScience/akoflow/internal/domain/resource/runtime"
)

func TestBuildRemoteCommandAuthentication(t *testing.T) {
	c := New().SetRuntime(runtime_entity.Runtime{})
	passwordRuntime := runtime_entity.Runtime{Name: "hpc", Metadata: map[string]string{"HPC_USER": "user", "HPC_HOST_CLUSTER": "host", "HPC_PASSWORD": "pass"}}
	command, err := c.BuildRemoteCommand(passwordRuntime, "hostname")
	if err != nil || !strings.Contains(command, "sshpass -p 'pass'") || !strings.Contains(command, "user@host") {
		t.Fatalf("unexpected command: %q %v", command, err)
	}
	if _, err := c.BuildRemoteCommand(runtime_entity.Runtime{Name: "hpc"}, "hostname"); err == nil {
		t.Fatal("missing auth must fail")
	}
}

func TestHPCEncodingAndLocalCommands(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte("value"))
	decoded, err := decodeBase64(encoded)
	if err != nil || decoded != "value" {
		t.Fatal("decode failed")
	}
	if _, err := decodeBase64("%%%"); err == nil {
		t.Fatal("invalid base64 must fail")
	}
	c := &ConnectorHPCRuntime{}
	out, err := c.RunCommandWithOutput("printf hpc")
	if err != nil || out != "hpc" {
		t.Fatalf("output: %q %v", out, err)
	}
	if _, err := c.RunCommandWithOutput("exit 2"); err == nil {
		t.Fatal("failure expected")
	}
	if _, err := c.RunCommand("true"); err != nil {
		t.Fatal(err)
	}
	c.ExecuteMultiplesCommand([]string{"printf one", "exit 1"})
	if err := c.handleCreateSSHKey("", "", ""); err != nil {
		t.Fatal(err)
	}
}

func TestHPCRemoteAndVPNFailurePaths(t *testing.T) {
	c := &ConnectorHPCRuntime{}
	passwordRuntime := runtime_entity.Runtime{Name: "hpc", Metadata: map[string]string{"HPC_USER": "u", "HPC_HOST_CLUSTER": "127.0.0.1", "HPC_PASSWORD": "p"}}
	c.SetRuntime(passwordRuntime)
	if _, err := c.RunCommandWithOutputRemote("true"); err == nil {
		t.Fatal("remote command should fail in test environment")
	}
	if ok, err := c.IsVPNConnected(); err != nil || !ok {
		t.Fatalf("vpn process probe: %v %v", ok, err)
	}
	if getAvailableShell() == "" {
		t.Fatal("shell")
	}
	if _, err := writeTempSSHKey("key", "nested/missing"); err == nil {
		t.Fatal("write should fail")
	}
	removeTempSSHKey("/tmp/nonexistent-akoflow-key")
}
