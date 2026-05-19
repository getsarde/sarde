package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coderoo-dev/coderoo/internal/config"
	"github.com/coderoo-dev/coderoo/embedded"
	"github.com/coderoo-dev/coderoo/internal/engine"
	"github.com/coderoo-dev/coderoo/internal/theme"
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
