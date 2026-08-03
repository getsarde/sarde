package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/getsarde/sarde/embedded"
	"github.com/getsarde/sarde/internal/config"
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
	cfg.Plugins.Enabled = []string{"scroll_to_top", "focus_mode", "reading_progress"}
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

// readPluginBundle returns the concatenated contents of every file in
// dist/assets/plugins/, or "" when the directory does not exist.
func readPluginBundle(t *testing.T, distDir string) string {
	t.Helper()
	pluginDir := filepath.Join(distDir, "assets", "plugins")
	entries, err := os.ReadDir(pluginDir)
	if err != nil {
		if os.IsNotExist(err) {
			return ""
		}
		t.Fatalf("reading plugin dir: %v", err)
	}
	var sb strings.Builder
	for _, e := range entries {
		data, err := os.ReadFile(filepath.Join(pluginDir, e.Name()))
		if err != nil {
			t.Fatalf("reading %s: %v", e.Name(), err)
		}
		sb.Write(data)
	}
	return sb.String()
}

// TestBuild_ClientPlugins_DisabledNotBundled is the end-to-end guard for
// plugins.enabled governing client plugins: a plugin that is not enabled must
// ship no code. Mirrors TestBuild_ExternalPlugin_DisabledViaConfig.
func TestBuild_ClientPlugins_DisabledNotBundled(t *testing.T) {
	projDir := createRichFixtureSite(t)
	cfg := config.Defaults()
	cfg.Plugins.Enabled = []string{"scroll_to_top"}
	themeCfg := buildThemeConfig()

	builder := NewSiteBuilder(BuildOptions{
		ProjectDir:  projDir,
		Config:      cfg,
		ThemeConfig: themeCfg,
		EmbeddedFS:  embedded.ThemeFS(),
	})

	if _, err := builder.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	bundle := readPluginBundle(t, filepath.Join(projDir, "dist"))
	if bundle == "" {
		t.Fatal("no plugin bundle emitted, expected scroll-to-top")
	}
	if !strings.Contains(bundle, "sarde-scroll-to-top") {
		t.Error("bundle is missing scroll-to-top, which was enabled")
	}
	for _, marker := range []string{"Navigate between pages", "sarde-kbd-nav-hint", "sarde-reading-progress"} {
		if strings.Contains(bundle, marker) {
			t.Errorf("bundle contains %q from a plugin that was not enabled", marker)
		}
	}
}

// TestBuild_ClientPlugins_NoneEnabled covers a site that enables no client
// plugins: no bundle is written and no page references one.
func TestBuild_ClientPlugins_NoneEnabled(t *testing.T) {
	projDir := createRichFixtureSite(t)
	cfg := config.Defaults()
	cfg.Plugins.Enabled = []string{"seo"}
	themeCfg := buildThemeConfig()

	builder := NewSiteBuilder(BuildOptions{
		ProjectDir:  projDir,
		Config:      cfg,
		ThemeConfig: themeCfg,
		EmbeddedFS:  embedded.ThemeFS(),
	})

	if _, err := builder.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	distDir := filepath.Join(projDir, "dist")
	if bundle := readPluginBundle(t, distDir); bundle != "" {
		t.Error("plugin bundle emitted with no client plugins enabled")
	}
	if html := readFixture(t, distDir, "docs/guide/index.html"); strings.Contains(html, "/assets/plugins/plugins.") {
		t.Error("page references a plugin bundle with no client plugins enabled")
	}
}

// buildAndCollectWarnings builds the fixture and returns its warning messages.
func buildAndCollectWarnings(t *testing.T, projDir string, cfg *config.SiteConfig) []string {
	t.Helper()
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
	msgs := make([]string, 0, len(result.Warnings))
	for _, w := range result.Warnings {
		msgs = append(msgs, w.Message)
	}
	return msgs
}

func containsSubstring(msgs []string, want string) bool {
	for _, m := range msgs {
		if strings.Contains(m, want) {
			return true
		}
	}
	return false
}

// TestBuild_PluginConfig_IgnoredWarns covers the diagnostic for the failure
// mode this fix addresses: configuring a plugin that is not enabled used to be
// a silent no-op.
func TestBuild_PluginConfig_IgnoredWarns(t *testing.T) {
	cfg := config.Defaults()
	cfg.Plugins.Enabled = []string{"scroll_to_top"}
	cfg.Plugins.Config = map[string]map[string]any{
		"keyboard_nav":  {"show_hint": false},
		"keyboard_navv": {"show_hint": false},
		"scroll_to_top": {"show_tooltip": true},
	}

	msgs := buildAndCollectWarnings(t, createRichFixtureSite(t), cfg)

	if !containsSubstring(msgs, `plugin "keyboard_nav" is not in plugins.enabled`) {
		t.Errorf("expected a not-enabled warning for keyboard_nav, warnings: %v", msgs)
	}
	if !containsSubstring(msgs, `unknown plugin "keyboard_navv"`) {
		t.Errorf("expected an unknown-plugin warning for the typo, warnings: %v", msgs)
	}
	if containsSubstring(msgs, "scroll_to_top") {
		t.Errorf("enabled plugin should not warn, warnings: %v", msgs)
	}
}

