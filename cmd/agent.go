package cmd

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/thirai/cco/internal/agent"
)

var agentCmd = &cobra.Command{
	Use:                "agent [-- <claude-args>...]",
	Short:              "Start a Claude Code agent",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		socketPath, err := getSocketPath()
		if err != nil {
			return err
		}

		if err := ensureDaemon(socketPath); err != nil {
			return fmt.Errorf("could not start daemon: %w", err)
		}

		return agent.Run(args, socketPath)
	},
}

func getSocketPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not determine home directory: %w", err)
	}
	return filepath.Join(home, ".cco", "cco.sock"), nil
}

func ensureDaemon(socketPath string) error {
	// Check if socket exists and is connectable
	if isSocketAlive(socketPath) {
		return nil
	}

	// Fork daemon process
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not determine executable path: %w", err)
	}

	daemonCmd := exec.Command(exe, "daemon")
	daemonCmd.Stdout = nil
	daemonCmd.Stderr = nil
	daemonCmd.Stdin = nil
	// Setsid puts the daemon in a new session so it is detached from the
	// controlling terminal and will not be killed when the user's shell exits.
	daemonCmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := daemonCmd.Start(); err != nil {
		return fmt.Errorf("could not start daemon: %w", err)
	}

	// Wait up to 3 seconds for socket to appear
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if isSocketAlive(socketPath) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("daemon did not start within 3 seconds")
}

func isSocketAlive(socketPath string) bool {
	conn, err := net.DialTimeout("unix", socketPath, 500*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
