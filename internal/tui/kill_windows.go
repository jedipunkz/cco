//go:build windows

package tui

import "os"

// killProcess terminates the process on Windows.
func killProcess(pid int) {
	p, err := os.FindProcess(pid)
	if err == nil {
		_ = p.Kill()
	}
}