// TestBuild_PluginConfig_EnabledNoWarn is the counterpart: once the plugin is
// enabled, its config block is live and must not warn.
func TestBuild_PluginConfig_EnabledNoWarn(t *testing.T) {
	cfg := config.Defaults()
	cfg.Plugins.Enabled = []string{"scroll_to_top", "keyboard_nav"}
	cfg.Plugins.Config = map[string]map[string]any{
		"keyboard_nav": {"show_hint": false},
	}

	msgs := buildAndCollectWarnings(t, createRichFixtureSite(t), cfg)

	if containsSubstring(msgs, "keyboard_nav") {
		t.Errorf("enabled plugin should not warn, warnings: %v", msgs)
	}
}

func TestBuild_ClientPlugins_BundleOnEveryPage(t *testing.T) {
	projDir := createRichFixtureSite(t)
	cfg := config.Defaults()
	cfg.Plugins.Enabled = []string{"scroll_to_top", "reading_progress", "image_lightbox"}
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
		"scroll_to_top", "image_lightbox",
		"focus_mode",
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

	// guide.md has images + headings + docs layout
	guideHTML := readFixture(t, distDir, "docs/guide/index.html")

	// Config for image-lightbox should be injected (has_images → guide has images)
	if !strings.Contains(guideHTML, `image_lightbox`) {
		t.Error("guide page should have image-lightbox config (has images)")
	}

	// plain.md has no images → config script should NOT be injected
	plainHTML := readFixture(t, distDir, "docs/plain/index.html")

	if strings.Contains(plainHTML, `image_lightbox`) {
		t.Error("plain page should NOT have image-lightbox config (no images)")
	}
}

func TestBuild_ClientPlugins_ConfigInjection(t *testing.T) {
	projDir := createRichFixtureSite(t)
	cfg := config.Defaults()
	cfg.Plugins.Enabled = []string{"scroll_to_top"}
	cfg.Plugins.Config = map[string]map[string]any{
		"scroll_to_top": {"threshold": 50, "showProgressRing": true},
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

	if !strings.Contains(homeHTML, `__SARDE__`) {
		t.Error("expected plugin config injection in page HTML")
	}
	if !strings.Contains(homeHTML, `scroll_to_top`) {
		t.Error("expected scroll-to-top config key in inline script")
	}
}

func TestBuild_ClientPlugins_ModuleScripts(t *testing.T) {
	projDir := createRichFixtureSite(t)
	cfg := config.Defaults()
	cfg.Plugins.Enabled = []string{"scroll_to_top"}
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
	cfg.Plugins.Enabled = []string{"scroll_to_top"}
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
		"scroll_to_top", "copy_section_link", "external_links",
		"image_lightbox", "keyboard_nav", "focus_mode",
		"reading_progress", "search_highlighter", "text_highlighter",
		"reading_position_memory",
		"reading_preferences",
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
		t.Fatalf("Build with all 12 plugins failed: %v", err)
	}

	if result.PageCount == 0 {
		t.Error("expected pages in build result")
	}

	distDir := filepath.Join(projDir, "dist")
	pluginDir := filepath.Join(distDir, "assets", "plugins")

	// Should have 2 bundle files + announcements dir (12 client plugins total)
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
	cfg.Plugins.Enabled = []string{"image_lightbox"}
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

	// guide.md has images → config script should be injected
	guideHTML := readFixture(t, distDir, "docs/guide/index.html")
	if !strings.Contains(guideHTML, `image_lightbox`) {
		t.Error("guide page should have image-lightbox config (HasImages=true)")
	}

	// plain.md has no images → config script should NOT be injected
	plainHTML := readFixture(t, distDir, "docs/plain/index.html")
	if strings.Contains(plainHTML, `image_lightbox`) {
		t.Error("plain page should NOT have image-lightbox config (HasImages=false)")
	}
}

// TestBuild_PluginConfig_KeyWarnings covers field-level config checking for an
// active client plugin: a typo warns as unknown, a legacy camelCase spelling
// warns as deprecated, and declared keys stay silent.
func TestBuild_PluginConfig_KeyWarnings(t *testing.T) {
	cfg := config.Defaults()
	cfg.Plugins.Enabled = []string{"scroll_to_top"}
	cfg.Plugins.Config = map[string]map[string]any{
		"scroll_to_top": {"show_tolltip": true, "showTooltip": true, "threshold": 40},
	}

	msgs := buildAndCollectWarnings(t, createRichFixtureSite(t), cfg)

	if !containsSubstring(msgs, `unknown config key "show_tolltip"`) {
		t.Errorf("expected unknown-key warning, warnings: %v", msgs)
	}
	if !containsSubstring(msgs, `config key "showTooltip" is deprecated; use "show_tooltip"`) {
		t.Errorf("expected deprecated-key warning, warnings: %v", msgs)
	}
	if containsSubstring(msgs, `"threshold"`) {
		t.Errorf("declared key should not warn, warnings: %v", msgs)
	}
}

