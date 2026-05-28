package plugin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/frostybee/sarde/internal/config"
	"github.com/frostybee/sarde/internal/engine"
)

func TestLlmsTxt_BasicOutput(t *testing.T) {
	outDir := t.TempDir()
	var warnings []engine.ValidationWarning

	ctx := &BuildDoneContext{
		Config: &config.SiteConfig{
			Site: config.SiteIdentity{Description: "A test site"},
		},
		Site: &engine.SiteContext{
			Title:   "My Site",
			BaseURL: "https://example.com",
		},
		OutputDir: outDir,
		Pages: []*engine.Page{
			{PageIdentity: engine.PageIdentity{Title: "About", RelPermalink: "/about/", Kind: engine.KindPage}},
			{PageIdentity: engine.PageIdentity{Title: "Contact", RelPermalink: "/contact/", Kind: engine.KindPage}},
		},
	}
	ctx.SetWarnings(&warnings)

	if err := llmsTxtBuildDone(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(outDir, "llms.txt"))
	if err != nil {
		t.Fatalf("llms.txt not written: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "# My Site") {
		t.Error("missing site title")
	}
	if !strings.Contains(content, "> A test site") {
		t.Error("missing description")
	}
	if !strings.Contains(content, "- [About](https://example.com/about/)") {
		t.Error("missing About page link")
	}
	if !strings.Contains(content, "- [Contact](https://example.com/contact/)") {
		t.Error("missing Contact page link")
	}
}

func TestLlmsTxt_ExcludeBlog(t *testing.T) {
	outDir := t.TempDir()
	var warnings []engine.ValidationWarning

	blogCol := &engine.Collection{Name: "blog"}
	docsCol := &engine.Collection{Name: "docs"}

	enabled := true
	includeBlog := false

	ctx := &BuildDoneContext{
		Config: &config.SiteConfig{
			LlmsTxt: config.LlmsTxtSettings{
				Enabled:     &enabled,
				IncludeBlog: &includeBlog,
			},
		},
		Site:      &engine.SiteContext{Title: "Test"},
		OutputDir: outDir,
		Pages: []*engine.Page{
			{PageIdentity: engine.PageIdentity{Title: "Getting Started", RelPermalink: "/docs/getting-started/", Kind: engine.KindPage}, PageRelationships: engine.PageRelationships{Collection: docsCol}},
			{PageIdentity: engine.PageIdentity{Title: "Hello World", RelPermalink: "/blog/hello-world/", Kind: engine.KindPage}, PageRelationships: engine.PageRelationships{Collection: blogCol}},
		},
	}
	ctx.SetWarnings(&warnings)

	if err := llmsTxtBuildDone(ctx); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(filepath.Join(outDir, "llms.txt"))
	content := string(data)

	if !strings.Contains(content, "Getting Started") {
		t.Error("docs page should be included")
	}
	if strings.Contains(content, "Hello World") {
		t.Error("blog page should be excluded when include_blog is false")
	}
}

func TestLlmsTxt_Disabled(t *testing.T) {
	outDir := t.TempDir()
	var warnings []engine.ValidationWarning

	enabled := false
	ctx := &BuildDoneContext{
		Config: &config.SiteConfig{
			LlmsTxt: config.LlmsTxtSettings{Enabled: &enabled},
		},
		Site:      &engine.SiteContext{Title: "Test"},
		OutputDir: outDir,
		Pages:     []*engine.Page{{PageIdentity: engine.PageIdentity{Title: "Page", RelPermalink: "/page/", Kind: engine.KindPage}}},
	}
	ctx.SetWarnings(&warnings)

	if err := llmsTxtBuildDone(ctx); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(outDir, "llms.txt")); !os.IsNotExist(err) {
		t.Error("llms.txt should not be written when disabled")
	}
}

func TestLlmsTxt_SkipsSectionsAndDrafts(t *testing.T) {
	outDir := t.TempDir()
	var warnings []engine.ValidationWarning

	ctx := &BuildDoneContext{
		Config:    &config.SiteConfig{},
		Site:      &engine.SiteContext{Title: "Test"},
		OutputDir: outDir,
		Pages: []*engine.Page{
			{PageIdentity: engine.PageIdentity{Title: "Home", RelPermalink: "/", Kind: engine.KindHome}},
			{PageIdentity: engine.PageIdentity{Title: "Docs", RelPermalink: "/docs/", Kind: engine.KindSection}},
			{PageIdentity: engine.PageIdentity{Title: "Draft", RelPermalink: "/draft/", Kind: engine.KindPage}, PageMeta: engine.PageMeta{Draft: true}},
			{PageIdentity: engine.PageIdentity{Title: "Visible", RelPermalink: "/visible/", Kind: engine.KindPage}},
		},
	}
	ctx.SetWarnings(&warnings)

	if err := llmsTxtBuildDone(ctx); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(filepath.Join(outDir, "llms.txt"))
	content := string(data)

	if strings.Contains(content, "Home") {
		t.Error("home page should be excluded")
	}
	if strings.Contains(content, "[Docs]") {
		t.Error("section pages should be excluded")
	}
	if strings.Contains(content, "Draft") {
		t.Error("drafts should be excluded")
	}
	if !strings.Contains(content, "Visible") {
		t.Error("visible page should be included")
	}
}
