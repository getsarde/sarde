package build

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/getsarde/sarde/embedded"
	"github.com/getsarde/sarde/internal/config"
)

// writeTabbedSite writes a "courses" collection with two tabs. rootBody is
// the body of courses/_index.md; empty keeps it frontmatter-only.
func writeTabbedSite(t *testing.T, rootBody string) string {
	t.Helper()
	dir := t.TempDir()
	writeFixture(t, dir, "content/_index.md", "---\ntitle: Home\n---\n# Home\n")
	writeFixture(t, dir, "content/courses/_index.md", "---\ntitle: Courses\ndescription: All courses.\n---\n"+rootBody)
	writeFixture(t, dir, "content/courses/web/_index.md", "---\ntitle: Web Basics\ndescription: HTML and CSS.\nsidebar:\n  order: 1\n  badge:\n    text: Beginner\n    variant: tip\n---\nWeb overview.\n")
	writeFixture(t, dir, "content/courses/web/html.md", "---\ntitle: HTML\n---\nHTML lesson.\n")
	writeFixture(t, dir, "content/courses/go/_index.md", "---\ntitle: Go Basics\ndescription: The Go language.\nsidebar:\n  order: 2\n---\nGo overview.\n")
	writeFixture(t, dir, "content/courses/go/types.md", "---\ntitle: Types\n---\nTypes lesson.\n")
	return dir
}

func TestBuild_TabbedRootWithoutBodyRedirects(t *testing.T) {
	dir := writeTabbedSite(t, "")
	builder := newIncrementalBuilder(dir, config.Defaults())
	if _, err := builder.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	html := readFixture(t, filepath.Join(dir, "dist"), "courses/index.html")
	if !strings.Contains(html, `http-equiv="refresh" content="0;url=/courses/web/"`) {
		t.Errorf("expected a redirect to the first tab, got:\n%s", html)
	}
}

func TestBuild_TabbedRootWithBodyIsCatalog(t *testing.T) {
	dir := writeTabbedSite(t, "Pick a course to start.\n")
	builder := newIncrementalBuilder(dir, config.Defaults())
	if _, err := builder.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	html := readFixture(t, filepath.Join(dir, "dist"), "courses/index.html")
	if strings.Contains(html, `http-equiv="refresh"`) {
		t.Fatal("a root with a body must render as a page, not redirect")
	}
	for _, want := range []string{
		"Pick a course to start.",
		`class="sarde-card-grid sarde-catalog-grid"`,
		`<span class="sarde-badge sarde-badge-tip">Beginner</span>`,
		"The Go language.",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("catalog missing %q", want)
		}
	}
	web := strings.Index(html, `href="/courses/web/" class="sarde-doc-card sarde-catalog-card"`)
	goTab := strings.Index(html, `href="/courses/go/" class="sarde-doc-card sarde-catalog-card"`)
	if web == -1 || goTab == -1 || web > goTab {
		t.Errorf("want one card per tab in tab order (web before go), got positions %d, %d", web, goTab)
	}
	for _, unwanted := range []string{`id="sarde-sidebar"`, "sarde-tab-switcher"} {
		if strings.Contains(html, unwanted) {
			t.Errorf("catalog should have no sidebar or tab switcher, found %q", unwanted)
		}
	}
}

// In sarde dev, editing a course description must refresh its catalog card,
// which reads tab data copied when the collection registry builds. An
// _index.md edit falls back to a full rebuild, which rebuilds the tabs.
func TestContentRebuild_TabDescriptionRefreshesCatalog(t *testing.T) {
	dir := writeTabbedSite(t, "Pick a course to start.\n")
	builder := newIncrementalBuilder(dir, config.Defaults())
	if _, err := builder.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	writeFixture(t, dir, "content/courses/go/_index.md", "---\ntitle: Go Basics\ndescription: Learn Go from scratch.\nsidebar:\n  order: 2\n---\nGo overview.\n")
	if _, err := builder.ContentRebuild([]string{filepath.Join(dir, "content", "courses", "go", "_index.md")}); err != nil {
		t.Fatalf("ContentRebuild failed: %v", err)
	}

	html := readFixture(t, filepath.Join(dir, "dist"), "courses/index.html")
	if !strings.Contains(html, "Learn Go from scratch.") {
		t.Error("catalog card still shows the old course description after a dev rebuild")
	}
}

