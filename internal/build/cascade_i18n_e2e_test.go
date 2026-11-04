package build

import (
	"path/filepath"
	"testing"

	"github.com/getsarde/sarde/embedded"
	"github.com/getsarde/sarde/internal/config"
)

// writeCascadeI18nSite writes a bilingual docs collection whose guide
// _index.md cascades a different banner in each language, and whose api
// folder has an _index.md in English only.
func writeCascadeI18nSite(t *testing.T, dir string) {
	t.Helper()
	writeFixture(t, dir, "content/_index.md", "---\ntitle: Home\n---\n")
	writeFixture(t, dir, "content/fr/_index.md", "---\ntitle: Accueil\n---\n")
	writeFixture(t, dir, "content/docs/_index.md", "---\ntitle: Docs\n---\n")
	writeFixture(t, dir, "content/fr/docs/_index.md", "---\ntitle: Docs\n---\n")
	writeFixture(t, dir, "content/docs/guide/_index.md",
		"---\ntitle: Guide\nsidebar:\n  order: 10\ncascade:\n  banner:\n    content: English guide notice\n---\n")
	writeFixture(t, dir, "content/fr/docs/guide/_index.md",
		"---\ntitle: Guide\nsidebar:\n  order: 10\ncascade:\n  banner:\n    content: Avis du guide\n---\n")
	writeFixture(t, dir, "content/docs/guide/intro.md", "---\ntitle: Intro\n---\nHello.\n")
	writeFixture(t, dir, "content/fr/docs/guide/intro.md", "---\ntitle: Intro\n---\nBonjour.\n")
	writeFixture(t, dir, "content/docs/api/_index.md",
		"---\ntitle: API\ncascade:\n  banner:\n    content: API notice\n---\n")
	writeFixture(t, dir, "content/docs/api/endpoints.md", "---\ntitle: Endpoints\n---\nList.\n")
	writeFixture(t, dir, "content/fr/docs/api/endpoints.md", "---\ntitle: Points d'accès\n---\nListe.\n")
}

func cascadeI18nConfig() *config.SiteConfig {
	cfg := config.Defaults()
	cfg.Build.Minify = config.BoolPtr(false)
	cfg.Build.Cache = config.BoolPtr(false)
	cfg.Build.LastUpdated = "false"
	cfg.I18n.DefaultLanguage = "en"
	cfg.I18n.Languages = map[string]config.LanguageConfig{
		"en": {Name: "English", Weight: 1, Dir: "ltr"},
		"fr": {Name: "Français", Weight: 2, Dir: "ltr"},
	}
	return cfg
}

// TestBuild_CascadePerLanguage: each language's pages inherit the cascade of
// that language's _index.md, although both files share one folder slot in
// the collection's section tree. A French page in a folder with no French
// _index.md inherits the English one.
func TestBuild_CascadePerLanguage(t *testing.T) {
	dir := t.TempDir()
	writeCascadeI18nSite(t, dir)
	builder := NewSiteBuilder(BuildOptions{
		ProjectDir:  dir,
		Config:      cascadeI18nConfig(),
		ThemeConfig: buildThemeConfig(),
		EmbeddedFS:  embedded.ThemeFS(),
	})
	if _, err := builder.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	dist := filepath.Join(dir, "dist")

	assertFixtureFileContains(t, dist, "docs/guide/intro/index.html", "English guide notice")
	assertFixtureFileNotContains(t, dist, "docs/guide/intro/index.html", "Avis du guide")
	assertFixtureFileContains(t, dist, "fr/docs/guide/intro/index.html", "Avis du guide")
	assertFixtureFileNotContains(t, dist, "fr/docs/guide/intro/index.html", "English guide notice")
	assertFixtureFileContains(t, dist, "fr/docs/api/endpoints/index.html", "API notice")
}

// TestBuild_GeneratedListPagePerLanguage: collections without an _index.md
// get a generated list page in every language, whichever language wrote the
// newest post. The pages link to each other with hreflang, and a language
// with no posts gets fallback list pages for every collection.
func TestBuild_GeneratedListPagePerLanguage(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "content/_index.md", "---\ntitle: Home\n---\n")
	writeFixture(t, dir, "content/fr/_index.md", "---\ntitle: Accueil\n---\n")
	writeFixture(t, dir, "content/de/_index.md", "---\ntitle: Startseite\n---\n")
	writeFixture(t, dir, "content/blog/old.md", "---\ntitle: Old Post\ndate: 2026-01-01\n---\nOld.\n")
	writeFixture(t, dir, "content/fr/blog/neuf.md", "---\ntitle: Billet neuf\ndate: 2026-03-01\n---\nNeuf.\n")
	writeFixture(t, dir, "content/news/launch.md", "---\ntitle: Launch\ndate: 2026-01-02\n---\nLaunch.\n")
	writeFixture(t, dir, "content/fr/news/lancement.md", "---\ntitle: Lancement\ndate: 2026-03-02\n---\nLancement.\n")

	cfg := cascadeI18nConfig()
	cfg.Site.URL = "https://example.com"
	cfg.I18n.Languages["de"] = config.LanguageConfig{Name: "Deutsch", Weight: 3, Dir: "ltr"}
	builder := NewSiteBuilder(BuildOptions{
		ProjectDir:  dir,
		Config:      cfg,
		ThemeConfig: buildThemeConfig(),
		EmbeddedFS:  embedded.ThemeFS(),
	})
	if _, err := builder.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	dist := filepath.Join(dir, "dist")

	for _, col := range []string{"blog", "news"} {
		en := col + "/index.html"
		fr := "fr/" + col + "/index.html"
		de := "de/" + col + "/index.html"
		assertFixtureFileExists(t, dist, en)
		assertFixtureFileExists(t, dist, fr)
		assertFixtureFileExists(t, dist, de)
		assertFixtureFileContains(t, dist, en, `hreflang="fr" href="https://example.com/fr/`+col+`/"`)
		assertFixtureFileContains(t, dist, fr, `hreflang="en" href="https://example.com/`+col+`/"`)
	}
	assertFixtureFileContains(t, dist, "blog/index.html", "Old Post")
	assertFixtureFileNotContains(t, dist, "blog/index.html", "Billet neuf")
	assertFixtureFileContains(t, dist, "fr/blog/index.html", "Billet neuf")
}
