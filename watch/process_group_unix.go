//go:build darwin || linux

package watch

import (
	"os"
	"os/exec"
	"syscall"
)

func configureProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func signalProcessGroup(process *os.Process, sig os.Signal) error {
	return syscall.Kill(-process.Pid, sig.(syscall.Signal))
}
