package plugin

import (
	"testing"

	"github.com/frostybee/sarde/internal/config"
	"github.com/frostybee/sarde/internal/engine"
)

func boolPtr(b bool) *bool { return &b }

func TestContentLint_HeadingLength(t *testing.T) {
	issues := checkHeadingLength([]string{
		"# Short",
		"## This heading is way too long for the configured maximum length limit",
		"Normal text",
	}, 20)

	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}
	if issues[0].Line != 2 {
		t.Errorf("expected line 2, got %d", issues[0].Line)
	}
}

func TestContentLint_HeadingIncrement(t *testing.T) {
	issues := checkHeadingIncrement([]string{
		"# Title",
		"### Skipped h2",
		"## Back to h2",
		"#### Skipped h3",
	})

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
	issues := checkImageAlt([]string{
		"![Good alt](image.png)",
		"![](missing-alt.png)",
		"Text with ![](inline.png) image",
	})

	if len(issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(issues))
	}
}

func TestContentLint_EmptyLinks(t *testing.T) {
	issues := checkEmptyLinks([]string{
		"[Good link](url)",
		"[](empty-text.html)",
		"![](image.png)", // should NOT match — it's an image
	})

	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}
	if issues[0].Line != 2 {
		t.Errorf("expected line 2, got %d", issues[0].Line)
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
