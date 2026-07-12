package plugin

import (
	"testing"

	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/content"
	"github.com/getsarde/sarde/internal/engine"
)

func buildTestContext(pages []*engine.Page, validationData map[string]engine.ValidationEntry, siteURL string) (*BuildDoneContext, *[]engine.ValidationWarning) {
	idx := content.BuildPageIndex(pages)
	for _, p := range pages {
		if len(p.Headings) > 0 {
			ids := make([]string, len(p.Headings))
			for i, h := range p.Headings {
				ids[i] = h.ID
			}
			idx.SetHeadings(p.Permalink, ids)
		}
	}

	changedPages := make([]*engine.Page, 0, len(validationData))
	for permalink := range validationData {
		changedPages = append(changedPages, &engine.Page{
			PageIdentity: engine.PageIdentity{Permalink: permalink},
		})
	}

	collections := make(map[string]*engine.Collection)
	for _, p := range pages {
		if p.Collection != nil {
			collections[p.Collection.Name] = p.Collection
		}
	}

	var warnings []engine.ValidationWarning
	ctx := &BuildDoneContext{
		Config: &config.SiteConfig{
			LinkValidation: config.LinkValidationSettings{},
		},
		Site:           &engine.SiteContext{BaseURL: siteURL},
		Resolver:       &engine.URLResolver{BasePath: "/"},
		PageIndex:      idx,
		ValidationData: validationData,
		Collections:    collections,
		Incremental:    true,
		ChangedPages:   changedPages,
	}
	ctx.SetWarnings(&warnings)
	return ctx, &warnings
}

