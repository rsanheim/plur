//go:build windows

package watch

import (
	"os"
	"os/exec"
)

func configureProcessGroup(_ *exec.Cmd) {}

func signalProcessGroup(process *os.Process, sig os.Signal) error {
	return process.Signal(sig)
}
