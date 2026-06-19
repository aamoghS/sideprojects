package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Global flags
	noColor  bool
	limit    int
	company  string
	location string
)

var rootCmd = &cobra.Command{
	Use:   "simplijobs",
	Short: "Track new job postings from SimplifyJobs GitHub",
	Long: `simplijobs is a CLI tool that fetches job listings from the SimplifyJobs
GitHub repositories and shows you only the new postings since your last check.

Sources:
  internships  — Summer 2026 internship listings
  newgrad      — New grad / full-time positions

Usage:
  simplijobs check internships    Check for new internship listings
  simplijobs check newgrad        Check for new grad listings
  simplijobs check newgrad --all  Show all active listings
  simplijobs status               Show when you last checked
  simplijobs reset                Reset your last-checked timestamp`,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable colored output")
	rootCmd.PersistentFlags().IntVar(&limit, "limit", 0, "Limit output to N listings (0 = unlimited)")
	rootCmd.PersistentFlags().StringVar(&company, "company", "", "Filter by company name (substring match)")
	rootCmd.PersistentFlags().StringVar(&location, "location", "", "Filter by location (substring match)")
}
