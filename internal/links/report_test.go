package links

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGenerateReport_NoFindings(t *testing.T) {
	graph := NewLinkGraph()
	graph.Record(LinkRef{RawDest: "./ok.md", Status: StatusOK, Kind: KindContentRoot})

	result := GenerateReport(ReportInput{
		Graph:    graph,
		Coverage: CoverageSummary{TotalLinks: 1, TotalLanes: 1},
		Config:   LinkCheckConfig{OnBroken: "error", OnBrokenAnchor: "error", ReportFormat: "pretty"},
	})

	if result.HasErrors {
		t.Error("expected no errors")
	}
	if len(result.Findings) != 0 {
		t.Errorf("expected 0 findings, got %d", len(result.Findings))
	}
	if !strings.Contains(result.Summary, "0 broken targets") {
		t.Errorf("unexpected summary: %s", result.Summary)
	}
}

func TestGenerateReport_BrokenTarget(t *testing.T) {
	graph := NewLinkGraph()
	graph.Record(LinkRef{
		FromFile: "content/docs/guide.md",
		RawDest:  "./missing.md",
		Dim:      DimKey{Collection: "docs", Lang: "en"},
		Status:   StatusBrokenTarget,
	})

	result := GenerateReport(ReportInput{
		Graph:    graph,
		Coverage: CoverageSummary{TotalLinks: 1, TotalLanes: 1},
		Config:   LinkCheckConfig{OnBroken: "error", ReportFormat: "pretty"},
	})

	if !result.HasErrors {
		t.Error("expected errors")
	}
	if len(result.Findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(result.Findings))
	}
	if result.Findings[0].Type != FindingBrokenTarget {
		t.Errorf("expected BrokenTarget, got %d", result.Findings[0].Type)
	}
}

func TestGenerateReport_UnverifiedInternal_DefaultWarn(t *testing.T) {
	graph := NewLinkGraph()
	graph.Record(LinkRef{
		FromFile: "content/docs/guide.md",
		RawDest:  "/docs/guide/ath",
		Dim:      DimKey{Collection: "docs", Lang: "en"},
		Kind:     KindContentRoot,
		Status:   StatusUnverified,
	})

	result := GenerateReport(ReportInput{
		Graph:    graph,
		Coverage: CoverageSummary{TotalLinks: 1, TotalLanes: 1},
		// Empty policy → defaults to "warn".
		Config: LinkCheckConfig{ReportFormat: "pretty"},
	})

	if result.HasErrors {
		t.Error("unverified_internal should warn, not error, by default")
	}
	if len(result.Findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(result.Findings))
	}
	if result.Findings[0].Type != FindingUnverifiedInternal {
		t.Errorf("expected FindingUnverifiedInternal, got %s", result.Findings[0].Type.String())
	}
	if result.Findings[0].Policy != "warn" {
		t.Errorf("expected policy warn, got %q", result.Findings[0].Policy)
	}
}

func TestGenerateReport_UnverifiedInternal_StrictError(t *testing.T) {
	graph := NewLinkGraph()
	graph.Record(LinkRef{
		FromFile: "content/docs/guide.md",
		RawDest:  "/docs/guide/ath",
		Status:   StatusUnverified,
	})

	result := GenerateReport(ReportInput{
		Graph:    graph,
		Coverage: CoverageSummary{TotalLinks: 1, TotalLanes: 1},
		Config:   LinkCheckConfig{OnUnverifiedInternal: "error", ReportFormat: "pretty"},
	})

	if !result.HasErrors {
		t.Error("expected errors when on_unverified_internal=error (strict mode)")
	}
}

func TestGenerateReport_UnverifiedInternal_Ignore(t *testing.T) {
	graph := NewLinkGraph()
	graph.Record(LinkRef{RawDest: "/docs/guide/ath", Status: StatusUnverified})

	result := GenerateReport(ReportInput{
		Graph:    graph,
		Coverage: CoverageSummary{TotalLinks: 1, TotalLanes: 1},
		Config:   LinkCheckConfig{OnUnverifiedInternal: "ignore", ReportFormat: "pretty"},
	})

	if len(result.Findings) != 0 {
		t.Errorf("expected 0 findings when ignored, got %d", len(result.Findings))
	}
}

