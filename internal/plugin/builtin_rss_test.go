package plugin

import (
	"html/template"
	"strings"
	"testing"
	"time"

	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/engine"
)

func TestRSS_GeneratesFeed(t *testing.T) {
	outDir := t.TempDir()
	var warnings []engine.ValidationWarning

	ctx := &BuildDoneContext{
		Config:    config.Defaults(),
		OutputDir: outDir,
		Site:      &engine.SiteContext{BaseURL: "https://example.com"},
		Collections: map[string]*engine.Collection{
			"blog": {
				Name:   "blog",
				Title:  "Blog",
				Config: &engine.CollectionConfig{Feed: true},
				Pages: []*engine.Page{
					{PageIdentity: engine.PageIdentity{Title: "Post 1", RelPermalink: "/blog/post-1/", Date: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)}},
					{PageIdentity: engine.PageIdentity{Title: "Post 2", RelPermalink: "/blog/post-2/", Date: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)}},
					{PageIdentity: engine.PageIdentity{Title: "Draft", RelPermalink: "/blog/draft/"}, PageMeta: engine.PageMeta{Draft: true}},
				},
			},
		},
	}
	ctx.SetWarnings(&warnings)

	err := rssBuildDone(ctx, nil)
	if err != nil {
		t.Fatalf("rssBuildDone failed: %v", err)
	}

	data, err := readTestFile(outDir, "blog/feed.xml")
	if err != nil {
		t.Fatalf("reading feed.xml: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "<?xml") {
		t.Error("expected XML declaration")
	}
	if !strings.Contains(content, "<title>Blog</title>") {
		t.Error("expected channel title")
	}
	if !strings.Contains(content, "<title>Post 1</title>") {
		t.Error("expected Post 1 item")
	}
	if !strings.Contains(content, "https://example.com/blog/post-1/") {
		t.Error("expected Post 1 link")
	}
	if strings.Contains(content, "Draft") {
		t.Error("draft posts should be excluded")
	}
}

func TestRSS_ItemLimit(t *testing.T) {
	outDir := t.TempDir()
	var warnings []engine.ValidationWarning

	pages := make([]*engine.Page, 30)
	for i := range pages {
		pages[i] = &engine.Page{
			PageIdentity: engine.PageIdentity{
				Title:        "Post",
				RelPermalink: "/blog/post/",
				Date:         time.Now(),
			},
		}
	}

	ctx := &BuildDoneContext{
		Config:    config.Defaults(),
		OutputDir: outDir,
		Site:      &engine.SiteContext{BaseURL: "https://example.com"},
		Collections: map[string]*engine.Collection{
			"blog": {
				Name:   "blog",
				Title:  "Blog",
				Config: &engine.CollectionConfig{Feed: true},
				Pages:  pages,
			},
		},
	}
	ctx.SetWarnings(&warnings)

	cfg := map[string]any{"limit": 5}
	rssBuildDone(ctx, cfg)

	data, _ := readTestFile(outDir, "blog/feed.xml")
	count := strings.Count(string(data), "<item>")
	if count != 5 {
		t.Errorf("expected 5 items, got %d", count)
	}
}

func TestRSS_NoFeedCollections(t *testing.T) {
	outDir := t.TempDir()
	var warnings []engine.ValidationWarning

	ctx := &BuildDoneContext{
		Config:    config.Defaults(),
		OutputDir: outDir,
		Site:      &engine.SiteContext{BaseURL: "https://example.com"},
		Collections: map[string]*engine.Collection{
			"docs": {
				Name:   "docs",
				Title:  "Docs",
				Config: &engine.CollectionConfig{Feed: false},
			},
		},
	}
	ctx.SetWarnings(&warnings)

	// Should not error and not produce a feed.
	err := rssBuildDone(ctx, nil)
	if err != nil {
		t.Fatalf("rssBuildDone failed: %v", err)
	}
}

// On a multi-language site, each language must get its own feed file with only
// that language's posts, at the localized path. Previously all languages were
// mixed into a single blog/feed.xml.
func TestRSS_PerLanguageFeeds(t *testing.T) {
	outDir := t.TempDir()
	var warnings []engine.ValidationWarning

	cfg := config.Defaults()
	cfg.I18n.DefaultLanguage = "en"
	cfg.I18n.Languages = map[string]config.LanguageConfig{"en": {}, "fr": {}}

	resolver := &engine.URLResolver{
		BasePath:    "/",
		BaseURL:     "https://example.com",
		I18nEnabled: true,
		DefaultLang: "en",
		Strategy:    "prefix-except-default",
		Languages:   map[string]bool{"en": true, "fr": true},
	}

	ctx := &BuildDoneContext{
		Config:    cfg,
		OutputDir: outDir,
		Resolver:  resolver,
		Site:      &engine.SiteContext{BaseURL: "https://example.com"},
		Collections: map[string]*engine.Collection{
			"blog": {
				Name:   "blog",
				Title:  "Blog",
				Config: &engine.CollectionConfig{Feed: true},
				Pages: []*engine.Page{
					{PageIdentity: engine.PageIdentity{Title: "English Post", RelPermalink: "/blog/en-post/", Permalink: "/blog/en-post/", Date: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)}, PageI18n: engine.PageI18n{Lang: "en"}},
					{PageIdentity: engine.PageIdentity{Title: "Article Francais", RelPermalink: "/blog/fr-post/", Permalink: "/fr/blog/fr-post/", Date: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)}, PageI18n: engine.PageI18n{Lang: "fr"}},
				},
			},
		},
	}
	ctx.SetWarnings(&warnings)

	if err := rssBuildDone(ctx, nil); err != nil {
		t.Fatalf("rssBuildDone failed: %v", err)
	}

	// Default language keeps the plain path with only English content.
	en, err := readTestFile(outDir, "blog/feed.xml")
	if err != nil {
		t.Fatalf("reading en feed: %v", err)
	}
	if !strings.Contains(string(en), "English Post") {
		t.Error("en feed missing English post")
	}
	if strings.Contains(string(en), "Article Francais") {
		t.Error("en feed must not contain French post")
	}

	// French language gets a localized feed with only French content.
	fr, err := readTestFile(outDir, "fr/blog/feed.xml")
	if err != nil {
		t.Fatalf("reading fr feed at fr/blog/feed.xml: %v", err)
	}
	if !strings.Contains(string(fr), "Article Francais") {
		t.Error("fr feed missing French post")
	}
	if strings.Contains(string(fr), "English Post") {
		t.Error("fr feed must not contain English post")
	}
}

func TestRSS_RenderedFallbackDescription(t *testing.T) {
	pages := []*engine.Page{{
		PageIdentity: engine.PageIdentity{Title: "Post", RelPermalink: "/blog/post/", Date: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)},
		PageContent:  engine.PageContent{Content: template.HTML("<p>Rendered prose &amp; more.</p>")},
	}}
	items := buildRSSItems(pages, "https://example.com", 20)
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Description != "Rendered prose & more." {
		t.Errorf("Description = %q", items[0].Description)
	}
}
