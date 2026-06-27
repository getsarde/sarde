package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/getsarde/sarde/embedded"
	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/content/markdown/icons"
	"github.com/getsarde/sarde/internal/engine"
	"github.com/getsarde/sarde/internal/theme"
)

// createFixtureSite creates a minimal site structure in a temp directory.
func createFixtureSite(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	writeFixture(t, dir, "content/_index.md", "---\ntitle: Home\n---\n# Welcome\nThis is the home page.\n")
	writeFixture(t, dir, "content/about.md", "---\ntitle: About\n---\nAbout this site.\n")
	writeFixture(t, dir, "content/docs/_index.md", "---\ntitle: Documentation\n---\n")
	writeFixture(t, dir, "content/docs/getting-started.md", "---\ntitle: Getting Started\nweight: 1\n---\n# Getting Started\nIntro content.\n")
	writeFixture(t, dir, "content/docs/advanced.md", "---\ntitle: Advanced\nweight: 2\n---\n# Advanced\nAdvanced content.\n")
	writeFixture(t, dir, "content/blog/_index.md", "---\ntitle: Blog\n---\n")
	writeFixture(t, dir, "content/blog/hello-world.md", "---\ntitle: Hello World\ndate: 2025-01-15T00:00:00Z\n---\n# Hello World\nFirst post.\n")

	return dir
}

func writeFixture(t *testing.T, base, rel, body string) {
	t.Helper()
	path := filepath.Join(base, filepath.FromSlash(rel))
	os.MkdirAll(filepath.Dir(path), 0o755)
	os.WriteFile(path, []byte(body), 0o644)
}

func buildThemeConfig() *engine.ThemeConfig {
	thm, _ := theme.LoadFromFS(embedded.ThemeFS(), ".")
	light := theme.ResolveTokens(theme.DefaultTokens(), thm, "", nil)
	light = theme.DeriveTokens(light)
	dark := theme.ResolveDarkTokens(theme.DefaultDarkTokens(), thm, "", nil)
	styleTag := theme.GenerateStyleTag(light, dark)

	name := "default"
	if thm != nil {
		name = thm.Name
	}
	return &engine.ThemeConfig{
		Name:        name,
		Tokens:      light,
		DarkTokens:  dark,
		DarkEnabled: true,
		StyleTag:    styleTag,
	}
}

func TestBuild_EndToEnd(t *testing.T) {
	projDir := createFixtureSite(t)
	cfg := config.Defaults()
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
		t.Error("expected at least 1 page")
	}

	distDir := filepath.Join(projDir, "dist")

	// Home page.
	assertFixtureFileExists(t, distDir, "index.html")
	assertFixtureFileContains(t, distDir, "index.html", "<!DOCTYPE html>")
	assertFixtureFileContains(t, distDir, "index.html", "Welcome")

	// About page (standalone).
	assertFixtureFileExists(t, distDir, "about/index.html")

	// Docs collection.
	assertFixtureFileExists(t, distDir, "docs/getting-started/index.html")
	assertFixtureFileContains(t, distDir, "docs/getting-started/index.html", "Getting Started")

	assertFixtureFileExists(t, distDir, "docs/advanced/index.html")

	// Blog collection.
	assertFixtureFileExists(t, distDir, "blog/hello-world/index.html")
	assertFixtureFileContains(t, distDir, "blog/hello-world/index.html", "Hello World")

	// 404 page.
	assertFixtureFileExists(t, distDir, "404.html")
}

// TestBuild_IconRenderModeBustsPageCache guards the page-cache key: toggling
// icons.render must invalidate cached pages so content re-renders in the new
// mode. Regression — the cache key previously omitted the icon render mode, so a
// warm cache served stale inline icons after switching to sprite (and the serial
// render path needed the key threaded through markdownRenderDeps).
func TestBuild_IconRenderModeBustsPageCache(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "content/_index.md", "---\ntitle: Home\n---\n# Home\n")
	writeFixture(t, dir, "content/docs/_index.md", "---\ntitle: Docs\n---\n")
	writeFixture(t, dir, "content/docs/icons.md", "---\ntitle: Icons\nweight: 1\n---\nStars :icon[star] :icon[star] :icon[star]\n")
	themeCfg := buildThemeConfig()
	defer icons.SetRenderMode("inline") // restore global mode for other tests

	build := func(render string) string {
		cfg := config.Defaults()
		cfg.Icons.Render = render
		cfg.Build.Minify = config.BoolPtr(false)
		b := NewSiteBuilder(BuildOptions{ProjectDir: dir, Config: cfg, ThemeConfig: themeCfg, EmbeddedFS: embedded.ThemeFS()})
		if _, err := b.Build(); err != nil {
			t.Fatalf("build (%s) failed: %v", render, err)
		}
		data, err := os.ReadFile(filepath.Join(dir, "dist", "docs", "icons", "index.html"))
		if err != nil {
			t.Fatalf("read output (%s): %v", render, err)
		}
		return string(data)
	}

	// First build (inline) warms the on-disk page cache with inline <svg> bodies.
	if inline := build("inline"); strings.Contains(inline, `<use href="#i-lucide-star"`) {
		t.Fatalf("inline build unexpectedly emitted <use> sprite refs")
	}

	// Second build (same project dir → warm cache) in sprite mode must bust the
	// cache and re-render with <use> refs + a <symbol>, not serve stale inline HTML.
	sprite := build("sprite")
	if !strings.Contains(sprite, `<use href="#i-lucide-star"`) {
		t.Error("sprite build did not emit <use> refs — render-mode change did not invalidate the page cache")
	}
	if !strings.Contains(sprite, `<symbol id="i-lucide-star"`) {
		t.Error("sprite build did not emit the <symbol> definition")
	}
}

