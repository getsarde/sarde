package plugin

import (
	"html/template"
	"strings"
	"testing"
	"time"

	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/engine"
)

func TestAtom_GeneratesFeed(t *testing.T) {
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
					{
						PageIdentity: engine.PageIdentity{
							Title:        "Post 1",
							RelPermalink: "/blog/post-1/",
							Date:         time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC),
						},
						Params: map[string]any{"author": "Alice"},
					},
					{
						PageIdentity: engine.PageIdentity{
							Title:        "Draft",
							RelPermalink: "/blog/draft/",
						},
						PageMeta: engine.PageMeta{Draft: true},
					},
				},
			},
		},
	}
	ctx.SetWarnings(&warnings)

	if err := atomBuildDone(ctx, nil); err != nil {
		t.Fatalf("atomBuildDone failed: %v", err)
	}

	data, err := readTestFile(outDir, "blog/atom.xml")
	if err != nil {
		t.Fatalf("reading atom.xml: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, `xmlns="http://www.w3.org/2005/Atom"`) {
		t.Error("expected Atom namespace")
	}
	if !strings.Contains(content, "<title>Blog</title>") {
		t.Error("expected feed title")
	}
	if !strings.Contains(content, "<title>Post 1</title>") {
		t.Error("expected Post 1 entry")
	}
	if !strings.Contains(content, "https://example.com/blog/post-1/") {
		t.Error("expected Post 1 link")
	}
	if !strings.Contains(content, "<name>Alice</name>") {
		t.Error("expected author name")
	}
	if strings.Contains(content, "Draft") {
		t.Error("drafts must be excluded")
	}
}

func TestAtom_Limit(t *testing.T) {
	outDir := t.TempDir()
	var warnings []engine.ValidationWarning

	pages := make([]*engine.Page, 10)
	for i := range pages {
		pages[i] = &engine.Page{
			PageIdentity: engine.PageIdentity{
				Title:        "Post",
				RelPermalink: "/blog/p/",
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

	atomBuildDone(ctx, map[string]any{"limit": 3})

	data, _ := readTestFile(outDir, "blog/atom.xml")
	if got := strings.Count(string(data), "<entry>"); got != 3 {
		t.Errorf("expected 3 entries, got %d", got)
	}
}

func TestAtom_RenderedFallbackSummary(t *testing.T) {
	pages := []*engine.Page{{
		PageIdentity: engine.PageIdentity{Title: "Post", RelPermalink: "/blog/post/", Date: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)},
		PageContent:  engine.PageContent{Content: template.HTML("<p>Rendered prose &amp; more.</p>")},
	}}
	entries := buildAtomEntries(pages, "https://example.com", 20)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Summary != "Rendered prose & more." {
		t.Errorf("Summary = %q", entries[0].Summary)
	}
}

// site.author becomes the feed's author, which Atom requires for entries
// that name none; a page's own author still wins on its entry.
func TestAtom_SiteAuthorIsFeedAuthor(t *testing.T) {
	outDir := t.TempDir()
	var warnings []engine.ValidationWarning
	cfg := config.Defaults()
	cfg.Site.Author = "Field Notes Team"
	ctx := &BuildDoneContext{
		Config:    cfg,
		OutputDir: outDir,
		Site:      &engine.SiteContext{BaseURL: "https://example.com"},
		Collections: map[string]*engine.Collection{
			"blog": {
				Name:   "blog",
				Title:  "Blog",
				Config: &engine.CollectionConfig{Feed: true},
				Pages: []*engine.Page{
					{PageIdentity: engine.PageIdentity{Title: "Unsigned", RelPermalink: "/blog/unsigned/", Date: time.Date(2025, 3, 2, 0, 0, 0, 0, time.UTC)}},
					{PageIdentity: engine.PageIdentity{Title: "Signed", RelPermalink: "/blog/signed/", Date: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)}, Params: map[string]any{"author": "Alice"}},
				},
			},
		},
	}
	ctx.SetWarnings(&warnings)
	if err := atomBuildDone(ctx, nil); err != nil {
		t.Fatal(err)
	}
	data, err := readTestFile(outDir, "blog/atom.xml")
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	feedHead, _, _ := strings.Cut(content, "<entry>")
	if !strings.Contains(feedHead, "<author>") || !strings.Contains(feedHead, "<name>Field Notes Team</name>") {
		t.Errorf("feed-level author missing:\n%s", content)
	}
	if !strings.Contains(content, "<name>Alice</name>") {
		t.Error("the page's own author must stay on its entry")
	}
}
