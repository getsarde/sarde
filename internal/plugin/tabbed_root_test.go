package plugin

import (
	"encoding/json"
	"html/template"
	"strings"
	"testing"

	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/engine"
)

// tabbedRootPages returns a tabbed collection root without a body (it only
// redirects to its first tab) and that tab's overview page.
func tabbedRootPages() []*engine.Page {
	col := &engine.Collection{
		Name:     "courses",
		IsTabbed: true,
		Tabs:     []*engine.DocsTab{{Slug: "web", Permalink: "/courses/web/"}},
	}
	root := &engine.Page{
		PageIdentity:      engine.PageIdentity{Title: "Courses", Kind: engine.KindSection, RelPermalink: "/courses/", Permalink: "/courses/"},
		PageContent:       engine.PageContent{Content: template.HTML("")},
		PageRelationships: engine.PageRelationships{Collection: col},
	}
	col.IndexPage = root
	overview := &engine.Page{
		PageIdentity:      engine.PageIdentity{Title: "Web", Kind: engine.KindSection, RelPermalink: "/courses/web/", Permalink: "/courses/web/"},
		PageContent:       engine.PageContent{Content: template.HTML("<p>Course overview</p>")},
		PageRelationships: engine.PageRelationships{Collection: col},
	}
	return []*engine.Page{root, overview}
}

func TestSitemap_SkipsTabbedRootRedirect(t *testing.T) {
	outDir := t.TempDir()
	var warnings []engine.ValidationWarning
	ctx := &BuildDoneContext{
		Config:    config.Defaults(),
		OutputDir: outDir,
		Site:      &engine.SiteContext{BaseURL: "https://example.com"},
		Pages:     tabbedRootPages(),
	}
	ctx.SetWarnings(&warnings)

	if err := sitemapBuildDone(ctx, nil); err != nil {
		t.Fatalf("sitemapBuildDone failed: %v", err)
	}
	data, err := readTestFile(outDir, "sitemap.xml")
	if err != nil {
		t.Fatalf("reading sitemap.xml: %v", err)
	}
	content := string(data)
	if strings.Contains(content, "<loc>https://example.com/courses/</loc>") {
		t.Error("a tabbed root that only redirects must not be in the sitemap")
	}
	if !strings.Contains(content, "<loc>https://example.com/courses/web/</loc>") {
		t.Error("expected the tab overview in the sitemap")
	}
}

func TestSearch_SkipsTabbedRootRedirect(t *testing.T) {
	outDir := t.TempDir()
	var warnings []engine.ValidationWarning
	ctx := &BuildDoneContext{
		Config:    config.Defaults(),
		OutputDir: outDir,
		Site:      &engine.SiteContext{BaseURL: "https://example.com"},
		Pages:     tabbedRootPages(),
	}
	ctx.SetWarnings(&warnings)

	if err := searchBuildDone(ctx, nil, &searchDocCache{}); err != nil {
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
	foundOverview := false
	for _, d := range docs {
		if d.Title == "Courses" {
			t.Error("a tabbed root that only redirects must not be indexed")
		}
		if d.Title == "Web" {
			foundOverview = true
		}
	}
	if !foundOverview {
		t.Error("expected the tab overview in the search index")
	}
}
