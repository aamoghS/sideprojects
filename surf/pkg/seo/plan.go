package seo

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

type AuditRecord struct {
	URL    string `json:"url"`
	Score  int    `json:"score"`
	Title  string `json:"title,omitempty"`
	Issues []string `json:"issues,omitempty"`
	Fixes  []Fix  `json:"fixes,omitempty"`
	Error  string `json:"error,omitempty"`
}

type planItem struct {
	Fix
	URLCount int
	URLs     []string
}

func LoadAuditRecords(path string) ([]AuditRecord, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ReadAuditRecords(f)
}

func ReadAuditRecords(r io.Reader) ([]AuditRecord, error) {
	var out []AuditRecord
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var rec AuditRecord
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, sc.Err()
}

func BuildFixPlan(records []AuditRecord) []planItem {
	byID := make(map[string]*planItem)
	for _, rec := range records {
		if rec.Error != "" {
			continue
		}
		for _, fix := range rec.Fixes {
			item, ok := byID[fix.ID]
			if !ok {
				cp := fix
				item = &planItem{Fix: cp, URLs: []string{}}
				byID[fix.ID] = item
			}
			item.URLCount++
			if len(item.URLs) < 5 {
				item.URLs = append(item.URLs, rec.URL)
			}
		}
	}

	out := make([]planItem, 0, len(byID))
	for _, item := range byID {
		out = append(out, *item)
	}
	sort.Slice(out, func(i, j int) bool {
		pi := priorityRank(out[i].Priority)
		pj := priorityRank(out[j].Priority)
		if pi != pj {
			return pi < pj
		}
		if out[i].URLCount != out[j].URLCount {
			return out[i].URLCount > out[j].URLCount
		}
		return out[i].Impact > out[j].Impact
	})
	return out
}

func priorityRank(p string) int {
	switch p {
	case "high":
		return 0
	case "medium":
		return 1
	default:
		return 2
	}
}

func PrintFixPlan(records []AuditRecord, w io.Writer) {
	plan := BuildFixPlan(records)
	if len(plan) == 0 {
		fmt.Fprintln(w, "No fixes needed — all audited URLs look clean.")
		return
	}

	scored := 0
	total := 0
	for _, rec := range records {
		if rec.Error != "" {
			continue
		}
		scored++
		total += rec.Score
	}
	if scored > 0 {
		fmt.Fprintf(w, "Site average score: %d/100 across %d URLs\n\n", total/scored, scored)
	}

	fmt.Fprintln(w, "Prioritized fix plan (do these in order):")
	fmt.Fprintln(w, strings.Repeat("-", 72))
	for i, item := range plan {
		fmt.Fprintf(w, "%d. [%s] %s (%d pages, +%d pts each)\n", i+1, strings.ToUpper(item.Priority), item.Issue, item.URLCount, item.Impact)
		fmt.Fprintf(w, "   Action: %s\n", item.Action)
		fmt.Fprintf(w, "   Example URLs: %s\n\n", strings.Join(item.URLs, ", "))
	}
}

func WriteFixPlan(inputJSONL, outputPath string) error {
	records, err := LoadAuditRecords(inputJSONL)
	if err != nil {
		return err
	}
	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()
	PrintFixPlan(records, f)
	return nil
}
