package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/frostybee/sarde/embedded"
	"github.com/frostybee/sarde/internal/config"
)

// createRichFixtureSite creates a fixture with content that triggers all inject_when rules:
// code blocks, images, headings, docs layout with sidebar, and updated timestamps.
func createRichFixtureSite(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	writeFixture(t, dir, "content/_index.md", "---\ntitle: Home\n---\n# Welcome\n")
	writeFixture(t, dir, "content/docs/_index.md", "---\ntitle: Docs\n---\n")
	writeFixture(t, dir, "content/docs/guide.md", `---
title: Guide
weight: 1
updated: 2025-06-01T00:00:00Z
description: A guide page with code and images.
---
## Introduction

Some introductory text.

### Code Example

`+"```go\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n```\n\n"+`
### An Image

![screenshot](https://example.com/img.png)

## Conclusion

Final thoughts.
`)
	writeFixture(t, dir, "content/docs/plain.md", "---\ntitle: Plain\nweight: 2\n---\nNo code or images here.\n")
	writeFixture(t, dir, "content/blog/_index.md", "---\ntitle: Blog\n---\n")
	writeFixture(t, dir, "content/blog/post.md", "---\ntitle: Post\ndate: 2025-01-15T00:00:00Z\n---\n## Hello\nA blog post.\n")

	return dir
}

func TestBuild_ClientPlugins_BundleOutput(t *testing.T) {
	projDir := createRichFixtureSite(t)
	cfg := config.Defaults()
	cfg.Plugins.Enabled = []string{"scroll-to-top", "focus-mode", "code-collapsible"}
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
	pluginDir := filepath.Join(distDir, "assets", "plugins")

	entries, err := os.ReadDir(pluginDir)
	if err != nil {
		t.Fatalf("reading plugin dir: %v", err)
	}

	var cssFound, jsFound bool
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, "plugins.") && strings.HasSuffix(name, ".css") {
			cssFound = true
		}
		if strings.HasPrefix(name, "plugins.") && strings.HasSuffix(name, ".js") {
			jsFound = true
		}
	}

	if !cssFound {
		t.Error("expected plugins.{hash}.css bundle file in dist/assets/plugins/")
	}
	if !jsFound {
		t.Error("expected plugins.{hash}.js bundle file in dist/assets/plugins/")
	}
}

func TestBuild_ClientPlugins_BundleOnEveryPage(t *testing.T) {
	projDir := createRichFixtureSite(t)
	cfg := config.Defaults()
	cfg.Plugins.Enabled = []string{"scroll-to-top", "code-collapsible", "image-lightbox"}
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

	// The bundle CSS/JS should be on every page
	for _, pagePath := range []string{"docs/guide/index.html", "docs/plain/index.html", "index.html"} {
		html := readFixture(t, distDir, pagePath)
		if !strings.Contains(html, "plugins.") || !strings.Contains(html, ".css") {
			t.Errorf("%s should have bundle CSS link", pagePath)
		}
		if !strings.Contains(html, "plugins.") || !strings.Contains(html, ".js") {
			t.Errorf("%s should have bundle JS script", pagePath)
		}
	}
}

