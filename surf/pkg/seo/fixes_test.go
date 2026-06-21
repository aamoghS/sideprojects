package seo

import (
	"bytes"
	"strings"
	"testing"
)

func TestBuildFixes_missingTitle(t *testing.T) {
	r := &AnalysisResult{
		URL:    "https://example.com",
		Issues: []string{"Missing title tag"},
	}
	fixes := BuildFixes(r)
	if len(fixes) != 2 {
		t.Fatalf("len(fixes) = %d, want 2 (title + canonical)", len(fixes))
	}
	if fixes[0].ID != "title-missing" {
		t.Errorf("fixes[0].ID = %q, want title-missing", fixes[0].ID)
	}
}

func TestBuildFixes_canonicalMismatch(t *testing.T) {
	r := &AnalysisResult{
		URL:          "https://example.com/page",
		CanonicalURL: "https://example.com/other",
	}
	fixes := BuildFixes(r)
	if len(fixes) != 1 {
		t.Fatalf("len(fixes) = %d, want 1", len(fixes))
	}
	if fixes[0].ID != "canonical-mismatch" {
		t.Errorf("fixes[0].ID = %q, want canonical-mismatch", fixes[0].ID)
	}
}

func TestBuildFixPlan_prioritizesHighImpact(t *testing.T) {
	records := []AuditRecord{
		{
			URL: "https://a.com/1",
			Fixes: []Fix{
				{ID: "title-missing", Priority: "high", Impact: 15, Issue: "Missing title tag", Action: "add title"},
			},
		},
		{
			URL: "https://a.com/2",
			Fixes: []Fix{
				{ID: "title-missing", Priority: "high", Impact: 15, Issue: "Missing title tag", Action: "add title"},
				{ID: "open-graph", Priority: "low", Impact: 5, Issue: "Missing Open Graph tags", Action: "add og"},
			},
		},
	}
	plan := BuildFixPlan(records)
	if len(plan) != 2 {
		t.Fatalf("len(plan) = %d, want 2", len(plan))
	}
	if plan[0].ID != "title-missing" {
		t.Errorf("plan[0].ID = %q, want title-missing first", plan[0].ID)
	}
	if plan[0].URLCount != 2 {
		t.Errorf("plan[0].URLCount = %d, want 2", plan[0].URLCount)
	}
}

func TestPrintDiff_scoreDelta(t *testing.T) {
	before := []AuditRecord{{URL: "https://x.com", Score: 60}}
	after := []AuditRecord{{URL: "https://x.com", Score: 75, Fixes: []Fix{{ID: "title-missing"}}}}
	before[0].Fixes = []Fix{{ID: "title-missing"}, {ID: "meta-missing"}}

	var buf bytes.Buffer
	if err := PrintDiff(before, after, &buf); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "Improved: 1") {
		t.Errorf("expected improved count in output: %s", buf.String())
	}
}
