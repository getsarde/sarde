package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/getsarde/sarde/embedded"
	"github.com/getsarde/sarde/internal/config"
)

// themeTogglePages are pages rendered by each of the four embedded bases:
// _default, _docs, _labs and _presentation.
var themeTogglePages = []string{
	"index.html",
	"docs/guide/index.html",
	"labs/web/build/scaffold/index.html",
	"slides/deck/index.html",
}

// buildThemeToggleSite builds the rich fixture plus a lab step and a slide deck
// and returns the dist directory and the HTML of themeTogglePages.
func buildThemeToggleSite(t *testing.T, cfg *config.SiteConfig) (string, map[string]string) {
	t.Helper()
	projDir := createRichFixtureSite(t)
	writeFixture(t, projDir, "content/labs/web/_index.md", "---\ntitle: Web\n---\n")
	writeFixture(t, projDir, "content/labs/web/build/_index.md", "---\ntitle: Build\n---\nIntro.\n")
	writeFixture(t, projDir, "content/labs/web/build/scaffold.md", "---\ntitle: Scaffold\n---\n## Step\nDo it.\n")
	writeFixture(t, projDir, "content/slides/deck.md", "---\ntitle: Deck\n---\n# One\n\n---\n\n# Two\n")
	cfg.Build.Minify = config.BoolPtr(false)
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
	pages := make(map[string]string, len(themeTogglePages))
	for _, p := range themeTogglePages {
		pages[p] = readFixture(t, distDir, p)
	}
	return distDir, pages
}

const themeScript = "localStorage.getItem('sd-theme')"

func TestBuild_ThemeToggle_Default(t *testing.T) {
	_, pages := buildThemeToggleSite(t, config.Defaults())

	// Each fixture page comes from a different base.
	for p, want := range map[string]string{
		"docs/guide/index.html":              "sd-docs-centered",
		"labs/web/build/scaffold/index.html": `class="sarde-layout-labs"`,
		"slides/deck/index.html":             `class="sarde-layout-presentation"`,
	} {
		if !strings.Contains(pages[p], want) {
			t.Errorf("%s: missing %q from its base", p, want)
		}
	}
	for p, html := range pages {
		if !strings.Contains(html, themeScript) {
			t.Errorf("%s: missing the sd-theme script", p)
		}
		if strings.Count(html, `media="(prefers-color-scheme:`) != 2 {
			t.Errorf("%s: want a light and a dark theme-color meta", p)
		}
		if strings.Contains(html, "data-sd-light-only") {
			t.Errorf("%s: data-sd-light-only without theme.dark: false", p)
		}
	}
	if !strings.Contains(pages["index.html"], "data-sarde-theme-toggle") {
		t.Error("home page: missing the theme toggle")
	}
	// The docs page carries one in the header and one in the sidebar footer.
	if n := strings.Count(pages["docs/guide/index.html"], "data-sarde-theme-toggle"); n != 2 {
		t.Errorf("docs page: %d theme toggles, want 2", n)
	}
}

// header.theme_toggle: false hides the toggle; dark mode still follows the
// visitor's system setting.
func TestBuild_ThemeToggle_HeaderThemeToggleHidden_KeepsDarkMode(t *testing.T) {
	cfg := config.Defaults()
	cfg.Header.ThemeToggle = config.BoolPtr(false)
	_, pages := buildThemeToggleSite(t, cfg)

	for p, html := range pages {
		if strings.Contains(html, "data-sarde-theme-toggle") {
			t.Errorf("%s: theme toggle rendered with header.theme_toggle: false", p)
		}
		if !strings.Contains(html, themeScript) {
			t.Errorf("%s: missing the sd-theme script", p)
		}
		if strings.Count(html, `media="(prefers-color-scheme:`) != 2 {
			t.Errorf("%s: want a light and a dark theme-color meta", p)
		}
		if strings.Contains(html, "data-sd-light-only") {
			t.Errorf("%s: data-sd-light-only with dark mode on", p)
		}
	}
}

// theme.dark: false makes the site light-only: no toggle, data-theme="light"
// rendered on <html>, and no script that could switch it to dark.
func TestBuild_ThemeToggle_ThemeDarkFalse_LightOnly(t *testing.T) {
	cfg := config.Defaults()
	cfg.Theme.Dark = config.BoolPtr(false)
	distDir, pages := buildThemeToggleSite(t, cfg)

	for p, html := range pages {
		if !strings.Contains(html, `<html lang="en" dir="ltr" data-theme="light" data-sd-light-only>`) {
			t.Errorf("%s: <html> is not marked light-only", p)
		}
		if strings.Contains(html, "data-sarde-theme-toggle") {
			t.Errorf("%s: theme toggle rendered with theme.dark: false", p)
		}
		if strings.Contains(html, "sd-theme") {
			t.Errorf("%s: the sd-theme script still runs", p)
		}
		if n := strings.Count(html, `name="theme-color"`); n != 1 {
			t.Errorf("%s: %d theme-color metas, want 1", p, n)
		}
	}
	if !strings.Contains(pages["docs/guide/index.html"], "localStorage.getItem('sd-sidebar')") {
		t.Error("docs page: the sidebar state script is gone too")
	}

	bundles, _ := filepath.Glob(filepath.Join(distDir, "assets", "js", "sarde*.js"))
	if len(bundles) != 1 {
		t.Fatalf("want one sarde.js bundle, got %v", bundles)
	}
	js, err := os.ReadFile(bundles[0])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(js), "data-sd-light-only") {
		t.Error("sarde.js does not stop the theme script on a light-only site")
	}
}
