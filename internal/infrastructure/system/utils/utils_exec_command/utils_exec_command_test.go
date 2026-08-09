package utils_exec_command

import "testing"

func TestRunCommandSuccessAndFailure(t *testing.T) {
	u := New()
	response, _, outputErr := u.RunCommand("sh", "-c", "exit 0")
	if response != nil || outputErr != nil {
		t.Fatalf("expected success, got %v / %v", response, outputErr)
	}
	response, _, outputErr = u.RunCommand("sh", "-c", "exit 7")
	if response == nil || outputErr == nil {
		t.Fatalf("expected command failure, got %v / %v", response, outputErr)
	}
	response, _, outputErr = u.RunCommand("/command/that/does/not/exist")
	if response == nil || outputErr == nil {
		t.Fatal("missing executable must fail")
	}
}