// TestContentRebuild_IconRenderKeyThreaded guards the incremental-rebuild
// page-cache key: ContentRebuild must reuse the icon render key from the last
// full build. Regression — markdownRenderDeps in ContentRebuild left
// iconRenderKey empty, so incremental cache keys never matched full-build keys
// (wasted re-renders, latent sprite/inline inconsistency).
func TestContentRebuild_IconRenderKeyThreaded(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "content/_index.md", "---\ntitle: Home\n---\n# Home\n")
	writeFixture(t, dir, "content/docs/_index.md", "---\ntitle: Docs\n---\n")
	writeFixture(t, dir, "content/docs/icons.md", "---\ntitle: Icons\nweight: 1\n---\nStars :icon[star]\n")
	defer icons.SetRenderMode("inline") // restore global mode for other tests

	cfg := config.Defaults()
	cfg.Icons.Render = "sprite"
	cfg.Build.Minify = config.BoolPtr(false)
	b := NewSiteBuilder(BuildOptions{ProjectDir: dir, Config: cfg, ThemeConfig: buildThemeConfig(), EmbeddedFS: embedded.ThemeFS()})
	if _, err := b.Build(); err != nil {
		t.Fatalf("full build failed: %v", err)
	}
	if b.lastIconRenderKey != "icon-sprite" {
		t.Fatalf("lastIconRenderKey = %q after sprite build, want %q", b.lastIconRenderKey, "icon-sprite")
	}

	writeFixture(t, dir, "content/docs/icons.md", "---\ntitle: Icons\nweight: 1\n---\nMore stars :icon[star] :icon[star]\n")
	if _, err := b.ContentRebuild([]string{filepath.Join(dir, "content", "docs", "icons.md")}); err != nil {
		t.Fatalf("ContentRebuild failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "dist", "docs", "icons", "index.html"))
	if err != nil {
		t.Fatalf("read rebuilt output: %v", err)
	}
	if !strings.Contains(string(data), `<use href="#i-lucide-star"`) {
		t.Error("incremental rebuild did not render sprite <use> refs")
	}
}

func TestBuild_HomeHero_BackCompatConfig(t *testing.T) {
	projDir := createFixtureSite(t)
	cfg := config.Defaults()
	cfg.Build.Minify = config.BoolPtr(false)
	cfg.Homepage.Template = "hero"
	cfg.Homepage.Hero.Title = "Velox"
	cfg.Homepage.Hero.Subtitle = "A blazing-fast Go HTTP router."
	cfg.Homepage.Hero.CTA = &config.HeroCTA{Label: "Get Started", URL: "/docs/"}

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
	assertFixtureFileContains(t, distDir, "index.html", `class="sarde-hero sarde-hero-bg-`)
	assertFixtureFileContains(t, distDir, "index.html", "Velox")
	assertFixtureFileContains(t, distDir, "index.html", "A blazing-fast Go HTTP router.")
	assertFixtureFileContains(t, distDir, "index.html", "Get Started")
	assertFixtureFileNotContains(t, distDir, "index.html", "sarde-hero-proof")
}

func TestBuild_HomeHero_OptionalFieldsRendered(t *testing.T) {
	projDir := createFixtureSite(t)
	cfg := config.Defaults()
	cfg.Build.Minify = config.BoolPtr(false)
	cfg.Homepage.Template = "hero"
	cfg.Homepage.Hero = config.HeroSettings{
		Eyebrow:    "Go HTTP Router",
		Title:      "Velox",
		Subtitle:   "Zero-allocation routing for production APIs.",
		Background: "gradient",
		CTA:        &config.HeroCTA{Label: "Get Started", URL: "/docs/"},
		SecondaryCTA: &config.HeroCTA{
			Label: "GitHub",
			URL:   "https://github.com/example/velox",
		},
		Stats: []config.HeroStat{
			{Value: "0", Label: "heap allocations"},
			{Value: "<1us", Label: "route matching"},
		},
		Code: &config.HeroCode{
			Title:    "Quick start",
			Language: "go",
			Body:     "r := velox.New()\nr.GET(\"/users/:id\", getUser)",
		},
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
	assertFixtureFileContains(t, distDir, "index.html", "Go HTTP Router")
	assertFixtureFileContains(t, distDir, "index.html", "sarde-hero-cta-secondary")
	assertFixtureFileContains(t, distDir, "index.html", "GitHub")
	assertFixtureFileContains(t, distDir, "index.html", "sarde-hero-proof")
	assertFixtureFileContains(t, distDir, "index.html", "Quick start")
	assertFixtureFileContains(t, distDir, "index.html", "language-go")
	assertFixtureFileContains(t, distDir, "index.html", "r := velox.New()")
	assertFixtureFileContains(t, distDir, "index.html", "heap allocations")
	assertFixtureFileContains(t, distDir, "index.html", "route matching")
}

func TestBuild_DraftFiltering(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "content/_index.md", "---\ntitle: Home\n---\n# Home\n")
	writeFixture(t, dir, "content/blog/_index.md", "---\ntitle: Blog\n---\n")
	writeFixture(t, dir, "content/blog/published.md", "---\ntitle: Published\ndate: 2025-01-01T00:00:00Z\n---\nPublished.\n")
	writeFixture(t, dir, "content/blog/draft.md", "---\ntitle: Draft Post\ndraft: true\ndate: 2025-01-01T00:00:00Z\n---\nDraft.\n")

	cfg := config.Defaults()
	themeCfg := buildThemeConfig()

	builder := NewSiteBuilder(BuildOptions{
		ProjectDir:  dir,
		Config:      cfg,
		ThemeConfig: themeCfg,
		EmbeddedFS:  embedded.ThemeFS(),
	})

	result, err := builder.Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	distDir := filepath.Join(dir, "dist")

	// Published should exist.
	assertFixtureFileExists(t, distDir, "blog/published/index.html")

	// Draft should NOT exist.
	draftPath := filepath.Join(distDir, "blog", "draft", "index.html")
	if _, err := os.Stat(draftPath); !os.IsNotExist(err) {
		t.Error("draft page should not be in output")
	}

	_ = result
}

func TestBuild_StaticFiles(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "content/_index.md", "---\ntitle: Home\n---\n# Home\n")
	writeFixture(t, dir, "static/robots.txt", "User-agent: *\nAllow: /\n")
	writeFixture(t, dir, "static/images/logo.svg", "<svg></svg>")

	cfg := config.Defaults()
	themeCfg := buildThemeConfig()

	builder := NewSiteBuilder(BuildOptions{
		ProjectDir:  dir,
		Config:      cfg,
		ThemeConfig: themeCfg,
		EmbeddedFS:  embedded.ThemeFS(),
	})

	_, err := builder.Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	distDir := filepath.Join(dir, "dist")
	assertFixtureFileContains(t, distDir, "robots.txt", "User-agent")
	assertFixtureFileContains(t, distDir, "images/logo.svg", "<svg>")
}