func TestBuild_ClientPlugins_PerPageConfigInjection(t *testing.T) {
	projDir := createRichFixtureSite(t)
	cfg := config.Defaults()
	cfg.Plugins.Enabled = []string{
		"scroll-to-top", "code-collapsible", "image-lightbox",
		"focus-mode",
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

	// guide.md has code blocks + images + headings + docs layout
	guideHTML := readFixture(t, distDir, "docs/guide/index.html")

	// Config for code-collapsible should be injected (has_code_blocks → guide has code blocks)
	if !strings.Contains(guideHTML, `code-collapsible`) {
		t.Error("guide page should have code-collapsible config (has code blocks)")
	}

	// Config for image-lightbox should be injected (has_images → guide has images)
	if !strings.Contains(guideHTML, `image-lightbox`) {
		t.Error("guide page should have image-lightbox config (has images)")
	}

	// plain.md has no code blocks, no images → those config scripts should NOT be injected
	plainHTML := readFixture(t, distDir, "docs/plain/index.html")

	if strings.Contains(plainHTML, `code-collapsible`) {
		t.Error("plain page should NOT have code-collapsible config (no code blocks)")
	}

	if strings.Contains(plainHTML, `image-lightbox`) {
		t.Error("plain page should NOT have image-lightbox config (no images)")
	}
}

func TestBuild_ClientPlugins_ConfigInjection(t *testing.T) {
	projDir := createRichFixtureSite(t)
	cfg := config.Defaults()
	cfg.Plugins.Enabled = []string{"scroll-to-top"}
	cfg.Plugins.Config = map[string]map[string]any{
		"scroll-to-top": {"threshold": 50, "showProgressRing": true},
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
	homeHTML := readFixture(t, distDir, "index.html")

	if !strings.Contains(homeHTML, `__pluginConfig`) {
		t.Error("expected plugin config injection in page HTML")
	}
	if !strings.Contains(homeHTML, `scroll-to-top`) {
		t.Error("expected scroll-to-top config key in inline script")
	}
}

func TestBuild_ClientPlugins_ModuleScripts(t *testing.T) {
	projDir := createRichFixtureSite(t)
	cfg := config.Defaults()
	cfg.Plugins.Enabled = []string{"scroll-to-top"}
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
	guideHTML := readFixture(t, distDir, "docs/guide/index.html")

	if !strings.Contains(guideHTML, `type="module"`) {
		t.Error("expected <script type=\"module\"> for plugin bundle")
	}
}

func TestBuild_ClientPlugins_FingerprintedBundle(t *testing.T) {
	projDir := createRichFixtureSite(t)
	cfg := config.Defaults()
	cfg.Plugins.Enabled = []string{"scroll-to-top"}
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
	pluginDir := filepath.Join(distDir, "assets", "plugins")

	entries, err := os.ReadDir(pluginDir)
	if err != nil {
		t.Fatalf("reading plugin dir: %v", err)
	}

	var jsFound, cssFound bool
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, "plugins.") && strings.HasSuffix(name, ".js") {
			parts := strings.Split(name, ".")
			if len(parts) == 3 && len(parts[1]) == 8 {
				jsFound = true
			}
		}
		if strings.HasPrefix(name, "plugins.") && strings.HasSuffix(name, ".css") {
			parts := strings.Split(name, ".")
			if len(parts) == 3 && len(parts[1]) == 8 {
				cssFound = true
			}
		}
	}

	if !jsFound {
		t.Error("expected fingerprinted JS bundle (plugins.XXXXXXXX.js)")
	}
	if !cssFound {
		t.Error("expected fingerprinted CSS bundle (plugins.XXXXXXXX.css)")
	}
}

