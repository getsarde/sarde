package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/getsarde/sarde/embedded"
	"github.com/getsarde/sarde/internal/config"
)

func svgFile(label string) string {
	return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"><title>` + label + `</title></svg>`
}

// TestBuild_TranslatedBundlesWriteBesideEachPage: the English and French
// copies of a translated bundle each have their own file at their own URL,
// and the fallback copies of an untranslated bundle get the files beside
// them, so a relative link in the body works on every copy.
func TestBuild_TranslatedBundlesWriteBesideEachPage(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "content/_index.md", "---\ntitle: Home\n---\n")
	writeFixture(t, dir, "content/fr/_index.md", "---\ntitle: Accueil\n---\n")
	writeFixture(t, dir, "content/de/_index.md", "---\ntitle: Startseite\n---\n")
	writeFixture(t, dir, "content/blog/trip/index.md", "---\ntitle: Trip\ndate: 2025-05-01\n---\n![Map](map.svg)\n")
	writeFixture(t, dir, "content/blog/trip/map.svg", svgFile("english"))
	writeFixture(t, dir, "content/fr/blog/trip/index.md", "---\ntitle: Sortie\ndate: 2025-05-01\n---\n![Carte](map.svg)\n")
	writeFixture(t, dir, "content/fr/blog/trip/map.svg", svgFile("french"))
	writeFixture(t, dir, "content/blog/solo/index.md", "---\ntitle: Solo\ndate: 2025-05-02\n---\n[Report](report.pdf)\n")
	writeFixture(t, dir, "content/blog/solo/report.pdf", "%PDF-1.4 english")

	cfg := cascadeI18nConfig()
	cfg.I18n.Fallback = "default"
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

	for rel, want := range map[string]string{
		"blog/trip/map.svg":       "english",
		"fr/blog/trip/map.svg":    "french",
		"de/blog/trip/map.svg":    "english",
		"blog/solo/report.pdf":    "english",
		"fr/blog/solo/report.pdf": "english",
		"de/blog/solo/report.pdf": "english",
	} {
		data, err := os.ReadFile(filepath.Join(dist, filepath.FromSlash(rel)))
		if err != nil {
			t.Errorf("%s not written: %v", rel, err)
			continue
		}
		if !strings.Contains(string(data), want) {
			t.Errorf("%s holds %q, want the %s file", rel, data, want)
		}
	}
	assertFixtureFileContains(t, dist, "blog/trip/index.html", `src="/blog/trip/map.svg"`)
	assertFixtureFileContains(t, dist, "fr/blog/trip/index.html", `src="/fr/blog/trip/map.svg"`)
	assertFixtureFileContains(t, dist, "de/blog/trip/index.html", `src="/de/blog/trip/map.svg"`)
}

// TestBuild_VersionedBundlesWriteBesideEachPage: a v1 page and its latest
// twin with the same body each link their own version's file, including when
// the second build renders from the page cache.
func TestBuild_VersionedBundlesWriteBesideEachPage(t *testing.T) {
	dir := t.TempDir()
	body := "---\ntitle: Guide\n---\n![Flow](flow.svg)\n"
	writeFixture(t, dir, "content/_index.md", "---\ntitle: Home\n---\n")
	writeFixture(t, dir, "content/docs/_index.md", "---\ntitle: Docs\n---\n")
	writeFixture(t, dir, "content/docs/guide/index.md", body)
	writeFixture(t, dir, "content/docs/guide/flow.svg", svgFile("v2"))
	writeFixture(t, dir, "content/docs/v1/_index.md", "---\ntitle: Docs\n---\n")
	writeFixture(t, dir, "content/docs/v1/guide/index.md", body)
	writeFixture(t, dir, "content/docs/v1/guide/flow.svg", svgFile("v1"))

	cfg := config.Defaults()
	cfg.Build.Minify = config.BoolPtr(false)
	cfg.Build.Cache = config.BoolPtr(true)
	cfg.Build.LastUpdated = "false"
	cfg.Collections = map[string]*config.CollectionSiteConfig{
		"docs": {Versioning: &config.VersioningConfig{
			Enabled: config.BoolPtr(true), LastVersion: "v2",
			Versions: []config.VersionEntry{{ID: "v2"}, {ID: "v1"}},
		}},
	}
	dist := filepath.Join(dir, "dist")
	for build := 1; build <= 2; build++ {
		builder := NewSiteBuilder(BuildOptions{
			ProjectDir:  dir,
			Config:      cfg,
			ThemeConfig: buildThemeConfig(),
			EmbeddedFS:  embedded.ThemeFS(),
		})
		if _, err := builder.Build(); err != nil {
			t.Fatalf("build %d failed: %v", build, err)
		}
		assertFixtureFileContains(t, dist, "docs/guide/index.html", `src="/docs/guide/flow.svg"`)
		assertFixtureFileContains(t, dist, "docs/v1/guide/index.html", `src="/docs/v1/guide/flow.svg"`)
	}
	for rel, want := range map[string]string{"docs/guide/flow.svg": "v2", "docs/v1/guide/flow.svg": "v1"} {
		data, err := os.ReadFile(filepath.Join(dist, filepath.FromSlash(rel)))
		if err != nil || !strings.Contains(string(data), want) {
			t.Errorf("%s = %q (%v), want the %s file", rel, data, err, want)
		}
	}
}