func TestGenerateReport_BrokenAnchor(t *testing.T) {
	graph := NewLinkGraph()
	graph.Record(LinkRef{
		FromFile: "content/docs/guide.md",
		RawDest:  "./intro.md#nonexistent",
		Fragment: "nonexistent",
		Dim:      DimKey{Collection: "docs", Lang: "en"},
		Status:   StatusBrokenAnchor,
	})

	result := GenerateReport(ReportInput{
		Graph:    graph,
		Coverage: CoverageSummary{TotalLinks: 1, TotalLanes: 1},
		Config:   LinkCheckConfig{OnBroken: "error", OnBrokenAnchor: "error", ReportFormat: "pretty"},
	})

	if !result.HasErrors {
		t.Error("expected errors")
	}
	if result.Findings[0].Type != FindingBrokenAnchor {
		t.Errorf("expected BrokenAnchor, got %d", result.Findings[0].Type)
	}
}

func TestGenerateReport_BrokenAnchorWarnPolicy(t *testing.T) {
	graph := NewLinkGraph()
	graph.Record(LinkRef{
		FromFile: "guide.md",
		RawDest:  "./intro.md#bad",
		Fragment: "bad",
		Status:   StatusBrokenAnchor,
	})

	result := GenerateReport(ReportInput{
		Graph:    graph,
		Coverage: CoverageSummary{TotalLinks: 1, TotalLanes: 1},
		Config:   LinkCheckConfig{OnBroken: "error", OnBrokenAnchor: "warn", ReportFormat: "pretty"},
	})

	if result.HasErrors {
		t.Error("expected no errors (anchor is warn)")
	}
	if len(result.Findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(result.Findings))
	}
	if result.Findings[0].Policy != "warn" {
		t.Errorf("expected warn policy, got %q", result.Findings[0].Policy)
	}
}

func TestGenerateReport_IgnorePolicy(t *testing.T) {
	graph := NewLinkGraph()
	graph.Record(LinkRef{RawDest: "./missing.md", Status: StatusBrokenTarget})

	result := GenerateReport(ReportInput{
		Graph:    graph,
		Coverage: CoverageSummary{TotalLinks: 1, TotalLanes: 1},
		Config:   LinkCheckConfig{OnBroken: "ignore", ReportFormat: "pretty"},
	})

	if result.HasErrors {
		t.Error("expected no errors under ignore")
	}
	if len(result.Findings) != 0 {
		t.Errorf("expected 0 findings under ignore, got %d", len(result.Findings))
	}
}

func TestGenerateReport_RelativeLink(t *testing.T) {
	graph := NewLinkGraph()
	graph.Record(LinkRef{
		FromFile: "guide.md",
		RawDest:  "./auth.md",
		Status:   StatusOK,
		Kind:     KindRelative,
	})

	result := GenerateReport(ReportInput{
		Graph:    graph,
		Coverage: CoverageSummary{TotalLinks: 1, TotalLanes: 1},
		Config:   LinkCheckConfig{OnRelativeLinks: "warn", ReportFormat: "pretty"},
	})

	if len(result.Findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(result.Findings))
	}
	if result.Findings[0].Type != FindingRelativeLink {
		t.Errorf("expected RelativeLink, got %s", result.Findings[0].Type.String())
	}
}

