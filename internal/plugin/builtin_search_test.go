package plugin

import (
	"encoding/json"
	"html/template"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/engine"
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
				PageIdentity:      engine.PageIdentity{Title: "Getting Started", RelPermalink: "/docs/getting-started/", Permalink: "/docs/getting-started/"},
				PageContent:       engine.PageContent{Content: template.HTML("<p>This is the getting started guide.</p>")},
				PageRelationships: engine.PageRelationships{Collection: &engine.Collection{Name: "docs"}},
				PageTaxonomy:      engine.PageTaxonomy{Tags: []string{"tutorial", "beginner"}},
			},
			{
				PageIdentity: engine.PageIdentity{Title: "Draft", RelPermalink: "/docs/draft/"},
				PageMeta:     engine.PageMeta{Draft: true},
			},
		},
	}
	ctx.SetWarnings(&warnings)

	err := searchBuildDone(ctx, nil)
	if err != nil {
		t.Fatalf("searchBuildDone failed: %v", err)
	}

	data, err := readTestFile(outDir, "search-index.en.json")
	if err != nil {
		t.Fatalf("reading search-index.en.json: %v", err)
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
		Pages:     []*engine.Page{{PageIdentity: engine.PageIdentity{Title: "T", RelPermalink: "/t/"}}},
	}
	ctx.SetWarnings(&warnings)

	if err := searchBuildDone(ctx, nil); err != nil {
		t.Fatalf("searchBuildDone: %v", err)
	}

	for _, rel := range []string{"assets/vendor/orama/orama.esm.js", "assets/js/static-search.js", "search-index.en.json"} {
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
		Page:      &engine.Page{PageIdentity: engine.PageIdentity{Title: "X"}},
		RouteData: rd,
	})
	if err != nil {
		t.Fatalf("BeforeRender: %v", err)
	}
	if len(rd.Scripts) != 1 {
		t.Fatalf("expected 1 script appended, got %d: %v", len(rd.Scripts), rd.Scripts)
	}
}

func TestSearch_IncludesDescription(t *testing.T) {
	outDir := t.TempDir()
	var warnings []engine.ValidationWarning
	ctx := &BuildDoneContext{
		Config:    config.Defaults(),
		OutputDir: outDir,
		Pages: []*engine.Page{
			{PageIdentity: engine.PageIdentity{Title: "T", RelPermalink: "/t/"}, PageMeta: engine.PageMeta{Description: "short summary"}},
		},
	}
	ctx.SetWarnings(&warnings)

	if err := searchBuildDone(ctx, nil); err != nil {
		t.Fatal(err)
	}
	data, err := readTestFile(outDir, "search-index.en.json")
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
			{PageIdentity: engine.PageIdentity{Title: "Long", RelPermalink: "/long/"}, PageContent: engine.PageContent{Content: template.HTML(longContent)}},
		},
	}
	ctx.SetWarnings(&warnings)

	cfg := map[string]any{"max_content_length": 100}
	searchBuildDone(ctx, cfg)

	data, err := readTestFile(outDir, "search-index.en.json")
	if err != nil {
		t.Fatalf("reading search-index.en.json: %v", err)
	}
	var docs []searchDocument
	if err := json.Unmarshal(data, &docs); err != nil {
		t.Fatalf("unmarshaling search index: %v", err)
	}
	if len(docs) == 0 {
		t.Fatal("expected at least 1 document in search index")
	}
	if len(docs[0].Content) > 100 {
		t.Errorf("content length = %d, should be truncated to 100", len(docs[0].Content))
	}
}

func TestTruncateRuneSafe(t *testing.T) {
	tests := []struct {
		name string
		s    string
		max  int
		want string
	}{
		{"ascii under max", "hello", 10, "hello"},
		{"ascii exact", "hello", 5, "hello"},
		{"ascii cut", "hello", 3, "hel"},
		{"mid 2-byte rune", "ééé", 3, "é"},      // é = 2 bytes; cut at 3 lands mid-rune
		{"mid 4-byte rune", "😀😀", 6, "😀"},      // 😀 = 4 bytes; cut at 6 lands mid-rune
		{"boundary 2-byte", "ééé", 4, "éé"},     // cut on a rune boundary keeps both
		{"max zero", "héllo", 0, ""},
		{"max negative", "héllo", -5, ""},
		{"empty", "", 5, ""},
	}
	for _, tt := range tests {
		got := truncateRuneSafe(tt.s, tt.max)
		if got != tt.want {
			t.Errorf("%s: truncateRuneSafe(%q, %d) = %q, want %q", tt.name, tt.s, tt.max, got, tt.want)
		}
		if !utf8.ValidString(got) {
			t.Errorf("%s: result %q is not valid UTF-8", tt.name, got)
		}
	}
}

func TestSearch_TruncationRuneSafe(t *testing.T) {
	outDir := t.TempDir()
	var warnings []engine.ValidationWarning

	// 200 two-byte runes → 400 bytes stripped; max_content_length 101 lands
	// mid-rune and must back off to a rune boundary.
	content := "<p>" + strings.Repeat("é", 200) + "</p>"

	ctx := &BuildDoneContext{
		Config:    config.Defaults(),
		OutputDir: outDir,
		Pages: []*engine.Page{
			{PageIdentity: engine.PageIdentity{Title: "Accents", RelPermalink: "/accents/"}, PageContent: engine.PageContent{Content: template.HTML(content)}},
		},
	}
	ctx.SetWarnings(&warnings)

	cfg := map[string]any{"max_content_length": 101}
	if err := searchBuildDone(ctx, cfg); err != nil {
		t.Fatalf("searchBuildDone: %v", err)
	}

	data, err := readTestFile(outDir, "search-index.en.json")
	if err != nil {
		t.Fatalf("reading search-index.en.json: %v", err)
	}
	if !utf8.Valid(data) {
		t.Error("search index file is not valid UTF-8")
	}
	var docs []searchDocument
	if err := json.Unmarshal(data, &docs); err != nil {
		t.Fatalf("unmarshaling search index: %v", err)
	}
	if len(docs) != 1 {
		t.Fatalf("expected 1 document, got %d", len(docs))
	}
	got := docs[0].Content
	if len(got) > 101 {
		t.Errorf("content length = %d, want <= 101", len(got))
	}
	if !utf8.ValidString(got) {
		t.Errorf("truncated content is not valid UTF-8: %q", got)
	}
	if strings.ContainsRune(got, utf8.RuneError) {
		t.Errorf("truncated content contains replacement char: %q", got)
	}
}

func TestBuildBreadcrumb_CycleSafe(t *testing.T) {
	a := &engine.Section{Title: "A"}
	b := &engine.Section{Title: "B"}
	a.Parent = b
	b.Parent = a // deliberate cycle

	page := &engine.Page{
		PageIdentity:      engine.PageIdentity{Title: "P"},
		PageRelationships: engine.PageRelationships{Section: a},
	}
	// The assertion is that this returns at all (depth-capped walk).
	got := buildBreadcrumb(page)
	if got == "" {
		t.Error("expected non-empty breadcrumb")
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
