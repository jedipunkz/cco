package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thirai/cco/internal/tui"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show TUI status of all agents",
	RunE: func(cmd *cobra.Command, args []string) error {
		socketPath, err := getSocketPath()
		if err != nil {
			return err
		}

		if err := ensureDaemon(socketPath); err != nil {
			return fmt.Errorf("could not start daemon: %w", err)
		}

		return tui.Run(socketPath)
	},
}