// TestWarnPluginConfigKeys_UndeclaredPluginSkipped guards against false
// positives: plugins without a declared field set (external plugins, Go
// built-ins) must produce no key warnings at all.
func TestWarnPluginConfigKeys_UndeclaredPluginSkipped(t *testing.T) {
	if warns := warnPluginConfigKeys("some_external_plugin", map[string]any{"anything": 1}); len(warns) != 0 {
		t.Errorf("expected no warnings for a plugin without a declared field set, got %v", warns)
	}
	if warns := warnPluginConfigKeys("seo", map[string]any{"anything": 1}); len(warns) != 0 {
		t.Errorf("expected no warnings for a Go built-in, got %v", warns)
	}
}

// TestBuild_LegacyPluginSlugAndKey_EndToEnd proves a legacy kebab slug and a
// legacy camelCase field key still work through the real config path:
// sarde.yaml -> config.Resolve (alias normalization) -> build.
func TestBuild_LegacyPluginSlugAndKey_EndToEnd(t *testing.T) {
	projDir := createRichFixtureSite(t)
	writeFixture(t, projDir, "sarde.yaml", `site:
  title: Legacy
plugins:
  enabled:
    - scroll-to-top
  config:
    scroll-to-top:
      showProgressRing: true
`)

	cfg, err := config.Resolve(config.ResolveOptions{
		ConfigPath:   filepath.Join(projDir, "sarde.yaml"),
		KnownPlugins: KnownPluginNames(projDir),
	})
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if len(cfg.Plugins.Enabled) != 1 || cfg.Plugins.Enabled[0] != "scroll_to_top" {
		t.Fatalf("legacy slug not normalized, enabled = %v", cfg.Plugins.Enabled)
	}
	if _, ok := cfg.Plugins.Config["scroll_to_top"]; !ok {
		t.Fatalf("legacy config slug not re-keyed, config = %v", cfg.Plugins.Config)
	}

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
	if bundle := readPluginBundle(t, distDir); !strings.Contains(bundle, "sarde-scroll-to-top") {
		t.Error("bundle missing scroll_to_top assets when enabled via legacy slug")
	}
	homeHTML := readFixture(t, distDir, "index.html")
	if !strings.Contains(homeHTML, `"show_progress_ring":true`) {
		t.Error("legacy camelCase key not normalized in injected config")
	}
	if strings.Contains(homeHTML, "showProgressRing") {
		t.Error("legacy camelCase key leaked into injected config")
	}
}

// TestBuild_CanonicalSlugAndKey_NoLegacyPath is the counterpart: canonical
// spellings resolve to the same enabled set and injected config.
func TestBuild_CanonicalSlugAndKey_NoLegacyPath(t *testing.T) {
	projDir := createRichFixtureSite(t)
	writeFixture(t, projDir, "sarde.yaml", `site:
  title: Canonical
plugins:
  enabled:
    - scroll_to_top
  config:
    scroll_to_top:
      show_progress_ring: true
`)

	cfg, err := config.Resolve(config.ResolveOptions{
		ConfigPath:   filepath.Join(projDir, "sarde.yaml"),
		KnownPlugins: KnownPluginNames(projDir),
	})
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if len(cfg.Plugins.Enabled) != 1 || cfg.Plugins.Enabled[0] != "scroll_to_top" {
		t.Fatalf("canonical slug mangled, enabled = %v", cfg.Plugins.Enabled)
	}

	builder := NewSiteBuilder(BuildOptions{
		ProjectDir:  projDir,
		Config:      cfg,
		ThemeConfig: buildThemeConfig(),
		EmbeddedFS:  embedded.ThemeFS(),
	})
	if _, err := builder.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	homeHTML := readFixture(t, filepath.Join(projDir, "dist"), "index.html")
	if !strings.Contains(homeHTML, `"show_progress_ring":true`) {
		t.Error("canonical key missing from injected config")
	}
}

// TestReservedPluginNames_IncludesLegacyAliases: the reserved set must keep
// the deprecated kebab spellings claimed, or an external plugin could shadow
// a legacy slug the config alias layer still resolves.
func TestReservedPluginNames_IncludesLegacyAliases(t *testing.T) {
	set := make(map[string]bool)
	for _, n := range ReservedPluginNames("") {
		set[n] = true
	}
	for _, want := range []string{"scroll_to_top", "scroll-to-top", "keyboard_nav", "keyboard-nav", "content_lint", "content-lint"} {
		if !set[want] {
			t.Errorf("reserved set missing %q", want)
		}
	}
}
