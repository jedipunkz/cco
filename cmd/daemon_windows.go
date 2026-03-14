//go:build windows

package cmd

import (
	"os"
	"os/exec"
)

// setDaemonSysProcAttr is a no-op on Windows; detachment is handled by the OS.
func setDaemonSysProcAttr(_ *exec.Cmd) {}

// killPID terminates the given process on Windows.
func killPID(pid int) {
	p, err := os.FindProcess(pid)
	if err == nil {
		_ = p.Kill()
	}
}