func TestGenerateReport_LocalLink(t *testing.T) {
	graph := NewLinkGraph()
	graph.Record(LinkRef{
		FromFile: "guide.md",
		RawDest:  "http://localhost:3000/api",
		Status:   StatusExternal,
	})

	result := GenerateReport(ReportInput{
		Graph:    graph,
		Coverage: CoverageSummary{TotalLinks: 1, TotalLanes: 1},
		Config:   LinkCheckConfig{OnLocalLinks: "warn", ReportFormat: "pretty"},
	})

	if len(result.Findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(result.Findings))
	}
	if result.Findings[0].Type != FindingLocalLink {
		t.Errorf("expected LocalLink, got %s", result.Findings[0].Type.String())
	}
}

func TestGenerateReport_SameSite(t *testing.T) {
	graph := NewLinkGraph()
	graph.Record(LinkRef{
		FromFile: "guide.md",
		RawDest:  "https://example.com/docs/guide/",
		Status:   StatusExternal,
	})

	result := GenerateReport(ReportInput{
		Graph:    graph,
		Coverage: CoverageSummary{TotalLinks: 1, TotalLanes: 1},
		Config:   LinkCheckConfig{SameSitePolicy: "error", ReportFormat: "pretty"},
		SiteURL:  "https://example.com",
	})

	if !result.HasErrors {
		t.Error("expected errors for same-site under error policy")
	}
	if result.Findings[0].Type != FindingSameSite {
		t.Errorf("expected SameSite, got %s", result.Findings[0].Type.String())
	}
}

func TestGenerateReport_SameSiteIgnore(t *testing.T) {
	graph := NewLinkGraph()
	graph.Record(LinkRef{
		FromFile: "guide.md",
		RawDest:  "https://example.com/docs/guide/",
		Status:   StatusExternal,
	})

	result := GenerateReport(ReportInput{
		Graph:    graph,
		Coverage: CoverageSummary{TotalLinks: 1, TotalLanes: 1},
		Config:   LinkCheckConfig{SameSitePolicy: "ignore", ReportFormat: "pretty"},
		SiteURL:  "https://example.com",
	})

	if len(result.Findings) != 0 {
		t.Errorf("expected 0 findings under ignore, got %d", len(result.Findings))
	}
}

func TestGenerateReport_Exclude(t *testing.T) {
	graph := NewLinkGraph()
	graph.Record(LinkRef{
		FromFile: "guide.md",
		RawDest:  "./excluded.md",
		Status:   StatusBrokenTarget,
	})
	graph.Record(LinkRef{
		FromFile: "guide.md",
		RawDest:  "./kept.md",
		Status:   StatusBrokenTarget,
	})

	result := GenerateReport(ReportInput{
		Graph:    graph,
		Coverage: CoverageSummary{TotalLinks: 2, TotalLanes: 1},
		Config:   LinkCheckConfig{OnBroken: "error", Exclude: []string{"./excluded.md"}, ReportFormat: "pretty"},
	})

	if len(result.Findings) != 1 {
		t.Fatalf("expected 1 finding (excluded filtered), got %d", len(result.Findings))
	}
	if result.Findings[0].Ref.RawDest != "./kept.md" {
		t.Errorf("expected './kept.md', got %q", result.Findings[0].Ref.RawDest)
	}
}

// Slash-containing glob patterns must behave identically on every platform
// (path.Match semantics, not filepath.Match).
func TestGenerateReport_ExcludeGlob(t *testing.T) {
	graph := NewLinkGraph()
	graph.Record(LinkRef{
		FromFile: "guide.md",
		RawDest:  "/drafts/wip",
		Status:   StatusBrokenTarget,
	})
	graph.Record(LinkRef{
		FromFile: "guide.md",
		RawDest:  "/docs/kept",
		Status:   StatusBrokenTarget,
	})

	result := GenerateReport(ReportInput{
		Graph:    graph,
		Coverage: CoverageSummary{TotalLinks: 2, TotalLanes: 1},
		Config:   LinkCheckConfig{OnBroken: "error", Exclude: []string{"/drafts/*"}, ReportFormat: "pretty"},
	})

	if len(result.Findings) != 1 {
		t.Fatalf("expected 1 finding (glob-excluded filtered), got %d", len(result.Findings))
	}
	if result.Findings[0].Ref.RawDest != "/docs/kept" {
		t.Errorf("expected '/docs/kept', got %q", result.Findings[0].Ref.RawDest)
	}
}

