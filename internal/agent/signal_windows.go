//go:build windows

package agent

import "os"

// setupWinchHandler is a no-op on Windows (SIGWINCH does not exist).
func setupWinchHandler(_ *os.File, _ <-chan struct{}) {}
