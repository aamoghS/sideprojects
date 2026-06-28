package display

import (
	"fmt"
	"strings"
	"time"

	"github.com/aamoghS/sideprojects/ideas/simplijobs/internal/fetcher"
	"github.com/aamoghS/sideprojects/ideas/simplijobs/internal/models"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

// Color palette
var (
	purple = lipgloss.Color("#A855F7")
	blue   = lipgloss.Color("#60A5FA")
	green  = lipgloss.Color("#34D399")
	cyan   = lipgloss.Color("#22D3EE")
	white  = lipgloss.Color("#F9FAFB")
	dim    = lipgloss.Color("#9CA3AF")
	dimmer = lipgloss.Color("#6B7280")
	bg1    = lipgloss.Color("#111827")
	bg2    = lipgloss.Color("#1F2937")
	border = lipgloss.Color("#374151")
)

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(purple)

	countStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(cyan)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(dim)

	successStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(green)

	emptyStyle = lipgloss.NewStyle().
			Foreground(dimmer).
			Italic(true).
			PaddingLeft(2)

	urlStyle = lipgloss.NewStyle().
			Foreground(blue).
			Underline(true)

	dividerStyle = lipgloss.NewStyle().
			Foreground(border)
)

// truncate shortens a string to max length, appending "…" if truncated.
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 1 {
		return "…"
	}
	return s[:max-1] + "…"
}

// RenderResults displays the job listings with a header summary and styled table.
func RenderResults(listings []models.Listing, source string, lastChecked int64, showAll bool) {
	emoji := fetcher.SourceEmoji(source)
	label := strings.ToLower(fetcher.SourceLabel(source))

	// ── Header ──────────────────────────────────────────
	fmt.Println()
	headerParts := []string{emoji, " "}

	if showAll {
		headerParts = append(headerParts,
			titleStyle.Render("simplijobs"),
			subtitleStyle.Render(" — showing all active "+label+" listings"),
		)
	} else if lastChecked == 0 {
		headerParts = append(headerParts,
			titleStyle.Render("simplijobs"),
			subtitleStyle.Render(" — first check! showing all "+label+" listings"),
		)
	} else {
		lastTime := time.Unix(lastChecked, 0).Format("Jan 2, 2006 3:04 PM")
		headerParts = append(headerParts,
			titleStyle.Render("simplijobs"),
			subtitleStyle.Render(" — "),
			countStyle.Render(fmt.Sprintf("%d", len(listings))),
			subtitleStyle.Render(" new "+label+" listings since "),
			subtitleStyle.Render(lastTime),
		)
	}
	fmt.Println(strings.Join(headerParts, ""))
	fmt.Println(dividerStyle.Render(strings.Repeat("━", 72)))

	// ── Empty state ─────────────────────────────────────
	if len(listings) == 0 {
		fmt.Println()
		fmt.Println(emptyStyle.Render("No new listings found. Check back later!"))
		fmt.Println()
		return
	}

	// ── Build table rows ────────────────────────────────
	rows := make([][]string, 0, len(listings))
	for i, l := range listings {
		rows = append(rows, []string{
			fmt.Sprintf("%d", i+1),
			truncate(l.CompanyName, 22),
			truncate(l.Title, 32),
			truncate(l.LocationString(), 22),
			l.TimeAgo(),
			l.URL,
		})
	}

	// ── Render table ────────────────────────────────────
	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(border)).
		Headers("#", "COMPANY", "ROLE", "LOCATION", "POSTED", "APPLY").
		Rows(rows...).
		StyleFunc(func(row, col int) lipgloss.Style {
			// Header row
			if row == table.HeaderRow {
				return lipgloss.NewStyle().
					Bold(true).
					Foreground(purple).
					Padding(0, 1).
					Align(lipgloss.Left)
			}

			s := lipgloss.NewStyle().Padding(0, 1)

			// Column-specific styling
			switch col {
			case 0: // #
				s = s.Foreground(dimmer).Align(lipgloss.Right)
			case 1: // Company
				s = s.Foreground(white).Bold(true)
			case 2: // Role
				s = s.Foreground(lipgloss.Color("#D1D5DB"))
			case 3: // Location
				s = s.Foreground(blue)
			case 4: // Posted
				s = s.Foreground(green)
			case 5: // Apply URL
				s = s.Foreground(cyan)
			}

			// Alternating row backgrounds
			if row%2 == 0 {
				s = s.Background(bg2)
			} else {
				s = s.Background(bg1)
			}

			return s
		})

	fmt.Println(t)

	// ── Footer ──────────────────────────────────────────
	fmt.Println()
	fmt.Printf("  %s %s listed  ·  Last checked updated to %s\n",
		successStyle.Render("✓"),
		countStyle.Render(fmt.Sprintf("%d", len(listings))),
		subtitleStyle.Render(time.Now().Format("Jan 2, 2006 3:04 PM")),
	)
	fmt.Println()
}

// RenderStatus displays the current state information.
func RenderStatus(lastInternships, lastNewGrad int64) {
	fmt.Println()
	fmt.Println(titleStyle.Render("  simplijobs status"))
	fmt.Println(dividerStyle.Render("  " + strings.Repeat("━", 40)))
	fmt.Println()

	internLabel := "never"
	if lastInternships > 0 {
		internLabel = time.Unix(lastInternships, 0).Format("Jan 2, 2006 3:04 PM")
	}
	newgradLabel := "never"
	if lastNewGrad > 0 {
		newgradLabel = time.Unix(lastNewGrad, 0).Format("Jan 2, 2006 3:04 PM")
	}

	fmt.Printf("  🎓 Internships last checked:  %s\n", subtitleStyle.Render(internLabel))
	fmt.Printf("  💼 New Grad last checked:     %s\n", subtitleStyle.Render(newgradLabel))
	fmt.Println()
}
