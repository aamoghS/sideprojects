package seo

import (
	"fmt"
	"io"
	"os"
)

type scoreDelta struct {
	URL    string
	Before int
	After  int
	Delta  int
}

func RunDiff(beforePath, afterPath string, w io.Writer) error {
	before, err := LoadAuditRecords(beforePath)
	if err != nil {
		return err
	}
	after, err := LoadAuditRecords(afterPath)
	if err != nil {
		return err
	}
	return PrintDiff(before, after, w)
}

func PrintDiff(before, after []AuditRecord, w io.Writer) error {
	bmap := mapByURL(before)
	amap := mapByURL(after)

	var deltas []scoreDelta
	improved, regressed, unchanged := 0, 0, 0

	for url, a := range amap {
		if a.Error != "" {
			continue
		}
		b, ok := bmap[url]
		if !ok || b.Error != "" {
			continue
		}
		d := a.Score - b.Score
		deltas = append(deltas, scoreDelta{URL: url, Before: b.Score, After: a.Score, Delta: d})
		switch {
		case d > 0:
			improved++
		case d < 0:
			regressed++
		default:
			unchanged++
		}
	}

	sortDeltas(deltas)

	fmt.Fprintf(w, "Compared %d URLs present in both audits\n", len(deltas))
	fmt.Fprintf(w, "Improved: %d  Regressed: %d  Unchanged: %d\n\n", improved, regressed, unchanged)

	if len(deltas) > 0 {
		fmt.Fprintln(w, "Biggest gains:")
		printTopDeltas(w, deltas, true, 10)
		fmt.Fprintln(w, "\nRegressions (fix these first):")
		printTopDeltas(w, deltas, false, 10)
	}

	fmt.Fprintln(w, "\nFixes cleared:")
	printClearedFixes(w, bmap, amap)
	return nil
}

func mapByURL(records []AuditRecord) map[string]AuditRecord {
	m := make(map[string]AuditRecord, len(records))
	for _, r := range records {
		m[r.URL] = r
	}
	return m
}

func sortDeltas(d []scoreDelta) {
	for i := 0; i < len(d); i++ {
		for j := i + 1; j < len(d); j++ {
			if d[j].Delta > d[i].Delta {
				d[i], d[j] = d[j], d[i]
			}
		}
	}
}

func printTopDeltas(w io.Writer, deltas []scoreDelta, gains bool, limit int) {
	n := 0
	if gains {
		for _, d := range deltas {
			if d.Delta <= 0 {
				continue
			}
			fmt.Fprintf(w, "  %+3d  %d -> %d  %s\n", d.Delta, d.Before, d.After, d.URL)
			n++
			if n >= limit {
				return
			}
		}
		if n == 0 {
			fmt.Fprintln(w, "  (none)")
		}
		return
	}
	for i := len(deltas) - 1; i >= 0; i-- {
		d := deltas[i]
		if d.Delta >= 0 {
			continue
		}
		fmt.Fprintf(w, "  %+3d  %d -> %d  %s\n", d.Delta, d.Before, d.After, d.URL)
		n++
		if n >= limit {
			return
		}
	}
	if n == 0 {
		fmt.Fprintln(w, "  (none)")
	}
}

func printClearedFixes(w io.Writer, before, after map[string]AuditRecord) {
	cleared := 0
	for url, b := range before {
		a, ok := after[url]
		if !ok || b.Error != "" || a.Error != "" {
			continue
		}
		beforeIDs := fixIDs(b.Fixes)
		afterIDs := fixIDs(a.Fixes)
		for id := range beforeIDs {
			if !afterIDs[id] {
				if cleared < 15 {
					fmt.Fprintf(w, "  %s: fixed %s\n", url, id)
				}
				cleared++
			}
		}
	}
	if cleared == 0 {
		fmt.Fprintln(w, "  (none yet — re-run audit after deploying fixes)")
	} else if cleared > 15 {
		fmt.Fprintf(w, "  ... and %d more\n", cleared-15)
	}
}

func fixIDs(fixes []Fix) map[string]bool {
	m := make(map[string]bool, len(fixes))
	for _, f := range fixes {
		m[f.ID] = true
	}
	return m
}

func WriteDiff(beforePath, afterPath, outputPath string) error {
	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()
	return RunDiff(beforePath, afterPath, f)
}