// A nav.yaml url to another site keeps its URL and, with external: true, opens
// in a new tab; a url that is a site path still gets the base path.
func TestBuild_NavYAMLExternalURL(t *testing.T) {
	navYAML := `- page: html
- label: Example
  url: "https://example.com"
  external: true
- label: Go course
  url: "/courses/go/"
`
	for _, tc := range []struct{ name, basePath, goHref string }{
		{"root", "/", `href="/courses/go/"`},
		{"base path", "/sub/", `href="/sub/courses/go/"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := writeTabbedSite(t, "")
			writeFixture(t, dir, "content/courses/web/nav.yaml", navYAML)
			cfg := config.Defaults()
			cfg.Build.BasePath = tc.basePath
			builder := NewSiteBuilder(BuildOptions{
				ProjectDir:  dir,
				Config:      cfg,
				ThemeConfig: buildThemeConfig(),
				EmbeddedFS:  embedded.ThemeFS(),
			})
			if _, err := builder.Build(); err != nil {
				t.Fatalf("Build failed: %v", err)
			}

			html := readFixture(t, filepath.Join(dir, "dist"), "courses/web/html/index.html")
			for _, want := range []string{
				`href="https://example.com" target="_blank" rel="noopener noreferrer"`,
				`<span class="sr-only"> (opens in new tab)</span>`,
				tc.goHref,
			} {
				if !strings.Contains(html, want) {
					t.Errorf("sidebar is missing %s", want)
				}
			}
			if strings.Contains(html, "https:/example.com") {
				t.Error("the full URL went through the site path resolver")
			}
		})
	}
}

// writeTabbedI18nSite writes writeTabbedSite's courses in English and
// French. The French web tab has its own _index.md with the given sidebar
// order; the go tab has no French files, so French gets fallback copies.
func writeTabbedI18nSite(t *testing.T, rootBody string, frWebOrder int) string {
	t.Helper()
	dir := writeTabbedSite(t, rootBody)
	writeFixture(t, dir, "content/fr/_index.md", "---\ntitle: Accueil\n---\n# Accueil\n")
	writeFixture(t, dir, "content/fr/courses/_index.md", "---\ntitle: Cours\n---\n"+rootBody)
	writeFixture(t, dir, "content/fr/courses/web/_index.md", fmt.Sprintf("---\ntitle: Bases du Web\nsidebar:\n  order: %d\n---\nPrésentation.\n", frWebOrder))
	writeFixture(t, dir, "content/fr/courses/web/html.md", "---\ntitle: HTML\n---\nLeçon HTML.\n")
	return dir
}

// Both web/_index.md files share one folder slot in the collection's section
// tree, and the English one wins it. French pages must still show the French
// tab label, and the French catalog lists the untranslated go tab through its
// fallback copy.
func TestBuild_TabLabelsFollowPageLanguage(t *testing.T) {
	dir := writeTabbedI18nSite(t, "Pick a course to start.\n", 1)
	builder := newIncrementalBuilder(dir, cascadeI18nConfig())
	if _, err := builder.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	dist := filepath.Join(dir, "dist")

	assertFixtureFileContains(t, dist, "fr/courses/web/html/index.html", "Bases du Web")
	assertFixtureFileNotContains(t, dist, "fr/courses/web/html/index.html", "Web Basics")
	assertFixtureFileContains(t, dist, "courses/web/html/index.html", "Web Basics")
	assertFixtureFileNotContains(t, dist, "courses/web/html/index.html", "Bases du Web")

	assertFixtureFileContains(t, dist, "fr/courses/index.html", "Bases du Web")
	assertFixtureFileContains(t, dist, "fr/courses/index.html", "Go Basics")
}

// The French root forwards to the first tab in French order: the French web
// tab sorts after go.
func TestBuild_TabbedRootRedirectFollowsLanguageOrder(t *testing.T) {
	dir := writeTabbedI18nSite(t, "", 3)
	builder := newIncrementalBuilder(dir, cascadeI18nConfig())
	if _, err := builder.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	dist := filepath.Join(dir, "dist")

	assertFixtureFileContains(t, dist, "courses/index.html", `content="0;url=/courses/web/"`)
	assertFixtureFileContains(t, dist, "fr/courses/index.html", `content="0;url=/fr/courses/go/"`)
}
