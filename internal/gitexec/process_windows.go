//go:build windows

package gitexec

import (
	"os/exec"
)

func configureProcess(*exec.Cmd) {}

func terminateProcess(process *exec.Cmd) {
	if process.Process != nil {
		_ = process.Process.Kill()
	}
}