func TestBuild_Announcements_BannerRendered(t *testing.T) {
	projDir := createRichFixtureSite(t)
	cfg := config.Defaults()
	cfg.Plugins.Enabled = []string{"announcements"}
	cfg.Plugins.Config = map[string]map[string]any{
		"announcements": {
			"items": []any{
				map[string]any{
					"message":     "Site under maintenance",
					"type":        "warning",
					"active":      true,
					"dismissible": true,
					"id":          "maint-1",
				},
			},
		},
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
	guideHTML := readFixture(t, distDir, "docs/guide/index.html")

	if !strings.Contains(guideHTML, "sarde-announcement-container") {
		t.Error("expected announcement-container wrapper")
	}
	if !strings.Contains(guideHTML, "sarde-announcement-banner") {
		t.Error("expected announcement banner in page HTML")
	}
	if !strings.Contains(guideHTML, "Site under maintenance") {
		t.Error("expected announcement message text")
	}
	if !strings.Contains(guideHTML, "announcement-warning") {
		t.Error("expected warning type class")
	}
	if !strings.Contains(guideHTML, `data-announcement-id="maint-1"`) {
		t.Error("expected announcement ID attribute")
	}
	if !strings.Contains(guideHTML, "sarde-announcement-dismiss") {
		t.Error("expected dismiss button")
	}
	if !strings.Contains(guideHTML, `aria-label="Dismiss announcement"`) {
		t.Error("expected i18n dismiss label")
	}
}

func TestBuild_Announcements_InactiveNoBanner(t *testing.T) {
	projDir := createRichFixtureSite(t)
	cfg := config.Defaults()
	cfg.Plugins.Enabled = []string{"announcements"}
	cfg.Plugins.Config = map[string]map[string]any{
		"announcements": {
			"items": []any{
				map[string]any{
					"message": "Hidden",
					"active":  false,
				},
			},
		},
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
	guideHTML := readFixture(t, distDir, "docs/guide/index.html")

	if strings.Contains(guideHTML, "sarde-announcement-banner") {
		t.Error("inactive announcement should not render banner")
	}
}

func TestBuild_Announcements_RotateMode(t *testing.T) {
	projDir := createRichFixtureSite(t)
	cfg := config.Defaults()
	cfg.Plugins.Enabled = []string{"announcements"}
	cfg.Plugins.Config = map[string]map[string]any{
		"announcements": {
			"display_mode":    "rotate",
			"rotate_interval": 3000,
			"items": []any{
				map[string]any{
					"id":      "first",
					"message": "First announcement",
					"type":    "info",
					"active":  true,
				},
				map[string]any{
					"id":      "second",
					"message": "Second announcement",
					"type":    "warning",
					"active":  true,
				},
			},
		},
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
	guideHTML := readFixture(t, distDir, "docs/guide/index.html")

	if !strings.Contains(guideHTML, `data-display-mode="rotate"`) {
		t.Error("expected rotate display mode on container")
	}
	if !strings.Contains(guideHTML, `data-rotate-interval="3000"`) {
		t.Error("expected rotate interval on container")
	}
	if !strings.Contains(guideHTML, `data-announcement-id="first"`) {
		t.Error("expected first announcement")
	}
	if !strings.Contains(guideHTML, `data-announcement-id="second"`) {
		t.Error("expected second announcement")
	}
}

func TestBuild_Announcements_DateAndPageTargeting(t *testing.T) {
	projDir := createRichFixtureSite(t)
	cfg := config.Defaults()
	cfg.Plugins.Enabled = []string{"announcements"}
	cfg.Plugins.Config = map[string]map[string]any{
		"announcements": {
			"items": []any{
				map[string]any{
					"id":         "dated",
					"message":    "Scheduled announcement",
					"active":     true,
					"start_date": "2025-01-01T00:00:00Z",
					"end_date":   "2027-12-31T23:59:59Z",
					"show_on":    []any{"/docs/**"},
					"hide_on":    []any{"/admin/**"},
				},
			},
		},
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
	guideHTML := readFixture(t, distDir, "docs/guide/index.html")

	if !strings.Contains(guideHTML, `data-start-date="2025-01-01T00:00:00Z"`) {
		t.Error("expected start date attribute")
	}
	if !strings.Contains(guideHTML, `data-end-date="2027-12-31T23:59:59Z"`) {
		t.Error("expected end date attribute")
	}
	if !strings.Contains(guideHTML, `data-show-on="/docs/**"`) {
		t.Error("expected show-on attribute")
	}
	if !strings.Contains(guideHTML, `data-hide-on="/admin/**"`) {
		t.Error("expected hide-on attribute")
	}
}

func TestBuild_AllClientPlugins(t *testing.T) {
	projDir := createRichFixtureSite(t)
	cfg := config.Defaults()
	cfg.Plugins.Enabled = []string{
		"scroll-to-top", "copy-section-link", "external-links",
		"image-lightbox", "keyboard-nav", "focus-mode",
		"reading-progress", "search-highlighter", "text-highlighter",
		"last-updated", "reading-position-memory",
		"reading-preferences", "code-collapsible",
		"announcements",
	}
	cfg.Plugins.Config = map[string]map[string]any{
		"announcements": {
			"items": []any{
				map[string]any{
					"message": "Test banner",
					"active":  true,
				},
			},
		},
	}
	themeCfg := buildThemeConfig()

	builder := NewSiteBuilder(BuildOptions{
		ProjectDir:  projDir,
		Config:      cfg,
		ThemeConfig: themeCfg,
		EmbeddedFS:  embedded.ThemeFS(),
	})

	result, err := builder.Build()
	if err != nil {
		t.Fatalf("Build with all 16 plugins failed: %v", err)
	}

	if result.PageCount == 0 {
		t.Error("expected pages in build result")
	}

	distDir := filepath.Join(projDir, "dist")
	pluginDir := filepath.Join(distDir, "assets", "plugins")

	// Should have 2 bundle files + announcements dir
	entries, err := os.ReadDir(pluginDir)
	if err != nil {
		t.Fatalf("reading plugin dir: %v", err)
	}

	var cssBundle, jsBundle, announcementsDir bool
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, "plugins.") && strings.HasSuffix(name, ".css") {
			cssBundle = true
		}
		if strings.HasPrefix(name, "plugins.") && strings.HasSuffix(name, ".js") {
			jsBundle = true
		}
		if name == "announcements" && e.IsDir() {
			announcementsDir = true
		}
	}

	if !cssBundle {
		t.Error("expected plugins.{hash}.css bundle")
	}
	if !jsBundle {
		t.Error("expected plugins.{hash}.js bundle")
	}
	if !announcementsDir {
		t.Error("expected announcements/ directory (separate subpackage)")
	}
}

func TestBuild_ContentFeatureFlags(t *testing.T) {
	projDir := createRichFixtureSite(t)
	cfg := config.Defaults()
	cfg.Plugins.Enabled = []string{"code-collapsible", "image-lightbox"}
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

	// guide.md has code blocks + images → config scripts should be injected
	guideHTML := readFixture(t, distDir, "docs/guide/index.html")
	if !strings.Contains(guideHTML, `code-collapsible`) {
		t.Error("guide page should have code-collapsible config (HasCodeBlocks=true)")
	}
	if !strings.Contains(guideHTML, `image-lightbox`) {
		t.Error("guide page should have image-lightbox config (HasImages=true)")
	}

	// plain.md has neither → config scripts should NOT be injected
	plainHTML := readFixture(t, distDir, "docs/plain/index.html")
	if strings.Contains(plainHTML, `code-collapsible`) {
		t.Error("plain page should NOT have code-collapsible config (HasCodeBlocks=false)")
	}
	if strings.Contains(plainHTML, `image-lightbox`) {
		t.Error("plain page should NOT have image-lightbox config (HasImages=false)")
	}
}
