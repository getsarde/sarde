package build

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/getsarde/sarde/embedded"
	"github.com/getsarde/sarde/internal/config"
)

func TestBuild_Plugins_SitemapAndRobots(t *testing.T) {
	projDir := createFixtureSite(t)
	cfg := config.Defaults()
	cfg.Site.URL = "https://example.com"
	cfg.Plugins.Enabled = []string{"sitemap", "robots"}
	themeCfg := buildThemeConfig()

	builder := NewSiteBuilder(BuildOptions{
		ProjectDir:  projDir,
		Config:      cfg,
		ThemeConfig: themeCfg,
		EmbeddedFS:  embedded.ThemeFS(),
	})

	result, err := builder.Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	_ = result

	distDir := filepath.Join(projDir, "dist")

	// Sitemap should exist.
	sitemapData, err := os.ReadFile(filepath.Join(distDir, "sitemap.xml"))
	if err != nil {
		t.Fatalf("sitemap.xml not found: %v", err)
	}
	sitemap := string(sitemapData)

	if !strings.Contains(sitemap, "<?xml") {
		t.Error("expected XML declaration in sitemap")
	}
	if !strings.Contains(sitemap, "https://example.com/") {
		t.Error("expected base URL in sitemap")
	}
	if !strings.Contains(sitemap, "<changefreq>") {
		t.Error("expected changefreq in sitemap")
	}

	// Robots should exist.
	robotsData, err := os.ReadFile(filepath.Join(distDir, "robots.txt"))
	if err != nil {
		t.Fatalf("robots.txt not found: %v", err)
	}
	robots := string(robotsData)

	if !strings.Contains(robots, "User-agent: *") {
		t.Error("expected User-agent in robots.txt")
	}
	if !strings.Contains(robots, "Sitemap: https://example.com/sitemap.xml") {
		t.Error("expected Sitemap line in robots.txt")
	}
}

func TestBuild_Plugins_SEO(t *testing.T) {
	projDir := createFixtureSite(t)
	cfg := config.Defaults()
	cfg.Site.URL = "https://example.com"
	cfg.Site.Title = "My Site"
	cfg.Plugins.Enabled = []string{"seo"}
	cfg.Plugins.Config = map[string]map[string]any{
		"seo": {"twitter_handle": "@test"},
	}
	themeCfg := buildThemeConfig()

	builder := NewSiteBuilder(BuildOptions{
		ProjectDir:  projDir,
		Config:      cfg,
		ThemeConfig: themeCfg,
		EmbeddedFS:  embedded.ThemeFS(),
	})

	_, err := builder.Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	distDir := filepath.Join(projDir, "dist")

	// Check that a page has OG meta tags.
	postHTML := readFixture(t, distDir, "blog/hello-world/index.html")

	if !strings.Contains(postHTML, `og:title`) {
		t.Error("expected og:title meta tag")
	}
	if !strings.Contains(postHTML, `og:url`) {
		t.Error("expected og:url meta tag")
	}
	if !strings.Contains(postHTML, `twitter:card`) {
		t.Error("expected twitter:card meta tag")
	}
}

func TestBuild_Plugins_Search(t *testing.T) {
	projDir := createFixtureSite(t)
	cfg := config.Defaults()
	cfg.Plugins.Enabled = []string{"search"}
	themeCfg := buildThemeConfig()

	builder := NewSiteBuilder(BuildOptions{
		ProjectDir:  projDir,
		Config:      cfg,
		ThemeConfig: themeCfg,
		EmbeddedFS:  embedded.ThemeFS(),
	})

	_, err := builder.Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	distDir := filepath.Join(projDir, "dist")

	data, err := os.ReadFile(filepath.Join(distDir, "search-index.en.json"))
	if err != nil {
		t.Fatalf("search-index.en.json not found: %v", err)
	}

	var docs []map[string]any
	if err := json.Unmarshal(data, &docs); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if len(docs) == 0 {
		t.Error("expected at least 1 document in search index")
	}

	// Check first doc has expected fields.
	for _, doc := range docs {
		if doc["title"] == nil {
			t.Error("expected title field in search document")
		}
		if doc["url"] == nil {
			t.Error("expected url field in search document")
		}
	}
}

func TestBuild_NoPlugins(t *testing.T) {
	projDir := createFixtureSite(t)
	cfg := config.Defaults()
	// No plugins enabled.
	cfg.Plugins.Enabled = nil
	themeCfg := buildThemeConfig()

	builder := NewSiteBuilder(BuildOptions{
		ProjectDir:  projDir,
		Config:      cfg,
		ThemeConfig: themeCfg,
		EmbeddedFS:  embedded.ThemeFS(),
	})

	result, err := builder.Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if result.PageCount == 0 {
		t.Error("expected pages even without plugins")
	}

	distDir := filepath.Join(projDir, "dist")

	// No sitemap or robots should exist.
	if _, err := os.Stat(filepath.Join(distDir, "sitemap.xml")); !os.IsNotExist(err) {
		t.Error("sitemap.xml should not exist without sitemap plugin")
	}
	if _, err := os.Stat(filepath.Join(distDir, "robots.txt")); !os.IsNotExist(err) {
		t.Error("robots.txt should not exist without robots plugin")
	}
}
