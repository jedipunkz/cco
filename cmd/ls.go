package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/thirai/cco/internal/store"
)

var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List all agent IDs",
	RunE: func(cmd *cobra.Command, args []string) error {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("could not determine home directory: %w", err)
		}
		stateFile := filepath.Join(home, ".cco", "state.json")

		data, err := os.ReadFile(stateFile)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Println("no agents found")
				return nil
			}
			return fmt.Errorf("could not read state: %w", err)
		}

		var agents []store.AgentState
		if err := json.Unmarshal(data, &agents); err != nil {
			return fmt.Errorf("could not parse state: %w", err)
		}

		if len(agents) == 0 {
			fmt.Println("no agents found")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tSTATUS\tWORK DIR")
		for _, a := range agents {
			fmt.Fprintf(w, "%s\t%s\t%s\n", a.ID, a.Status, a.WorkDir)
		}
		return w.Flush()
	},
}
