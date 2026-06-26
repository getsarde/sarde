package plugin

import (
	"strings"
	"testing"
	"time"

	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/engine"
)

func TestSitemap_GeneratesXML(t *testing.T) {
	outDir := t.TempDir()
	var warnings []engine.ValidationWarning

	ctx := &BuildDoneContext{
		Config:    config.Defaults(),
		OutputDir: outDir,
		Site:      &engine.SiteContext{BaseURL: "https://example.com"},
		Pages: []*engine.Page{
			{PageIdentity: engine.PageIdentity{Title: "Home", RelPermalink: "/", Date: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}},
			{PageIdentity: engine.PageIdentity{Title: "About", RelPermalink: "/about/", Date: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)}},
			{PageIdentity: engine.PageIdentity{Title: "Draft", RelPermalink: "/draft/"}, PageMeta: engine.PageMeta{Draft: true}},
		},
	}
	ctx.SetWarnings(&warnings)

	err := sitemapBuildDone(ctx, nil)
	if err != nil {
		t.Fatalf("sitemapBuildDone failed: %v", err)
	}

	data, err := readTestFile(outDir, "sitemap.xml")
	if err != nil {
		t.Fatalf("reading sitemap.xml: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "<?xml") {
		t.Error("expected XML declaration")
	}
	if !strings.Contains(content, "https://example.com/") {
		t.Error("expected home URL")
	}
	if !strings.Contains(content, "https://example.com/about/") {
		t.Error("expected about URL")
	}
	if strings.Contains(content, "/draft/") {
		t.Error("draft pages should be excluded")
	}
	if !strings.Contains(content, "<changefreq>weekly</changefreq>") {
		t.Error("expected default changefreq")
	}
}

func TestSitemap_ExcludePatterns(t *testing.T) {
	outDir := t.TempDir()
	var warnings []engine.ValidationWarning

	ctx := &BuildDoneContext{
		Config:    config.Defaults(),
		OutputDir: outDir,
		Site:      &engine.SiteContext{BaseURL: "https://example.com"},
		Pages: []*engine.Page{
			{PageIdentity: engine.PageIdentity{Title: "Home", RelPermalink: "/"}},
			{PageIdentity: engine.PageIdentity{Title: "Secret", RelPermalink: "/secret/"}},
		},
	}
	ctx.SetWarnings(&warnings)

	cfg := map[string]any{"exclude": []any{"/secret/"}}
	err := sitemapBuildDone(ctx, cfg)
	if err != nil {
		t.Fatalf("sitemapBuildDone failed: %v", err)
	}

	data, _ := readTestFile(outDir, "sitemap.xml")
	if strings.Contains(string(data), "/secret/") {
		t.Error("excluded page should not appear in sitemap")
	}
}

func TestSitemap_ExcludesPaginatedURLs(t *testing.T) {
	outDir := t.TempDir()
	var warnings []engine.ValidationWarning

	ctx := &BuildDoneContext{
		Config:    config.Defaults(),
		OutputDir: outDir,
		Site:      &engine.SiteContext{BaseURL: "https://example.com"},
		Pages: []*engine.Page{
			{PageIdentity: engine.PageIdentity{Title: "Blog", RelPermalink: "/blog/"}},
			{PageIdentity: engine.PageIdentity{Title: "Blog Page 2", RelPermalink: "/blog/page/2/"}},
			{PageIdentity: engine.PageIdentity{Title: "Blog Page 3", RelPermalink: "/blog/page/3/"}},
		},
	}
	ctx.SetWarnings(&warnings)

	err := sitemapBuildDone(ctx, nil)
	if err != nil {
		t.Fatalf("sitemapBuildDone failed: %v", err)
	}

	data, _ := readTestFile(outDir, "sitemap.xml")
	content := string(data)
	if !strings.Contains(content, "/blog/") {
		t.Error("expected blog index URL in sitemap")
	}
	if strings.Contains(content, "/blog/page/2/") {
		t.Error("paginated URL /blog/page/2/ should be excluded from sitemap")
	}
	if strings.Contains(content, "/blog/page/3/") {
		t.Error("paginated URL /blog/page/3/ should be excluded from sitemap")
	}
}
