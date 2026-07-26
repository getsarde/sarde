package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/getsarde/sarde/embedded"
	"github.com/getsarde/sarde/internal/config"
)

// createPageMetaFixtureSite creates a docs site with one page carrying an
// explicit `updated` frontmatter date, so the rendered "last updated" text is
// deterministic regardless of the last_updated strategy or git availability.
func createPageMetaFixtureSite(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeFixture(t, dir, "content/_index.md", "---\ntitle: Home\n---\n# Home\n")
	writeFixture(t, dir, "content/docs/_index.md", "---\ntitle: Docs\n---\n")
	writeFixture(t, dir, "content/docs/guide.md", "---\ntitle: Guide\nweight: 1\nupdated: 2025-06-01T00:00:00Z\n---\n# Guide\nBody.\n")
	return dir
}

// buildPageMetaSite builds the fixture with the given theme.date_format and
// returns the rendered docs/guide page. The raw preset name is passed
// through: since the locale-aware date work, presets are resolved at format
// time by the dateFormat template function, not at config load.
func buildPageMetaSite(t *testing.T, dateFormat string) string {
	t.Helper()
	dir := createPageMetaFixtureSite(t)
	cfg := config.Defaults()
	cfg.Build.Minify = config.BoolPtr(false)
	if dateFormat != "" {
		cfg.Theme.DateFormat = dateFormat
	}

	builder := NewSiteBuilder(BuildOptions{
		ProjectDir:  dir,
		Config:      cfg,
		ThemeConfig: buildThemeConfig(),
		EmbeddedFS:  embedded.ThemeFS(),
	})
	if _, err := builder.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "dist", "docs", "guide", "index.html"))
	if err != nil {
		t.Fatalf("reading rendered guide page: %v", err)
	}
	return string(data)
}

// TestBuild_PageMeta_LastUpdatedRendersOnDocsPage covers the meta row that
// groups the edit link and the last-updated date: the sarde-page-meta
// wrapper, the machine-readable datetime attribute, and the i18n label all
// render on a docs single page.
func TestBuild_PageMeta_LastUpdatedRendersOnDocsPage(t *testing.T) {
	html := buildPageMetaSite(t, "") // default "short" preset

	if !strings.Contains(html, `<div class="sarde-page-meta">`) {
		t.Error("expected the sarde-page-meta wrapper around EditLink/LastUpdated")
	}
	if !strings.Contains(html, `<time datetime="2025-06-01">`) {
		t.Error("expected a machine-readable datetime attribute for the updated date")
	}
	if !strings.Contains(html, "Last updated") {
		t.Error("expected the i18n last_updated_label text")
	}
	if !strings.Contains(html, "Jun 1, 2025") {
		t.Error("expected the short-format visible date")
	}
}

// TestBuild_PageMeta_DateFormatLongChangesDisplay covers theme.date_format
// actually reaching the rendered output, not just NormalizeDateFormat's own
// unit tests in internal/config.
func TestBuild_PageMeta_DateFormatLongChangesDisplay(t *testing.T) {
	html := buildPageMetaSite(t, "long")

	if !strings.Contains(html, "June 1, 2025") {
		t.Error("theme.date_format: long should render the long month name")
	}
	if strings.Contains(html, "Jun 1, 2025") {
		t.Error("theme.date_format: long should not render the short-format date")
	}
}

// TestBuild_PageMeta_FrenchPageLocalizesDate covers the locale-aware presets
// end to end on a multilingual site: the same theme.date_format preset
// renders CLDR French on the fr page while the en page keeps the old English
// output byte for byte.
func TestBuild_PageMeta_FrenchPageLocalizesDate(t *testing.T) {
	dir := t.TempDir()
	page := "---\ntitle: Guide\nweight: 1\nupdated: 2025-06-01T00:00:00Z\n---\n# Guide\nBody.\n"
	writeFixture(t, dir, "content/_index.md", "---\ntitle: Home\n---\n# Home\n")
	writeFixture(t, dir, "content/docs/_index.md", "---\ntitle: Docs\n---\n")
	writeFixture(t, dir, "content/docs/guide.md", page)
	writeFixture(t, dir, "content/fr/_index.md", "---\ntitle: Accueil\n---\n# Accueil\n")
	writeFixture(t, dir, "content/fr/docs/_index.md", "---\ntitle: Docs\n---\n")
	writeFixture(t, dir, "content/fr/docs/guide.md", page)

	cfg := config.Defaults()
	cfg.Build.Minify = config.BoolPtr(false)
	cfg.I18n.DefaultLanguage = "en"
	cfg.I18n.Languages = map[string]config.LanguageConfig{
		"en": {Name: "English", Weight: 1, Dir: "ltr"},
		"fr": {Name: "French", Weight: 2, Dir: "ltr"},
	}

	builder := NewSiteBuilder(BuildOptions{
		ProjectDir:  dir,
		Config:      cfg,
		ThemeConfig: buildThemeConfig(),
		EmbeddedFS:  embedded.ThemeFS(),
	})
	if _, err := builder.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	read := func(rel string) string {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(dir, "dist", filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("reading %s: %v", rel, err)
		}
		return string(data)
	}

	enHTML := read("docs/guide/index.html")
	if !strings.Contains(enHTML, "Jun 1, 2025") {
		t.Error("en page should keep the English short date")
	}

	frHTML := read("fr/docs/guide/index.html")
	if !strings.Contains(frHTML, "1 juin 2025") {
		t.Error("fr page should render the CLDR French date")
	}
	if strings.Contains(frHTML, "Jun 1, 2025") {
		t.Error("fr page must not render the English date")
	}
	if !strings.Contains(frHTML, `datetime="2025-06-01"`) {
		t.Error("fr page must keep the ISO datetime attribute")
	}
}