func TestBuild_RejectsUnsafeOutputDir(t *testing.T) {
	dir := createFixtureSite(t)
	cfg := config.Defaults()
	cfg.Build.Output = "."

	builder := NewSiteBuilder(BuildOptions{
		ProjectDir:  dir,
		Config:      cfg,
		ThemeConfig: buildThemeConfig(),
		EmbeddedFS:  embedded.ThemeFS(),
	})

	if _, err := builder.Build(); err == nil {
		t.Fatal("expected build to reject project-root output")
	}
}

func TestBuild_RejectsUnsafeAliasOutputPath(t *testing.T) {
	dir := createFixtureSite(t)
	writeFixture(t, dir, "content/about.md", "---\ntitle: About\naliases: [\"/../escape/\"]\n---\nAbout.\n")
	cfg := config.Defaults()

	builder := NewSiteBuilder(BuildOptions{
		ProjectDir:  dir,
		Config:      cfg,
		ThemeConfig: buildThemeConfig(),
		EmbeddedFS:  embedded.ThemeFS(),
	})

	if _, err := builder.Build(); err == nil {
		t.Fatal("expected build to reject alias path traversal")
	}
	if _, err := os.Stat(filepath.Join(dir, "escape", "index.html")); !os.IsNotExist(err) {
		t.Fatalf("unsafe alias escaped output dir, stat err = %v", err)
	}
}

