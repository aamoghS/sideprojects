package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/aamoghS/sideprojects/ideas/simplijobs/internal/display"
	"github.com/aamoghS/sideprojects/ideas/simplijobs/internal/fetcher"
	"github.com/aamoghS/sideprojects/ideas/simplijobs/internal/filter"
	"github.com/aamoghS/sideprojects/ideas/simplijobs/internal/models"
	"github.com/aamoghS/sideprojects/ideas/simplijobs/internal/store"
	"github.com/spf13/cobra"
)

var showAll bool

var checkCmd = &cobra.Command{
	Use:   "check <internships|newgrad>",
	Short: "Check for new job postings",
	Long: `Fetch and display new job postings since your last check.

You must specify a source:
  internships  — Summer 2026 internship listings
  newgrad      — New grad / full-time positions

Examples:
  simplijobs check internships
  simplijobs check newgrad --all
  simplijobs check internships --company "Google"
  simplijobs check newgrad --location "Remote" --limit 10`,
	Args:      cobra.ExactArgs(1),
	ValidArgs: []string{"internships", "newgrad"},
	RunE:      runCheck,
}

func init() {
	checkCmd.Flags().BoolVar(&showAll, "all", false, "Show all active listings, ignoring last-checked time")
	rootCmd.AddCommand(checkCmd)
}

func runCheck(cmd *cobra.Command, args []string) error {
	source := args[0]

	if !fetcher.IsValidSource(source) {
		return fmt.Errorf("invalid source: %q\nValid sources: internships, newgrad", source)
	}

	// Handle --no-color (lipgloss v1 respects NO_COLOR env var)
	if noColor {
		os.Setenv("NO_COLOR", "1")
	}

	// Load local state
	state, err := store.Load()
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}

	lastChecked := state.GetLastChecked(source)

	// Fetch listings from GitHub
	spinnerDone := showSpinner(source)
	listings, err := fetcher.FetchListings(source)
	spinnerDone()
	if err != nil {
		return err
	}

	// Build filter chain
	filters := []func(models.Listing) bool{
		filter.VisibleOnly(),
	}

	if !showAll && lastChecked > 0 {
		filters = append(filters, filter.NewerThan(lastChecked))
	}

	if company != "" {
		filters = append(filters, filter.CompanyContains(company))
	}

	if location != "" {
		filters = append(filters, filter.LocationContains(location))
	}

	// Apply filters
	filtered := filter.Apply(listings, filters...)

	// Sort newest first
	filter.SortByDate(filtered)

	// Apply limit
	if limit > 0 && len(filtered) > limit {
		filtered = filtered[:limit]
	}

	// Display results
	display.RenderResults(filtered, source, lastChecked, showAll)

	// Update last-checked timestamp
	state.SetLastChecked(source, time.Now().Unix())
	if err := store.Save(state); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	return nil
}

// showSpinner displays a simple loading message and returns a function to stop it.
func showSpinner(source string) func() {
	label := fetcher.SourceLabel(source)
	fmt.Fprintf(os.Stderr, "\r  ⏳ Fetching %s listings from GitHub...", label)
	return func() {
		fmt.Fprintf(os.Stderr, "\r%s\r", "                                                  ")
	}
}
