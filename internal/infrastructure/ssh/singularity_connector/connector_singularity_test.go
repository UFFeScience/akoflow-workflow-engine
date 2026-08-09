package connector_singularity

import (
	"strconv"
	"strings"
	"testing"
)

func TestSingularityConnectorCommands(t *testing.T) {
	c := New()
	out, err := c.RunCommandWithOutput("printf singularity")
	if err != nil || out != "singularity" {
		t.Fatalf("output: %q %v", out, err)
	}
	if _, err := c.RunCommandWithOutput("exit 3"); err == nil {
		t.Fatal("failed command must return error")
	}
	pid, err := c.RunCommand("true")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := strconv.Atoi(strings.TrimSpace(pid)); err != nil {
		t.Fatalf("invalid pid %q", pid)
	}
	if shell := getAvailableShell(); shell != "bash" && shell != "sh" {
		t.Fatalf("unexpected shell %s", shell)
	}
}