func TestBuild_ReusedBuilderRefreshesAssetManifestInTemplates(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "content/about.md", "---\ntitle: About\n---\nAbout.\n")
	writeFixture(t, dir, "assets/css/main.css", "body { color: red; }\n")
	writeFixture(t, dir, "layouts/_default/baseof.html", `<!doctype html><html><head><link href="{{ fingerprint "css/main.css" }}"></head><body>{{ block "content" . }}{{ end }}</body></html>`)
	writeFixture(t, dir, "layouts/_default/single.html", `{{ define "content" }}{{ .Page.Content }}{{ end }}`)

	cfg := config.Defaults()
	cfg.Head.CustomCSS = []string{"css/main.css"}
	builder := NewSiteBuilder(BuildOptions{
		ProjectDir:  dir,
		Config:      cfg,
		ThemeConfig: buildThemeConfig(),
		EmbeddedFS:  embedded.ThemeFS(),
	})

	if _, err := builder.Build(); err != nil {
		t.Fatalf("first Build failed: %v", err)
	}
	first := readFixtureFile(t, filepath.Join(dir, "dist", "about", "index.html"))

	writeFixture(t, dir, "assets/css/main.css", "body { color: blue; }\n")
	if _, err := builder.Build(); err != nil {
		t.Fatalf("second Build failed: %v", err)
	}
	second := readFixtureFile(t, filepath.Join(dir, "dist", "about", "index.html"))

	if first == second {
		t.Fatalf("expected fingerprinted asset URL to change after asset edit; HTML stayed %q", second)
	}
	if !strings.Contains(second, "/assets/css/main.") {
		t.Fatalf("second HTML did not contain fingerprinted CSS URL: %s", second)
	}
}

func TestValidate(t *testing.T) {
	projDir := createFixtureSite(t)
	cfg := config.Defaults()

	builder := NewSiteBuilder(BuildOptions{
		ProjectDir:  projDir,
		Config:      cfg,
		ThemeConfig: buildThemeConfig(),
		EmbeddedFS:  embedded.ThemeFS(),
	})

	result, err := builder.Validate()
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	if result.PageCount == 0 {
		t.Error("expected pages")
	}
	if result.Collections == 0 {
		t.Error("expected collections")
	}

	// Validate should NOT create output directory.
	distDir := filepath.Join(projDir, "dist")
	if _, err := os.Stat(distDir); !os.IsNotExist(err) {
		t.Error("validate should not create output directory")
	}
}

func assertFixtureFileExists(t *testing.T, base, rel string) {
	t.Helper()
	path := filepath.Join(base, filepath.FromSlash(rel))
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("expected file to exist: %s", rel)
	}
}

func assertFixtureFileContains(t *testing.T, base, rel, substr string) {
	t.Helper()
	path := filepath.Join(base, filepath.FromSlash(rel))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", rel, err)
	}
	if !strings.Contains(string(data), substr) {
		t.Errorf("file %s does not contain %q", rel, substr)
	}
}

func assertFixtureFileNotContains(t *testing.T, base, rel, substr string) {
	t.Helper()
	path := filepath.Join(base, filepath.FromSlash(rel))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", rel, err)
	}
	if strings.Contains(string(data), substr) {
		t.Errorf("file %s unexpectedly contains %q", rel, substr)
	}
}

func readFixtureFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(data)
}

func TestBuild_Custom404(t *testing.T) {
	projDir := createFixtureSite(t)
	writeFixture(t, projDir, "content/404.md", "---\ntitle: Oops!\n---\n# Custom Not Found\n\nSorry, this page doesn't exist.\n")

	cfg := config.Defaults()
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
	assertFixtureFileExists(t, distDir, "404.html")
	assertFixtureFileContains(t, distDir, "404.html", "Custom Not Found")
	assertFixtureFileContains(t, distDir, "404.html", "this page")
}

func TestBuild_Custom404_TemplateVariant(t *testing.T) {
	projDir := createFixtureSite(t)
	writeFixture(t, projDir, "content/404.md", "---\ntitle: Gone\ntemplate: 404-minimal\n---\nMinimal error page.\n")

	cfg := config.Defaults()
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
	assertFixtureFileExists(t, distDir, "404.html")
	assertFixtureFileContains(t, distDir, "404.html", "error-page--minimal")
}

func TestBuild_NoCustom404_UsesDefault(t *testing.T) {
	projDir := createFixtureSite(t)
	// No content/404.md — should use hardcoded fallback.

	cfg := config.Defaults()
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
	assertFixtureFileExists(t, distDir, "404.html")
	assertFixtureFileContains(t, distDir, "404.html", "Page Not Found")
}
