package utils_exec_command

import (
	"os/exec"
)

type UtilsExecCommand struct {
}

func New() *UtilsExecCommand {
	return &UtilsExecCommand{}
}

func (u *UtilsExecCommand) RunCommand(command string, args ...string) (error, []byte, error) {
	cmd := exec.Command(command, args...)
	out, err := cmd.CombinedOutput()
	return err, out, err
}
