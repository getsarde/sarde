package plugin

import (
	"testing"

	"github.com/frostybee/sarde/internal/config"
	"github.com/frostybee/sarde/internal/content"
	"github.com/frostybee/sarde/internal/engine"
)

func buildTestContext(pages []*engine.Page, validationData map[string]engine.ValidationEntry, siteURL string) (*BuildDoneContext, *[]engine.ValidationWarning) {
	idx := content.BuildPageIndex(pages)
	for _, p := range pages {
		if len(p.Headings) > 0 {
			ids := make([]string, len(p.Headings))
			for i, h := range p.Headings {
				ids[i] = h.ID
			}
			idx.SetHeadings(p.RelPermalink, ids)
		}
	}

	var warnings []engine.ValidationWarning
	ctx := &BuildDoneContext{
		Config: &config.SiteConfig{
			LinkValidation: config.LinkValidationSettings{},
		},
		Site:           &engine.SiteContext{BaseURL: siteURL},
		PageIndex:      idx,
		ValidationData: validationData,
	}
	ctx.SetWarnings(&warnings)
	return ctx, &warnings
}

func TestLinkValidatorInvalidLink(t *testing.T) {
	pages := []*engine.Page{
		{Slug: "guide", RelPermalink: "/docs/guide/", Permalink: "/docs/guide/"},
	}
	data := map[string]engine.ValidationEntry{
		"/docs/guide/": {
			FilePath: "content/docs/guide.md",
			Links: []engine.CollectedLink{
				{Href: "/nonexistent/"},
				{Href: "/docs/guide/"},
			},
		},
	}

	ctx, warnings := buildTestContext(pages, data, "")
	err := linkValidatorBuildDone(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if len(*warnings) != 1 {
		t.Fatalf("got %d warnings, want 1", len(*warnings))
	}
	if w := (*warnings)[0]; w.Message != "invalid link: /nonexistent/" {
		t.Errorf("warning message = %q, want %q", w.Message, "invalid link: /nonexistent/")
	}
}

func TestLinkValidatorInvalidHash(t *testing.T) {
	pages := []*engine.Page{
		{Slug: "guide", RelPermalink: "/docs/guide/", Permalink: "/docs/guide/",
			Headings: []engine.Heading{{ID: "intro", Text: "Intro", Level: 2}}},
	}
	data := map[string]engine.ValidationEntry{
		"/docs/guide/": {
			FilePath: "content/docs/guide.md",
			Links: []engine.CollectedLink{
				{Href: "/docs/guide/#intro"},
				{Href: "/docs/guide/#nonexistent"},
				{Href: "#intro"},
				{Href: "#missing"},
			},
		},
	}

	ctx, warnings := buildTestContext(pages, data, "")
	linkValidatorBuildDone(ctx)

	if len(*warnings) != 2 {
		t.Fatalf("got %d warnings, want 2 (one cross-page, one same-page)", len(*warnings))
	}
	for _, w := range *warnings {
		if w.Message != "invalid hash: /docs/guide/#nonexistent" && w.Message != "invalid hash: #missing" {
			t.Errorf("unexpected warning: %q", w.Message)
		}
	}
}

func TestLinkValidatorRelativeLink(t *testing.T) {
	data := map[string]engine.ValidationEntry{
		"/docs/guide/": {
			FilePath: "content/docs/guide.md",
			Links: []engine.CollectedLink{
				{Href: "./sibling"},
				{Href: "../parent"},
			},
		},
	}

	ctx, warnings := buildTestContext(nil, data, "")
	linkValidatorBuildDone(ctx)

	if len(*warnings) != 2 {
		t.Fatalf("got %d warnings, want 2", len(*warnings))
	}
	for _, w := range *warnings {
		if !contains(w.Message, "relative link") {
			t.Errorf("expected relative link warning, got %q", w.Message)
		}
	}
}

func TestLinkValidatorRelativeLinksDisabled(t *testing.T) {
	f := false
	data := map[string]engine.ValidationEntry{
		"/page/": {
			FilePath: "content/page.md",
			Links:    []engine.CollectedLink{{Href: "./relative"}},
		},
	}

	ctx, warnings := buildTestContext(nil, data, "")
	ctx.Config.LinkValidation.WarnRelativeLinks = &f
	linkValidatorBuildDone(ctx)

	if len(*warnings) != 0 {
		t.Fatalf("got %d warnings with relative links disabled, want 0", len(*warnings))
	}
}

func TestLinkValidatorLocalLink(t *testing.T) {
	data := map[string]engine.ValidationEntry{
		"/page/": {
			FilePath: "content/page.md",
			Links: []engine.CollectedLink{
				{Href: "http://localhost:4321/foo"},
				{Href: "https://127.0.0.1:8080/bar"},
			},
		},
	}

	ctx, warnings := buildTestContext(nil, data, "")
	linkValidatorBuildDone(ctx)

	if len(*warnings) != 2 {
		t.Fatalf("got %d warnings, want 2", len(*warnings))
	}
	for _, w := range *warnings {
		if !contains(w.Message, "local link") {
			t.Errorf("expected local link warning, got %q", w.Message)
		}
	}
}

func TestLinkValidatorSameSiteWarn(t *testing.T) {
	data := map[string]engine.ValidationEntry{
		"/page/": {
			FilePath: "content/page.md",
			Links:    []engine.CollectedLink{{Href: "https://example.com/docs/guide/"}},
		},
	}

	ctx, warnings := buildTestContext(nil, data, "https://example.com")
	ctx.Config.LinkValidation.SameSitePolicy = "warn"
	linkValidatorBuildDone(ctx)

	if len(*warnings) != 1 {
		t.Fatalf("got %d warnings, want 1", len(*warnings))
	}
	if !contains((*warnings)[0].Message, "same site") {
		t.Errorf("expected same site warning, got %q", (*warnings)[0].Message)
	}
}

func TestLinkValidatorSameSiteValidate(t *testing.T) {
	pages := []*engine.Page{
		{Slug: "guide", RelPermalink: "/docs/guide/", Permalink: "/docs/guide/"},
	}
	data := map[string]engine.ValidationEntry{
		"/page/": {
			FilePath: "content/page.md",
			Links: []engine.CollectedLink{
				{Href: "https://example.com/docs/guide/"},
				{Href: "https://example.com/nonexistent/"},
			},
		},
	}

	ctx, warnings := buildTestContext(pages, data, "https://example.com")
	ctx.Config.LinkValidation.SameSitePolicy = "validate"
	linkValidatorBuildDone(ctx)

	if len(*warnings) != 1 {
		t.Fatalf("got %d warnings, want 1 (only nonexistent)", len(*warnings))
	}
	if !contains((*warnings)[0].Message, "invalid link") {
		t.Errorf("expected invalid link warning, got %q", (*warnings)[0].Message)
	}
}

func TestLinkValidatorExternalSkipped(t *testing.T) {
	data := map[string]engine.ValidationEntry{
		"/page/": {
			FilePath: "content/page.md",
			Links: []engine.CollectedLink{
				{Href: "https://external.com/page"},
				{Href: "http://other.org"},
			},
		},
	}

	ctx, warnings := buildTestContext(nil, data, "")
	linkValidatorBuildDone(ctx)

	if len(*warnings) != 0 {
		t.Fatalf("got %d warnings for external links, want 0", len(*warnings))
	}
}

func TestLinkValidatorSpecialSchemesSkipped(t *testing.T) {
	data := map[string]engine.ValidationEntry{
		"/page/": {
			FilePath: "content/page.md",
			Links: []engine.CollectedLink{
				{Href: "mailto:user@example.com"},
				{Href: "tel:+1234567890"},
				{Href: "data:text/plain;base64,SGVsbG8="},
			},
		},
	}

	ctx, warnings := buildTestContext(nil, data, "")
	linkValidatorBuildDone(ctx)

	if len(*warnings) != 0 {
		t.Fatalf("got %d warnings for special schemes, want 0", len(*warnings))
	}
}

func TestLinkValidatorExclude(t *testing.T) {
	data := map[string]engine.ValidationEntry{
		"/page/": {
			FilePath: "content/page.md",
			Links: []engine.CollectedLink{
				{Href: "/nonexistent/"},
				{Href: "/excluded/"},
			},
		},
	}

	ctx, warnings := buildTestContext(nil, data, "")
	ctx.Config.LinkValidation.Exclude = []string{"/excluded/*", "/excluded/"}
	linkValidatorBuildDone(ctx)

	if len(*warnings) != 1 {
		t.Fatalf("got %d warnings, want 1 (excluded path should be skipped)", len(*warnings))
	}
}

func TestLinkValidatorStaticAsset(t *testing.T) {
	pages := []*engine.Page{
		{Slug: "page", RelPermalink: "/page/", Permalink: "/page/"},
	}
	data := map[string]engine.ValidationEntry{
		"/page/": {
			FilePath: "content/page.md",
			Links: []engine.CollectedLink{
				{Href: "/favicon.ico"},
			},
		},
	}

	ctx, warnings := buildTestContext(pages, data, "")
	// Manually add asset to the index.
	ctx.PageIndex.AddAssets(t.TempDir()) // empty dir, no assets
	// We need to test that HasAsset works; let's test without the asset
	linkValidatorBuildDone(ctx)

	// /favicon.ico is not a page and not an asset -> invalid link
	if len(*warnings) != 1 {
		t.Fatalf("got %d warnings, want 1", len(*warnings))
	}
}

func TestLinkValidatorDisabled(t *testing.T) {
	f := false
	data := map[string]engine.ValidationEntry{
		"/page/": {
			FilePath: "content/page.md",
			Links:    []engine.CollectedLink{{Href: "/broken/"}},
		},
	}

	ctx, warnings := buildTestContext(nil, data, "")
	ctx.Config.LinkValidation.Enabled = &f
	linkValidatorBuildDone(ctx)

	if len(*warnings) != 0 {
		t.Fatalf("got %d warnings with validator disabled, want 0", len(*warnings))
	}
}

func TestLinkValidatorFailBuild(t *testing.T) {
	tr := true
	data := map[string]engine.ValidationEntry{
		"/page/": {
			FilePath: "content/page.md",
			Links:    []engine.CollectedLink{{Href: "/broken/"}},
		},
	}

	ctx, _ := buildTestContext(nil, data, "")
	ctx.Config.LinkValidation.FailBuild = &tr
	err := linkValidatorBuildDone(ctx)

	if err == nil {
		t.Fatal("expected error with fail_build=true, got nil")
	}
	if !contains(err.Error(), "1 broken link") {
		t.Errorf("error message = %q, want to contain '1 broken link'", err.Error())
	}
}

func TestLinkValidatorImagesDisabled(t *testing.T) {
	f := false
	data := map[string]engine.ValidationEntry{
		"/page/": {
			FilePath: "content/page.md",
			Links: []engine.CollectedLink{
				{Href: "/broken-image.png", IsImage: true},
				{Href: "/broken-link/"},
			},
		},
	}

	ctx, warnings := buildTestContext(nil, data, "")
	ctx.Config.LinkValidation.CheckImages = &f
	linkValidatorBuildDone(ctx)

	if len(*warnings) != 1 {
		t.Fatalf("got %d warnings with images disabled, want 1", len(*warnings))
	}
	if !contains((*warnings)[0].Message, "/broken-link/") {
		t.Errorf("expected broken-link warning, got %q", (*warnings)[0].Message)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