func TestGenerateReport_JSONFormat(t *testing.T) {
	graph := NewLinkGraph()
	graph.Record(LinkRef{
		FromFile: "guide.md",
		RawDest:  "./missing.md",
		Dim:      DimKey{Collection: "docs", Lang: "en"},
		Status:   StatusBrokenTarget,
	})

	result := GenerateReport(ReportInput{
		Graph:    graph,
		Coverage: CoverageSummary{TotalLinks: 1, TotalLanes: 1},
		Config:   LinkCheckConfig{OnBroken: "error", ReportFormat: "json"},
	})

	var report jsonReport
	if err := json.Unmarshal([]byte(result.Output), &report); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, result.Output)
	}
	if report.Summary.BrokenTargets != 1 {
		t.Errorf("expected 1 broken target in JSON, got %d", report.Summary.BrokenTargets)
	}
	if len(report.Findings) != 1 {
		t.Fatalf("expected 1 finding in JSON, got %d", len(report.Findings))
	}
	if report.Findings[0].Type != "broken_target" {
		t.Errorf("expected 'broken_target', got %q", report.Findings[0].Type)
	}
}

func TestGenerateReport_PrettyGroupByFile(t *testing.T) {
	graph := NewLinkGraph()
	graph.Record(LinkRef{FromFile: "b.md", RawDest: "./m1.md", Status: StatusBrokenTarget})
	graph.Record(LinkRef{FromFile: "a.md", RawDest: "./m2.md", Status: StatusBrokenTarget})
	graph.Record(LinkRef{FromFile: "a.md", RawDest: "./m3.md", Status: StatusBrokenTarget})

	result := GenerateReport(ReportInput{
		Graph:    graph,
		Coverage: CoverageSummary{TotalLinks: 3, TotalLanes: 1},
		Config:   LinkCheckConfig{OnBroken: "error", ReportFormat: "pretty"},
	})

	lines := strings.Split(result.Output, "\n")
	var fileHeaders []string
	for _, l := range lines {
		stripped := strings.TrimSpace(l)
		if stripped != "" && !strings.HasPrefix(stripped, "ERROR") && !strings.HasPrefix(stripped, "WARN") &&
			!strings.HasPrefix(stripped, "link check") && !strings.HasPrefix(stripped, "checked") {
			if !strings.HasPrefix(stripped, "[") {
				fileHeaders = append(fileHeaders, stripped)
			}
		}
	}
	// Files should be sorted: a.md before b.md
	if len(fileHeaders) >= 2 && fileHeaders[0] > fileHeaders[1] {
		t.Errorf("expected alphabetical file ordering, got %v", fileHeaders)
	}
}

func TestGenerateReport_Deduplication(t *testing.T) {
	graph := NewLinkGraph()
	graph.Record(LinkRef{FromFile: "a.md", RawDest: "./same.md", Status: StatusBrokenTarget})
	graph.Record(LinkRef{FromFile: "a.md", RawDest: "./same.md", Status: StatusBrokenTarget})
	graph.Record(LinkRef{FromFile: "a.md", RawDest: "./same.md", Status: StatusBrokenTarget})

	result := GenerateReport(ReportInput{
		Graph:    graph,
		Coverage: CoverageSummary{TotalLinks: 3, TotalLanes: 1},
		Config:   LinkCheckConfig{OnBroken: "error", ReportFormat: "pretty"},
	})

	if !strings.Contains(result.Output, "(x3)") {
		t.Errorf("expected (x3) dedup in output:\n%s", result.Output)
	}
}

