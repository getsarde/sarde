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
	if result.SitemapCount != 1 {
		t.Errorf("SitemapCount = %d, want 1", result.SitemapCount)
	}
	if home := readFixture(t, distDir, "index.html"); !strings.Contains(home, `<link rel="sitemap" type="application/xml" href="/sitemap.xml">`) {
		t.Error("expected a root-relative <link rel=\"sitemap\"> on the home page")
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

func TestBuild_Plugins_SearchSkipsTaxonomyAndRepeatedCrumbs(t *testing.T) {
	projDir := createFixtureSite(t)
	writeFixture(t, projDir, "content/blog/tagged.md", "---\ntitle: Tagged\ndate: 2025-02-01T00:00:00Z\ntags: [python]\n---\nA tagged post.\n")
	writeFixture(t, projDir, "content/docs/guides/_index.md", "---\ntitle: Guides\n---\n## Overview\nAll the guides.\n")
	writeFixture(t, projDir, "content/docs/guides/install.md", "---\ntitle: Install\n---\n## Steps\nRun the installer.\n")
	writeFixture(t, projDir, "content/about.md", "---\ntitle: About\n---\n## Contact\nWrite to us.\n")

	cfg := config.Defaults()
	cfg.Plugins.Enabled = []string{"search"}
	builder := NewSiteBuilder(BuildOptions{
		ProjectDir:  projDir,
		Config:      cfg,
		ThemeConfig: buildThemeConfig(),
		EmbeddedFS:  embedded.ThemeFS(),
	})
	if _, err := builder.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	distDir := filepath.Join(projDir, "dist")
	// The tag page must exist, or the check below proves nothing.
	if _, err := os.Stat(filepath.Join(distDir, "tags", "python", "index.html")); err != nil {
		t.Fatalf("expected the python tag page to be built: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(distDir, "search-index.en.json"))
	if err != nil {
		t.Fatalf("search-index.en.json not found: %v", err)
	}
	var docs []struct {
		Title      string `json:"title"`
		URL        string `json:"url"`
		Breadcrumb string `json:"breadcrumb"`
	}
	if err := json.Unmarshal(data, &docs); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	crumbs := make(map[string]string, len(docs))
	for _, d := range docs {
		if strings.HasPrefix(d.URL, "/tags/") {
			t.Errorf("taxonomy page %q must not be indexed", d.URL)
		}
		crumbs[d.Title] = d.Breadcrumb
	}
	for title, want := range map[string]string{
		"Install":  "Documentation > Guides",
		"Steps":    "Documentation > Guides > Install",
		"Overview": "Documentation > Guides",
		"Contact":  "About",
	} {
		got, ok := crumbs[title]
		if !ok {
			t.Errorf("no search document titled %q", title)
			continue
		}
		if got != want {
			t.Errorf("%s: breadcrumb = %q, want %q", title, got, want)
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

// The sitemap file, its count in the build result and the <link rel="sitemap">
// tag must agree whichever switch turns the sitemap off.
func TestBuild_SitemapOff_CountAndLinkAgree(t *testing.T) {
	for name, setup := range map[string]func(*config.SiteConfig){
		"plugins.disabled":     func(c *config.SiteConfig) { c.Plugins.Disabled = []string{"sitemap"} },
		"build.sitemap: false": func(c *config.SiteConfig) { c.Build.Sitemap = config.BoolPtr(false) },
	} {
		t.Run(name, func(t *testing.T) {
			projDir := createFixtureSite(t)
			cfg := config.Defaults()
			cfg.Site.URL = "https://example.com"
			cfg.Plugins.Enabled = []string{"sitemap", "robots"}
			setup(cfg)

			builder := NewSiteBuilder(BuildOptions{
				ProjectDir:  projDir,
				Config:      cfg,
				ThemeConfig: buildThemeConfig(),
				EmbeddedFS:  embedded.ThemeFS(),
			})
			result, err := builder.Build()
			if err != nil {
				t.Fatalf("Build failed: %v", err)
			}

			distDir := filepath.Join(projDir, "dist")
			if _, err := os.Stat(filepath.Join(distDir, "sitemap.xml")); !os.IsNotExist(err) {
				t.Error("sitemap.xml was written")
			}
			if result.SitemapCount != 0 {
				t.Errorf("SitemapCount = %d, want 0", result.SitemapCount)
			}
			if home := readFixture(t, distDir, "index.html"); strings.Contains(home, `rel="sitemap"`) {
				t.Error(`home page links a sitemap that was not built`)
			}
			if robots := readFixture(t, distDir, "robots.txt"); strings.Contains(robots, "Sitemap:") {
				t.Errorf("robots.txt references a sitemap that was not built:\n%s", robots)
			}
		})
	}
}