func TestLinkValidatorInvalidLink(t *testing.T) {
	pages := []*engine.Page{
		{PageIdentity: engine.PageIdentity{Slug: "guide", RelPermalink: "/docs/guide/", Permalink: "/docs/guide/"}},
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
		{
			PageIdentity: engine.PageIdentity{Slug: "guide", RelPermalink: "/docs/guide/", Permalink: "/docs/guide/"},
			PageContent:  engine.PageContent{Headings: []engine.Heading{{ID: "intro", Text: "Intro", Level: 2}}},
		},
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
	docsCol := &engine.Collection{Name: "docs", Config: &engine.CollectionConfig{}}
	source := &engine.Page{
		PageIdentity:      engine.PageIdentity{Slug: "guide", RelPermalink: "/docs/guide/", Permalink: "/docs/guide/"},
		PageI18n:          engine.PageI18n{LangRelPath: "docs/guide.md"},
		PageRelationships: engine.PageRelationships{Collection: docsCol},
	}
	data := map[string]engine.ValidationEntry{
		"/docs/guide/": {
			FilePath: "content/docs/guide.md",
			Links: []engine.CollectedLink{
				{Href: "./sibling.md"},
				{Href: "../parent.md"},
			},
		},
	}

	ctx, warnings := buildTestContext([]*engine.Page{source}, data, "")
	linkValidatorBuildDone(ctx)

	if len(*warnings) != 2 {
		t.Fatalf("got %d warnings, want 2 (both relative targets are missing)", len(*warnings))
	}
	for _, w := range *warnings {
		if !contains(w.Message, "invalid link") {
			t.Errorf("expected invalid link warning for missing relative target, got %q", w.Message)
		}
	}
}

func TestLinkValidatorRelativeLinksDisabled(t *testing.T) {
	docsCol := &engine.Collection{Name: "docs", Config: &engine.CollectionConfig{}}
	source := &engine.Page{
		PageIdentity:      engine.PageIdentity{Slug: "page", RelPermalink: "/docs/page/", Permalink: "/docs/page/"},
		PageI18n:          engine.PageI18n{LangRelPath: "docs/page.md"},
		PageRelationships: engine.PageRelationships{Collection: docsCol},
	}
	data := map[string]engine.ValidationEntry{
		"/docs/page/": {
			FilePath: "content/page.md",
			Links:    []engine.CollectedLink{{Href: "./relative.md"}},
		},
	}

	ctx, warnings := buildTestContext([]*engine.Page{source}, data, "")
	ctx.Config.LinkValidation.OnRelativeLinks = "ignore"
	linkValidatorBuildDone(ctx)

	// Relative links are now resolved through the canonical resolver. With the
	// target missing, the link is flagged as broken regardless of the
	// on_relative_links config (that policy controls the full-build report, not
	// the incremental plugin's resolution). This test verifies the plugin does
	// not crash when on_relative_links is "ignore".
	if len(*warnings) != 1 {
		t.Fatalf("got %d warnings, want 1 (missing relative target)", len(*warnings))
	}
}

func TestLinkValidatorRelativeResolvesWhenTargetExists(t *testing.T) {
	docsCol := &engine.Collection{Name: "docs", Config: &engine.CollectionConfig{}}
	source := &engine.Page{
		PageIdentity:      engine.PageIdentity{Slug: "guide", RelPermalink: "/docs/guide/", Permalink: "/docs/guide/"},
		PageI18n:          engine.PageI18n{LangRelPath: "docs/guide.md"},
		PageRelationships: engine.PageRelationships{Collection: docsCol},
	}
	target := &engine.Page{
		PageIdentity:      engine.PageIdentity{Slug: "sibling", RelPermalink: "/docs/sibling/", Permalink: "/docs/sibling/"},
		PageI18n:          engine.PageI18n{LangRelPath: "docs/sibling.md"},
		PageRelationships: engine.PageRelationships{Collection: docsCol},
	}
	data := map[string]engine.ValidationEntry{
		"/docs/guide/": {
			FilePath: "content/docs/guide.md",
			Links:    []engine.CollectedLink{{Href: "./sibling.md"}},
		},
	}

	ctx, warnings := buildTestContext([]*engine.Page{source, target}, data, "")
	linkValidatorBuildDone(ctx)

	if len(*warnings) != 0 {
		t.Fatalf("got %d warnings, want 0 (relative target exists)", len(*warnings))
	}
}

func TestLinkValidatorLocalLink(t *testing.T) {
	source := &engine.Page{
		PageIdentity: engine.PageIdentity{Slug: "page", RelPermalink: "/page/", Permalink: "/page/"},
	}
	data := map[string]engine.ValidationEntry{
		"/page/": {
			FilePath: "content/page.md",
			Links: []engine.CollectedLink{
				{Href: "http://localhost:4321/foo"},
				{Href: "https://127.0.0.1:8080/bar"},
			},
		},
	}

	ctx, warnings := buildTestContext([]*engine.Page{source}, data, "")
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
	source := &engine.Page{
		PageIdentity: engine.PageIdentity{Slug: "page", RelPermalink: "/page/", Permalink: "/page/"},
	}
	data := map[string]engine.ValidationEntry{
		"/page/": {
			FilePath: "content/page.md",
			Links:    []engine.CollectedLink{{Href: "https://example.com/docs/guide/"}},
		},
	}

	ctx, warnings := buildTestContext([]*engine.Page{source}, data, "https://example.com")
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
	source := &engine.Page{
		PageIdentity: engine.PageIdentity{Slug: "page", RelPermalink: "/page/", Permalink: "/page/"},
	}
	target := &engine.Page{
		PageIdentity: engine.PageIdentity{Slug: "guide", RelPermalink: "/docs/guide/", Permalink: "/docs/guide/"},
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

	ctx, warnings := buildTestContext([]*engine.Page{source, target}, data, "https://example.com")
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
	source := &engine.Page{
		PageIdentity: engine.PageIdentity{Slug: "page", RelPermalink: "/page/", Permalink: "/page/"},
	}
	data := map[string]engine.ValidationEntry{
		"/page/": {
			FilePath: "content/page.md",
			Links: []engine.CollectedLink{
				{Href: "https://external.com/page"},
				{Href: "http://other.org"},
			},
		},
	}

	ctx, warnings := buildTestContext([]*engine.Page{source}, data, "")
	linkValidatorBuildDone(ctx)

	if len(*warnings) != 0 {
		t.Fatalf("got %d warnings for external links, want 0", len(*warnings))
	}
}

func TestLinkValidatorSpecialSchemesSkipped(t *testing.T) {
	source := &engine.Page{
		PageIdentity: engine.PageIdentity{Slug: "page", RelPermalink: "/page/", Permalink: "/page/"},
	}
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

	ctx, warnings := buildTestContext([]*engine.Page{source}, data, "")
	linkValidatorBuildDone(ctx)

	if len(*warnings) != 0 {
		t.Fatalf("got %d warnings for special schemes, want 0", len(*warnings))
	}
}

func TestLinkValidatorExclude(t *testing.T) {
	source := &engine.Page{
		PageIdentity: engine.PageIdentity{Slug: "page", RelPermalink: "/page/", Permalink: "/page/"},
	}
	data := map[string]engine.ValidationEntry{
		"/page/": {
			FilePath: "content/page.md",
			Links: []engine.CollectedLink{
				{Href: "/nonexistent/"},
				{Href: "/excluded/"},
			},
		},
	}

	ctx, warnings := buildTestContext([]*engine.Page{source}, data, "")
	ctx.Config.LinkValidation.Exclude = []string{"/excluded/*", "/excluded/"}
	linkValidatorBuildDone(ctx)

	if len(*warnings) != 1 {
		t.Fatalf("got %d warnings, want 1 (excluded path should be skipped)", len(*warnings))
	}
}

func TestLinkValidatorStaticAsset(t *testing.T) {
	source := &engine.Page{
		PageIdentity: engine.PageIdentity{Slug: "page", RelPermalink: "/page/", Permalink: "/page/"},
	}
	data := map[string]engine.ValidationEntry{
		"/page/": {
			FilePath: "content/page.md",
			Links: []engine.CollectedLink{
				{Href: "/favicon.ico"},
			},
		},
	}

	ctx, warnings := buildTestContext([]*engine.Page{source}, data, "")
	linkValidatorBuildDone(ctx)

	// /favicon.ico has a file extension, so the canonical resolver treats it as
	// a static asset (StatusExternal). No warning is expected.
	if len(*warnings) != 0 {
		t.Fatalf("got %d warnings, want 0 (static asset should not be flagged)", len(*warnings))
	}
}

func TestLinkValidatorDisabled(t *testing.T) {
	f := false
	source := &engine.Page{
		PageIdentity: engine.PageIdentity{Slug: "page", RelPermalink: "/page/", Permalink: "/page/"},
	}
	data := map[string]engine.ValidationEntry{
		"/page/": {
			FilePath: "content/page.md",
			Links:    []engine.CollectedLink{{Href: "/broken/"}},
		},
	}

	ctx, warnings := buildTestContext([]*engine.Page{source}, data, "")
	ctx.Config.LinkValidation.Enabled = &f
	linkValidatorBuildDone(ctx)

	if len(*warnings) != 0 {
		t.Fatalf("got %d warnings with validator disabled, want 0", len(*warnings))
	}
}

func TestLinkValidatorFailBuild(t *testing.T) {
	tr := true
	source := &engine.Page{
		PageIdentity: engine.PageIdentity{Slug: "page", RelPermalink: "/page/", Permalink: "/page/"},
	}
	data := map[string]engine.ValidationEntry{
		"/page/": {
			FilePath: "content/page.md",
			Links:    []engine.CollectedLink{{Href: "/broken/"}},
		},
	}

	ctx, _ := buildTestContext([]*engine.Page{source}, data, "")
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
	source := &engine.Page{
		PageIdentity: engine.PageIdentity{Slug: "page", RelPermalink: "/page/", Permalink: "/page/"},
	}
	data := map[string]engine.ValidationEntry{
		"/page/": {
			FilePath: "content/page.md",
			Links: []engine.CollectedLink{
				{Href: "/broken-image.png", IsImage: true},
				{Href: "/broken-link/"},
			},
		},
	}

	ctx, warnings := buildTestContext([]*engine.Page{source}, data, "")
	ctx.Config.LinkValidation.CheckImages = &f
	linkValidatorBuildDone(ctx)

	if len(*warnings) != 1 {
		t.Fatalf("got %d warnings with images disabled, want 1", len(*warnings))
	}
	if !contains((*warnings)[0].Message, "/broken-link/") {
		t.Errorf("expected broken-link warning, got %q", (*warnings)[0].Message)
	}
}

func TestLinkValidatorWithBasePath(t *testing.T) {
	pages := []*engine.Page{
		{PageIdentity: engine.PageIdentity{Slug: "auth", RelPermalink: "/docs/guide/auth/", Permalink: "/swarm/docs/guide/auth/"}},
		{
			PageIdentity: engine.PageIdentity{Slug: "intro", RelPermalink: "/docs/guide/intro/", Permalink: "/swarm/docs/guide/intro/"},
			PageContent:  engine.PageContent{Headings: []engine.Heading{{ID: "overview", Text: "Overview", Level: 2}}},
		},
	}
	data := map[string]engine.ValidationEntry{
		"/swarm/docs/guide/auth/": {
			FilePath: "content/docs/guide/auth.md",
			Links: []engine.CollectedLink{
				{Href: "/docs/guide/intro"},
				{Href: "/docs/guide/intro#overview"},
				{Href: "/docs/guide/nonexistent"},
			},
		},
	}

	idx := content.BuildPageIndex(pages)
	for _, p := range pages {
		if len(p.Headings) > 0 {
			ids := make([]string, len(p.Headings))
			for i, h := range p.Headings {
				ids[i] = h.ID
			}
			idx.SetHeadings(p.Permalink, ids)
		}
	}

	var warnings []engine.ValidationWarning
	ctx := &BuildDoneContext{
		Config: &config.SiteConfig{
			LinkValidation: config.LinkValidationSettings{},
		},
		Site:           &engine.SiteContext{},
		Resolver:       &engine.URLResolver{BasePath: "/swarm/"},
		PageIndex:      idx,
		ValidationData: data,
		Collections:    make(map[string]*engine.Collection),
		Incremental:    true,
		ChangedPages:   []*engine.Page{{PageIdentity: engine.PageIdentity{Permalink: "/swarm/docs/guide/auth/"}}},
	}
	ctx.SetWarnings(&warnings)

	err := linkValidatorBuildDone(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if len(warnings) != 1 {
		t.Fatalf("got %d warnings, want 1 (only nonexistent); warnings: %v", len(warnings), warnings)
	}
	if !contains(warnings[0].Message, "/docs/guide/nonexistent") {
		t.Errorf("expected warning about nonexistent, got %q", warnings[0].Message)
	}
}

func TestLinkValidatorContentRootResolvesInCollection(t *testing.T) {
	docsCol := &engine.Collection{Name: "docs", Config: &engine.CollectionConfig{}}
	source := &engine.Page{
		PageIdentity:      engine.PageIdentity{Slug: "intro", RelPermalink: "/docs/guide/intro/", Permalink: "/docs/guide/intro/"},
		PageI18n:          engine.PageI18n{LangRelPath: "docs/guide/intro.md"},
		PageRelationships: engine.PageRelationships{Collection: docsCol},
	}
	target := &engine.Page{
		PageIdentity:      engine.PageIdentity{Slug: "auth", RelPermalink: "/docs/guide/auth/", Permalink: "/docs/guide/auth/"},
		PageI18n:          engine.PageI18n{LangRelPath: "docs/guide/auth.md"},
		PageRelationships: engine.PageRelationships{Collection: docsCol},
	}
	data := map[string]engine.ValidationEntry{
		"/docs/guide/intro/": {
			FilePath: "content/docs/guide/intro.md",
			Links: []engine.CollectedLink{
				{Href: "/guide/auth"},
			},
		},
	}

	ctx, warnings := buildTestContext([]*engine.Page{source, target}, data, "")
	linkValidatorBuildDone(ctx)

	if len(*warnings) != 0 {
		t.Fatalf("got %d warnings, want 0 (content-root link should resolve in collection); warnings: %v", len(*warnings), *warnings)
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

func TestLinkValidatorSkipsFullBuild(t *testing.T) {
	source := &engine.Page{
		PageIdentity: engine.PageIdentity{Slug: "guide", RelPermalink: "/docs/guide/", Permalink: "/docs/guide/"},
	}
	data := map[string]engine.ValidationEntry{
		"/docs/guide/": {
			FilePath: "content/docs/guide.md",
			Links:    []engine.CollectedLink{{Href: "./sibling.md"}},
		},
	}
	ctx, warnings := buildTestContext([]*engine.Page{source}, data, "")
	ctx.Config.LinkValidation.OnRelativeLinks = "warn"
	ctx.Incremental = false // full build

	if err := linkValidatorBuildDone(ctx); err != nil {
		t.Fatal(err)
	}
	if len(*warnings) != 0 {
		t.Errorf("plugin must not emit warnings on a full build, got %d: %v", len(*warnings), *warnings)
	}
}

func TestLinkValidatorSameSiteError(t *testing.T) {
	source := &engine.Page{
		PageIdentity: engine.PageIdentity{Slug: "page", RelPermalink: "/page/", Permalink: "/page/"},
	}
	data := map[string]engine.ValidationEntry{
		"/page/": {
			FilePath: "content/page.md",
			Links:    []engine.CollectedLink{{Href: "https://example.com/docs/guide/"}},
		},
	}
	ctx, warnings := buildTestContext([]*engine.Page{source}, data, "https://example.com")
	ctx.Config.LinkValidation.SameSitePolicy = "error"
	ctx.Config.LinkValidation.FailBuild = boolPtr(true)

	err := linkValidatorBuildDone(ctx)
	if len(*warnings) != 1 || !contains((*warnings)[0].Message, "same site") {
		t.Fatalf("expected one same-site finding, got %v", *warnings)
	}
	if err == nil {
		t.Error("same_site_policy error with fail_build must fail the build")
	}
}

func TestLinkValidatorIncrementalScopesToChangedPages(t *testing.T) {
	changed := &engine.Page{
		PageIdentity: engine.PageIdentity{Slug: "changed", RelPermalink: "/changed/", Permalink: "/changed/"},
	}
	unchanged := &engine.Page{
		PageIdentity: engine.PageIdentity{Slug: "unchanged", RelPermalink: "/unchanged/", Permalink: "/unchanged/"},
	}
	data := map[string]engine.ValidationEntry{
		"/changed/": {
			FilePath: "content/changed.md",
			Links:    []engine.CollectedLink{{Href: "/broken/"}},
		},
		"/unchanged/": {
			FilePath: "content/unchanged.md",
			Links:    []engine.CollectedLink{{Href: "/also-broken/"}},
		},
	}

	idx := content.BuildPageIndex([]*engine.Page{changed, unchanged})
	var warnings []engine.ValidationWarning
	ctx := &BuildDoneContext{
		Config: &config.SiteConfig{
			LinkValidation: config.LinkValidationSettings{},
		},
		Site:           &engine.SiteContext{},
		Resolver:       &engine.URLResolver{BasePath: "/"},
		PageIndex:      idx,
		ValidationData: data,
		Collections:    make(map[string]*engine.Collection),
		Incremental:    true,
		ChangedPages:   []*engine.Page{{PageIdentity: engine.PageIdentity{Permalink: "/changed/"}}},
	}
	ctx.SetWarnings(&warnings)

	if err := linkValidatorBuildDone(ctx); err != nil {
		t.Fatal(err)
	}

	if len(warnings) != 1 {
		t.Fatalf("got %d warnings, want 1 (only the changed page's broken link)", len(warnings))
	}
	if !contains(warnings[0].Message, "/broken/") {
		t.Errorf("expected warning about /broken/, got %q", warnings[0].Message)
	}
}
