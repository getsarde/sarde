package plugin

import (
	"testing"

	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/engine"
)

func boolPtr(b bool) *bool { return &b }

// lintLines is a test helper: computes the fence mask the way LintPages does.
func lintLines(lines []string) ([]string, []bool) {
	return lines, fencedLines(lines)
}

func TestContentLint_HeadingLength(t *testing.T) {
	lines, fenced := lintLines([]string{
		"# Short",
		"## This heading is way too long for the configured maximum length limit",
		"Normal text",
	})
	issues := checkHeadingLength(lines, fenced, 20)

	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}
	if issues[0].Line != 2 {
		t.Errorf("expected line 2, got %d", issues[0].Line)
	}
}

func TestContentLint_HeadingIncrement(t *testing.T) {
	issues := checkHeadingIncrement(lintLines([]string{
		"# Title",
		"### Skipped h2",
		"## Back to h2",
		"#### Skipped h3",
	}))

	if len(issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(issues))
	}
	if issues[0].Line != 2 {
		t.Errorf("first issue at line %d, want 2", issues[0].Line)
	}
	if issues[1].Line != 4 {
		t.Errorf("second issue at line %d, want 4", issues[1].Line)
	}
}

func TestContentLint_ImageAlt(t *testing.T) {
	issues := checkImageAlt(lintLines([]string{
		"![Good alt](image.png)",
		"![](missing-alt.png)",
		"Text with ![](inline.png) image",
	}))

	if len(issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(issues))
	}
}

func TestContentLint_EmptyLinks(t *testing.T) {
	issues := checkEmptyLinks(lintLines([]string{
		"[Good link](url)",
		"[](empty-text.html)",
		"![](image.png)", // should NOT match — it's an image
	}))

	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}
	if issues[0].Line != 2 {
		t.Errorf("expected line 2, got %d", issues[0].Line)
	}
}

func TestContentLint_FencedBlocksSkipped(t *testing.T) {
	lines, fenced := lintLines([]string{
		"```md",
		"![](in-backtick-fence.png)",
		"[](in-backtick-fence.html)",
		"```",
		"~~~",
		"![](in-tilde-fence.png)",
		"~~~",
		"````",
		"![](in-long-fence.png)",
		"`````", // closing fence may be longer than the opener
		"![](real-violation.png)",
	})

	imgIssues := checkImageAlt(lines, fenced)
	if len(imgIssues) != 1 {
		t.Fatalf("expected 1 image issue (outside fences), got %d: %v", len(imgIssues), imgIssues)
	}
	if imgIssues[0].Line != 11 {
		t.Errorf("image issue at line %d, want 11", imgIssues[0].Line)
	}
	if linkIssues := checkEmptyLinks(lines, fenced); len(linkIssues) != 0 {
		t.Errorf("expected 0 empty-link issues, got %d: %v", len(linkIssues), linkIssues)
	}
}

func TestContentLint_UnclosedFenceSkippedToEOF(t *testing.T) {
	lines, fenced := lintLines([]string{
		"Intro text",
		"```",
		"![](never-closed.png)",
		"[](never-closed.html)",
	})

	if issues := checkImageAlt(lines, fenced); len(issues) != 0 {
		t.Errorf("expected 0 image issues in unclosed fence, got %d", len(issues))
	}
	if issues := checkEmptyLinks(lines, fenced); len(issues) != 0 {
		t.Errorf("expected 0 link issues in unclosed fence, got %d", len(issues))
	}
}

func TestContentLint_InlineCodeSpansSkipped(t *testing.T) {
	lines, fenced := lintLines([]string{
		"Use `![](path)` for images.",
		"Use ``![](path)`` in double-backtick spans.",
		"The `![](x)` example plus a real ![](violation.png) image.",
		"An unmatched ` backtick with ![](also-real.png).",
	})

	issues := checkImageAlt(lines, fenced)
	if len(issues) != 2 {
		t.Fatalf("expected 2 image issues (real ones only), got %d: %v", len(issues), issues)
	}
	if issues[0].Line != 3 || issues[1].Line != 4 {
		t.Errorf("issues at lines %d, %d; want 3, 4", issues[0].Line, issues[1].Line)
	}
}

func TestContentLint_HeadingRulesSkipFences(t *testing.T) {
	lines, fenced := lintLines([]string{
		"## Real h2",
		"```python",
		"# this comment would look like an overlong h1 heading if fences were not masked",
		"#### fake heading",
		"```",
		"### Real h3",
	})

	if issues := checkHeadingLength(lines, fenced, 30); len(issues) != 0 {
		t.Errorf("expected 0 heading-length issues, got %d: %v", len(issues), issues)
	}
	// h2 -> h3 is a valid increment; the fence content must not perturb state.
	if issues := checkHeadingIncrement(lines, fenced); len(issues) != 0 {
		t.Errorf("expected 0 heading-increment issues, got %d: %v", len(issues), issues)
	}
}

