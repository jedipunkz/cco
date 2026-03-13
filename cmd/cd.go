package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/thirai/cco/internal/store"
)

var cdCmd = &cobra.Command{
	Use:   "cd <id>",
	Short: "Print the worktree path for an agent (use: cd (cco cd <id>))",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]

		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("could not determine home directory: %w", err)
		}
		stateFile := filepath.Join(home, ".cco", "state.json")

		data, err := os.ReadFile(stateFile)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("agent %q not found", id)
			}
			return fmt.Errorf("could not read state: %w", err)
		}

		var agents []store.AgentState
		if err := json.Unmarshal(data, &agents); err != nil {
			return fmt.Errorf("could not parse state: %w", err)
		}

		for _, a := range agents {
			if a.ID == id {
				fmt.Print(a.WorkDir)
				return nil
			}
		}

		return fmt.Errorf("agent %q not found", id)
	},
}
