package cmd

import (
	"fmt"
	"os"

	"github.com/bootcamp/simplijobs/internal/display"
	"github.com/bootcamp/simplijobs/internal/store"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show when you last checked for new listings",
	Long:  `Display the last-checked timestamps for each source (internships, newgrad).`,
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	if noColor {
		os.Setenv("NO_COLOR", "1")
	}

	state, err := store.Load()
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}

	display.RenderStatus(
		state.GetLastChecked("internships"),
		state.GetLastChecked("newgrad"),
	)

	return nil
}
