package build

import (
	"cmp"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/getsarde/sarde/embedded"
	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/engine"
)

// TestBuild_ReportData covers the data the CLI build report is printed from:
// live phase callbacks, link warnings with source positions, the link
// summary, the slowest pages, and stable warning order.
func TestBuild_ReportData(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "content/_index.md", "---\ntitle: Home\n---\n# Home\n")
	writeFixture(t, dir, "content/docs/_index.md", "---\ntitle: Docs\n---\n")
	writeFixture(t, dir, "content/docs/a.md", "---\ntitle: A\n---\n# A\n\nSee [x](#nope).\n")
	writeFixture(t, dir, "content/docs/b.md", "---\ntitle: B\n---\n# B\n\nBack to [A](./a.md).\n")

	cfg := config.Defaults()
	cfg.Build.Minify = config.BoolPtr(false)
	cfg.Build.Cache = config.BoolPtr(false)
	cfg.Build.LastUpdated = "false"
	cfg.LinkValidation.Enabled = config.BoolPtr(true)
	cfg.LinkValidation.OnBrokenAnchor = "warn"
	cfg.LinkValidation.OnRelativeLinks = "ignore"

	builder := NewSiteBuilder(BuildOptions{
		ProjectDir:  dir,
		Config:      cfg,
		ThemeConfig: buildThemeConfig(),
		EmbeddedFS:  embedded.ThemeFS(),
	})
	var observed []string
	builder.SetPhaseObserver(func(pt engine.PhaseTiming) {
		observed = append(observed, pt.Phase)
	})

	result, err := builder.Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if got, want := strings.Join(observed, ", "), strings.Join(phaseNames(result.PhaseTimings), ", "); got != want {
		t.Errorf("observer saw\n %s\nbut PhaseTimings has\n %s", got, want)
	}
	if got, want := strings.Join(observed, ", "), strings.Join(FullBuildPhases(), ", "); got != want {
		t.Errorf("observer saw\n %s\nwant\n %s", got, want)
	}

	var linkWarn *engine.ValidationWarning
	for i := range result.Warnings {
		if result.Warnings[i].Field == "link" {
			linkWarn = &result.Warnings[i]
		}
	}
	if linkWarn == nil {
		t.Fatalf("expected a link warning, got %+v", result.Warnings)
	}
	if linkWarn.Line != 6 || linkWarn.Col <= 0 {
		t.Errorf("link warning position = %d:%d, want line 6 and a column", linkWarn.Line, linkWarn.Col)
	}

	if result.Links == nil {
		t.Fatal("expected a link summary")
	}
	if result.Links.BrokenAnchors != 1 || result.Links.Warnings != 1 || result.Links.Errors != 0 {
		t.Errorf("unexpected link summary: %+v", *result.Links)
	}
	if result.Links.Links < 2 {
		t.Errorf("expected at least 2 checked links, got %d", result.Links.Links)
	}

	if n := len(result.SlowestPages); n < 1 || n > slowestPagesCount {
		t.Fatalf("SlowestPages has %d entries, want 1 to %d", n, slowestPagesCount)
	}
	if !slices.IsSortedFunc(result.SlowestPages, func(a, b engine.PageTiming) int {
		return cmp.Compare(b.Duration, a.Duration)
	}) {
		t.Errorf("SlowestPages not slowest first: %+v", result.SlowestPages)
	}
	for _, pt := range result.SlowestPages {
		if filepath.IsAbs(pt.Path) || !strings.HasPrefix(pt.Path, "content/") {
			t.Errorf("SlowestPages path %q should be project-relative", pt.Path)
		}
	}

	if !slices.IsSortedFunc(result.Warnings, func(a, b engine.ValidationWarning) int {
		return cmp.Or(cmp.Compare(a.File, b.File), cmp.Compare(a.Line, b.Line), cmp.Compare(a.Col, b.Col))
	}) {
		t.Errorf("warnings not sorted: %+v", result.Warnings)
	}
}

func TestSlowestPages(t *testing.T) {
	projectDir := t.TempDir()
	a := &engine.Page{PageIdentity: engine.PageIdentity{FilePath: filepath.Join(projectDir, "content", "a.md")}}
	b := &engine.Page{PageIdentity: engine.PageIdentity{FilePath: filepath.Join(projectDir, "content", "b.md")}}
	tags := &engine.Page{PageIdentity: engine.PageIdentity{RelPermalink: "/tags/"}}
	md := []time.Duration{3 * time.Millisecond, time.Millisecond, 0}
	rendered := []RenderedPage{
		{Page: b, renderTime: 5 * time.Millisecond},
		{Page: tags, renderTime: 2 * time.Millisecond},
	}

	got := slowestPages(projectDir, []*engine.Page{a, b, tags}, md, rendered, 5)
	want := []engine.PageTiming{
		{Path: "content/b.md", Duration: 6 * time.Millisecond},
		{Path: "content/a.md", Duration: 3 * time.Millisecond},
		{Path: "/tags/", Duration: 2 * time.Millisecond},
	}
	if !slices.Equal(got, want) {
		t.Errorf("slowestPages = %+v, want %+v", got, want)
	}

	if top := slowestPages(projectDir, []*engine.Page{a, b, tags}, md, rendered, 1); len(top) != 1 || top[0].Path != "content/b.md" {
		t.Errorf("top 1 = %+v, want content/b.md only", top)
	}
}
