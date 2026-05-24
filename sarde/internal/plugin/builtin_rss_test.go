package plugin

import (
	"strings"
	"testing"
	"time"

	"github.com/frostybee/sarde/internal/config"
	"github.com/frostybee/sarde/internal/engine"
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
					{Title: "Post 1", RelPermalink: "/blog/post-1/", Date: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)},
					{Title: "Post 2", RelPermalink: "/blog/post-2/", Date: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)},
					{Title: "Draft", RelPermalink: "/blog/draft/", Draft: true},
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
			Title:        "Post",
			RelPermalink: "/blog/post/",
			Date:         time.Now(),
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
