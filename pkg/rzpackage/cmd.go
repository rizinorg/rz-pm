package rzpackage

import "os/exec"

type commandExecutor struct{}

func (ce *commandExecutor) Run(cmd *exec.Cmd) error {
	return cmd.Run()
}
