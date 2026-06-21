package seo

import (
	"encoding/json"
	"fmt"
	"os"
)

type AuditOptions struct {
	BaseURL  string
	Depth    int
	Output   string
	PlanPath string
}

func RunAudit(opts AuditOptions) error {
	gen, err := NewSitemapGenerator(opts.BaseURL, opts.Depth)
	if err != nil {
		return err
	}
	if err := gen.Crawl(); err != nil {
		return fmt.Errorf("crawl: %w", err)
	}

	urls := gen.GetURLs()
	if len(urls) == 0 {
		return fmt.Errorf("no URLs found at %s", opts.BaseURL)
	}

	out, err := os.Create(opts.Output)
	if err != nil {
		return err
	}
	defer out.Close()

	analyzer := NewAnalyzer()
	for _, raw := range urls {
		rec := AuditRecord{URL: raw}
		result, err := analyzer.Analyze(raw)
		if err != nil {
			rec.Error = err.Error()
		} else {
			rec.Score = result.Score
			rec.Title = result.Title
			rec.Issues = result.Issues
			rec.Fixes = result.Fixes
		}
		data, err := json.Marshal(rec)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintln(out, string(data)); err != nil {
			return err
		}
	}

	if opts.PlanPath != "" {
		records, err := LoadAuditRecords(opts.Output)
		if err != nil {
			return err
		}
		if err := writePlanFile(records, opts.PlanPath); err != nil {
			return err
		}
	}
	return nil
}

func writePlanFile(records []AuditRecord, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	PrintFixPlan(records, f)
	return nil
}