func TestIsLocalHref(t *testing.T) {
	tests := []struct {
		href string
		want bool
	}{
		{"http://localhost:3000/api", true},
		{"http://127.0.0.1:8080/", true},
		{"http://0.0.0.0/", true},
		{"https://example.com/", false},
		{"./relative.md", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := isLocalHref(tt.href); got != tt.want {
			t.Errorf("isLocalHref(%q) = %v, want %v", tt.href, got, tt.want)
		}
	}
}

func TestIsSameSiteHref(t *testing.T) {
	tests := []struct {
		href, site string
		want       bool
	}{
		{"https://example.com/docs/", "https://example.com", true},
		{"https://example.com/docs/", "https://example.com/", true},
		{"http://example.com/docs/", "https://example.com", false}, // scheme mismatch
		{"https://other.com/docs/", "https://example.com", false},
		{"./relative.md", "https://example.com", false},
	}
	for _, tt := range tests {
		if got := isSameSiteHref(tt.href, tt.site); got != tt.want {
			t.Errorf("isSameSiteHref(%q, %q) = %v, want %v", tt.href, tt.site, got, tt.want)
		}
	}
}

func TestGenerateReport_ExternalBroken_Warn(t *testing.T) {
	graph := NewLinkGraph()
	graph.Record(LinkRef{
		FromFile: "guide.md",
		RawDest:  "https://dead.example.com/page",
		Status:   StatusExternalBroken,
	})

	result := GenerateReport(ReportInput{
		Graph:    graph,
		Coverage: CoverageSummary{TotalLinks: 1, TotalLanes: 1},
		Config:   LinkCheckConfig{OnExternalBroken: "warn", ReportFormat: "pretty"},
	})

	if result.HasErrors {
		t.Error("expected no errors under warn policy")
	}
	if len(result.Findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(result.Findings))
	}
	if result.Findings[0].Type != FindingExternalBroken {
		t.Errorf("expected ExternalBroken, got %s", result.Findings[0].Type.String())
	}
	if result.Findings[0].Policy != "warn" {
		t.Errorf("expected warn policy, got %q", result.Findings[0].Policy)
	}
}

func TestGenerateReport_ExternalBroken_Error(t *testing.T) {
	graph := NewLinkGraph()
	graph.Record(LinkRef{
		FromFile: "guide.md",
		RawDest:  "https://dead.example.com/page",
		Status:   StatusExternalBroken,
	})

	result := GenerateReport(ReportInput{
		Graph:    graph,
		Coverage: CoverageSummary{TotalLinks: 1, TotalLanes: 1},
		Config:   LinkCheckConfig{OnExternalBroken: "error", ReportFormat: "pretty"},
	})

	if !result.HasErrors {
		t.Error("expected errors under error policy")
	}
}

func TestGenerateReport_ExternalBroken_Ignore(t *testing.T) {
	graph := NewLinkGraph()
	graph.Record(LinkRef{
		FromFile: "guide.md",
		RawDest:  "https://dead.example.com/page",
		Status:   StatusExternalBroken,
	})

	result := GenerateReport(ReportInput{
		Graph:    graph,
		Coverage: CoverageSummary{TotalLinks: 1, TotalLanes: 1},
		Config:   LinkCheckConfig{OnExternalBroken: "ignore", ReportFormat: "pretty"},
	})

	if len(result.Findings) != 0 {
		t.Errorf("expected 0 findings under ignore, got %d", len(result.Findings))
	}
}

func TestFindingTypeString(t *testing.T) {
	tests := []struct {
		ft   FindingType
		want string
	}{
		{FindingBrokenTarget, "broken_target"},
		{FindingBrokenAnchor, "broken_anchor"},
		{FindingRelativeLink, "relative_link"},
		{FindingLocalLink, "local_link"},
		{FindingSameSite, "same_site"},
		{FindingExternalBroken, "external_broken"},
		{FindingUnverifiedInternal, "unverified_internal"},
	}
	for _, tt := range tests {
		if got := tt.ft.String(); got != tt.want {
			t.Errorf("FindingType(%d).String() = %q, want %q", tt.ft, got, tt.want)
		}
	}
}
