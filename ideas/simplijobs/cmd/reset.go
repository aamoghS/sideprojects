package cmd

import (
	"fmt"
	"os"

	"github.com/aamoghS/sideprojects/ideas/simplijobs/internal/store"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset the last-checked timestamp",
	Long: `Reset your last-checked timestamps so the next 'check' command
will show all active listings as if it's your first time.`,
	RunE: runReset,
}

func init() {
	rootCmd.AddCommand(resetCmd)
}

func runReset(cmd *cobra.Command, args []string) error {
	if noColor {
		os.Setenv("NO_COLOR", "1")
	}

	state, err := store.Load()
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}

	state.LastChecked = make(map[string]int64)

	if err := store.Save(state); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	successStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#34D399"))
	fmt.Printf("\n  %s Last-checked timestamps have been reset.\n", successStyle.Render("✓"))
	fmt.Println("  Next 'check' will show all active listings.")
	fmt.Println()

	return nil
}
