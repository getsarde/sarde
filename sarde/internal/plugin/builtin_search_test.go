package plugin

import (
	"encoding/json"
	"html/template"
	"testing"

	"github.com/frostybee/sarde/internal/config"
	"github.com/frostybee/sarde/internal/engine"
)

func TestSearch_GeneratesIndex(t *testing.T) {
	outDir := t.TempDir()
	var warnings []engine.ValidationWarning

	ctx := &BuildDoneContext{
		Config:    config.Defaults(),
		OutputDir: outDir,
		Site:      &engine.SiteContext{BaseURL: "https://example.com"},
		Pages: []*engine.Page{
			{
				Title:        "Getting Started",
				RelPermalink: "/docs/getting-started/",
				Content:      template.HTML("<p>This is the getting started guide.</p>"),
				Collection:   &engine.Collection{Name: "docs"},
				Tags:         []string{"tutorial", "beginner"},
			},
			{
				Title:        "Draft",
				RelPermalink: "/docs/draft/",
				Draft:        true,
			},
		},
	}
	ctx.SetWarnings(&warnings)

	err := searchBuildDone(ctx, nil)
	if err != nil {
		t.Fatalf("searchBuildDone failed: %v", err)
	}

	data, err := readTestFile(outDir, "search-index.json")
	if err != nil {
		t.Fatalf("reading search-index.json: %v", err)
	}

	var docs []searchDocument
	if err := json.Unmarshal(data, &docs); err != nil {
		t.Fatalf("unmarshaling search index: %v", err)
	}

	if len(docs) != 1 {
		t.Fatalf("expected 1 document (draft excluded), got %d", len(docs))
	}

	doc := docs[0]
	if doc.Title != "Getting Started" {
		t.Errorf("Title = %q", doc.Title)
	}
	if doc.URL != "/docs/getting-started/" {
		t.Errorf("URL = %q", doc.URL)
	}
	if doc.ID != "/docs/getting-started/" {
		t.Errorf("ID = %q, want RelPermalink", doc.ID)
	}
	if doc.Section != "docs" {
		t.Errorf("Section = %q", doc.Section)
	}
	if len(doc.Tags) != 2 {
		t.Errorf("Tags count = %d, want 2", len(doc.Tags))
	}
	// Content should have HTML stripped.
	if doc.Content != "This is the getting started guide." {
		t.Errorf("Content = %q (should be stripped)", doc.Content)
	}
}

func TestSearch_BuildDoneWritesVendorAssets(t *testing.T) {
	outDir := t.TempDir()
	var warnings []engine.ValidationWarning
	ctx := &BuildDoneContext{
		Config:    config.Defaults(),
		OutputDir: outDir,
		Pages:     []*engine.Page{{Title: "T", RelPermalink: "/t/"}},
	}
	ctx.SetWarnings(&warnings)

	if err := searchBuildDone(ctx, nil); err != nil {
		t.Fatalf("searchBuildDone: %v", err)
	}

	for _, rel := range []string{"assets/vendor/orama/orama.esm.js", "assets/js/static-search.js", "search-index.json"} {
		if _, err := readTestFile(outDir, rel); err != nil {
			t.Errorf("missing %s: %v", rel, err)
		}
	}
}

func TestSearch_BeforeRenderAppendsScripts(t *testing.T) {
	p := newSearchPlugin(nil)
	if p.Hooks.BeforeRender == nil {
		t.Fatal("search plugin should register BeforeRender hook")
	}
	rd := &engine.RouteData{}
	err := p.Hooks.BeforeRender(&BeforeRenderContext{
		Page:      &engine.Page{Title: "X"},
		RouteData: rd,
	})
	if err != nil {
		t.Fatalf("BeforeRender: %v", err)
	}
	if len(rd.Scripts) != 2 {
		t.Fatalf("expected 2 scripts appended, got %d: %v", len(rd.Scripts), rd.Scripts)
	}
}

func TestSearch_IncludesDescription(t *testing.T) {
	outDir := t.TempDir()
	var warnings []engine.ValidationWarning
	ctx := &BuildDoneContext{
		Config:    config.Defaults(),
		OutputDir: outDir,
		Pages: []*engine.Page{
			{Title: "T", Description: "short summary", RelPermalink: "/t/"},
		},
	}
	ctx.SetWarnings(&warnings)

	if err := searchBuildDone(ctx, nil); err != nil {
		t.Fatal(err)
	}
	data, err := readTestFile(outDir, "search-index.json")
	if err != nil {
		t.Fatal(err)
	}
	var docs []searchDocument
	if err := json.Unmarshal(data, &docs); err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 || docs[0].Description != "short summary" {
		t.Errorf("docs[0].Description = %q, want %q", docs[0].Description, "short summary")
	}
}

func TestSearch_ContentTruncation(t *testing.T) {
	outDir := t.TempDir()
	var warnings []engine.ValidationWarning

	longContent := "<p>" + string(make([]byte, 10000)) + "</p>"

	ctx := &BuildDoneContext{
		Config:    config.Defaults(),
		OutputDir: outDir,
		Pages: []*engine.Page{
			{Title: "Long", RelPermalink: "/long/", Content: template.HTML(longContent)},
		},
	}
	ctx.SetWarnings(&warnings)

	cfg := map[string]any{"max_content_length": 100}
	searchBuildDone(ctx, cfg)

	data, _ := readTestFile(outDir, "search-index.json")
	var docs []searchDocument
	json.Unmarshal(data, &docs)

	if len(docs[0].Content) > 100 {
		t.Errorf("content length = %d, should be truncated to 100", len(docs[0].Content))
	}
}

func TestStripHTML(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"<p>Hello</p>", "Hello"},
		{"<h1>Title</h1><p>Body</p>", "Title Body"},
		{"plain text", "plain text"},
		{"<a href=\"/\">link</a>", "link"},
	}

	for _, tt := range tests {
		got := stripHTML(tt.input)
		if got != tt.want {
			t.Errorf("stripHTML(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