func TestContentLint_LineNumbersIncludeFrontmatterOffset(t *testing.T) {
	page := &engine.Page{
		PageIdentity: engine.PageIdentity{Title: "Offset", FilePath: "content/offset.md"},
		PageContent:  engine.PageContent{RawContent: "Fine line\n![](no-alt.png)\n", FrontmatterLines: 5},
	}
	warnings := LintPages([]*engine.Page{page}, config.ContentLintSettings{
		Rules: config.ContentLintRules{ImageAltRequired: boolPtr(true)},
	})

	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(warnings))
	}
	// Violation is on RawContent line 2; with 5 frontmatter lines the on-disk
	// line is 7.
	if want := "line 7: image missing alt text"; warnings[0].Message != want {
		t.Errorf("message = %q, want %q", warnings[0].Message, want)
	}
}

func TestContentLint_FrontmatterRequired(t *testing.T) {
	page := &engine.Page{
		PageIdentity: engine.PageIdentity{Title: "Has Title"},
		PageMeta:     engine.PageMeta{Description: ""},
		Params:       map[string]any{"author": "Alice"},
	}

	issues := checkFrontmatterRequired(page, []string{"title", "description", "author", "category"})

	if len(issues) != 2 {
		t.Fatalf("expected 2 issues (description, category), got %d", len(issues))
	}
}

func TestContentLint_Disabled(t *testing.T) {
	outDir := t.TempDir()
	var warnings []engine.ValidationWarning

	ctx := &BuildDoneContext{
		Config: &config.SiteConfig{
			ContentLint: config.ContentLintSettings{
				Enabled: boolPtr(false),
				Rules: config.ContentLintRules{
					HeadingMaxLength: 10,
				},
			},
		},
		OutputDir: outDir,
		Pages: []*engine.Page{
			{
				PageIdentity: engine.PageIdentity{Title: "Test"},
				PageContent:  engine.PageContent{RawContent: "# This heading is very long and would normally fail"},
			},
		},
	}
	ctx.SetWarnings(&warnings)

	if err := contentLintBuildDone(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings when disabled, got %d", len(warnings))
	}
}

func TestContentLint_IncrementalLintsOnlyChangedPages(t *testing.T) {
	violating := &engine.Page{
		PageIdentity: engine.PageIdentity{Title: "Bad", FilePath: "content/bad.md"},
		PageContent:  engine.PageContent{RawContent: "![](no-alt.png)\n"},
	}
	clean := &engine.Page{
		PageIdentity: engine.PageIdentity{Title: "Clean", FilePath: "content/clean.md"},
		PageContent:  engine.PageContent{RawContent: "# Clean\n\nNo issues here.\n"},
	}
	cfg := &config.SiteConfig{
		ContentLint: config.ContentLintSettings{
			Enabled: boolPtr(true),
			Rules:   config.ContentLintRules{ImageAltRequired: boolPtr(true)},
		},
	}

	// Incremental: only ChangedPages are linted, so the violating page in
	// Pages produces no warning.
	var warnings []engine.ValidationWarning
	ctx := &BuildDoneContext{
		Config:       cfg,
		OutputDir:    t.TempDir(),
		Pages:        []*engine.Page{violating, clean},
		ChangedPages: []*engine.Page{clean},
		Incremental:  true,
	}
	ctx.SetWarnings(&warnings)
	if err := contentLintBuildDone(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 0 {
		t.Errorf("incremental lint of clean changed page: got %d warnings, want 0", len(warnings))
	}

	// Full build: all pages are linted.
	warnings = nil
	ctx = &BuildDoneContext{
		Config:    cfg,
		OutputDir: t.TempDir(),
		Pages:     []*engine.Page{violating, clean},
	}
	ctx.SetWarnings(&warnings)
	if err := contentLintBuildDone(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 1 {
		t.Errorf("full lint: got %d warnings, want 1", len(warnings))
	}
}

func TestContentLint_Integration(t *testing.T) {
	outDir := t.TempDir()
	var warnings []engine.ValidationWarning

	ctx := &BuildDoneContext{
		Config: &config.SiteConfig{
			ContentLint: config.ContentLintSettings{
				Enabled: boolPtr(true),
				Rules: config.ContentLintRules{
					HeadingMaxLength: 20,
					HeadingIncrement: boolPtr(true),
					ImageAltRequired: boolPtr(true),
					NoEmptyLinks:     boolPtr(true),
				},
			},
		},
		OutputDir: outDir,
		Pages: []*engine.Page{
			{
				PageIdentity: engine.PageIdentity{Title: "Test Page", FilePath: "content/test.md"},
				PageContent:  engine.PageContent{RawContent: "# OK\n### Skipped h2\n![](no-alt.png)\n[](empty.html)\n"},
			},
		},
	}
	ctx.SetWarnings(&warnings)

	if err := contentLintBuildDone(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Expect: heading increment (1), image alt (1), empty link (1) = 3 warnings.
	if len(warnings) != 3 {
		t.Errorf("expected 3 warnings, got %d", len(warnings))
		for _, w := range warnings {
			t.Logf("  %s: %s", w.File, w.Message)
		}
	}
}
