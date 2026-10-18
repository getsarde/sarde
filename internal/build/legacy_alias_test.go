package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/getsarde/sarde/embedded"
	"github.com/getsarde/sarde/internal/config"
)

// legacyAliasConfig is a Netlify site with the redirects plugin, so one
// build writes both the HTML stubs and _redirects.
func legacyAliasConfig() *config.SiteConfig {
	cfg := config.Defaults()
	cfg.Build.LastUpdated = "false"
	cfg.Deploy.Provider = "netlify"
	cfg.Plugins.Enabled = []string{"redirects"}
	return cfg
}

func buildLegacyAliasSite(t *testing.T, dir string, cfg *config.SiteConfig) *SiteBuilder {
	t.Helper()
	builder := NewSiteBuilder(BuildOptions{
		ProjectDir:  dir,
		Config:      cfg,
		ThemeConfig: buildThemeConfig(),
		EmbeddedFS:  embedded.ThemeFS(),
	})
	if _, err := builder.Build(); err != nil {
		t.Fatalf("Build: %v", err)
	}
	return builder
}

func readOutput(t *testing.T, dir, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "dist", filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("reading %s: %v", rel, err)
	}
	return string(data)
}

func outputExists(dir, rel string) bool {
	_, err := os.Stat(filepath.Join(dir, "dist", filepath.FromSlash(rel)))
	return err == nil
}

// A date-prefixed post's 1.5 URL redirects to its current one, as an HTML
// stub and a _redirects rule.
func TestLegacyDateAlias_RedirectsOldURL(t *testing.T) {
	dir := createFixtureSite(t)
	writeFixture(t, dir, "content/blog/2026-03-15-hello.md", "---\ntitle: Hello\n---\nBody.\n")
	buildLegacyAliasSite(t, dir, legacyAliasConfig())

	stub := readOutput(t, dir, "blog/03-15-hello/index.html")
	if !strings.Contains(stub, `url=/blog/hello/"`) {
		t.Errorf("stub = %q, want a redirect to /blog/hello/", stub)
	}
	if !strings.Contains(readOutput(t, dir, "blog/hello/index.html"), "Body.") {
		t.Error("the page is not at /blog/hello/")
	}
	if rules := readOutput(t, dir, "_redirects"); !strings.Contains(rules, "/blog/03-15-hello/  /blog/hello/  301!") {
		t.Errorf("_redirects = %q, want the 1.5 URL rule", rules)
	}
}

// A real page at the 1.5 URL keeps it; no stub overwrites the page.
func TestLegacyDateAlias_RealPageWins(t *testing.T) {
	dir := createFixtureSite(t)
	writeFixture(t, dir, "content/blog/2026-03-15-hello.md", "---\ntitle: Hello\n---\nBody.\n")
	writeFixture(t, dir, "content/blog/03-15-hello/index.md", "---\ntitle: Bundle\n---\nBundle body.\n")
	buildLegacyAliasSite(t, dir, legacyAliasConfig())

	if page := readOutput(t, dir, "blog/03-15-hello/index.html"); !strings.Contains(page, "Bundle body.") {
		t.Errorf("/blog/03-15-hello/ = %q, want the bundle page", page)
	}
	// With no redirects at all, the plugin writes no _redirects.
	if outputExists(dir, "_redirects") {
		t.Errorf("_redirects = %q, want no rule for a real page", readOutput(t, dir, "_redirects"))
	}
}

// An author alias or a global redirect at the 1.5 URL wins, and is listed
// once.
func TestLegacyDateAlias_AuthorRedirectsWin(t *testing.T) {
	dir := createFixtureSite(t)
	writeFixture(t, dir, "content/blog/2026-03-15-hello.md", "---\ntitle: Hello\naliases: [/blog/03-15-hello/]\n---\nBody.\n")
	writeFixture(t, dir, "content/blog/2026-04-02-news.md", "---\ntitle: News\n---\nBody.\n")
	cfg := legacyAliasConfig()
	cfg.Redirects = map[string]string{"/blog/04-02-news/": "/about/"}
	buildLegacyAliasSite(t, dir, cfg)

	rules := readOutput(t, dir, "_redirects")
	if n := strings.Count(rules, "/blog/03-15-hello/ "); n != 1 {
		t.Errorf("_redirects lists /blog/03-15-hello/ %d times, want 1:\n%s", n, rules)
	}
	if !strings.Contains(rules, "/blog/04-02-news/  /about/  301!") {
		t.Errorf("_redirects = %q, want the global redirect's own target", rules)
	}
}

// Two posts that shared a 1.5 URL give it to the first in build order.
func TestLegacyDateAlias_SharedOldURL(t *testing.T) {
	dir := createFixtureSite(t)
	writeFixture(t, dir, "content/blog/2025-01-10-recap.md", "---\ntitle: Recap 2025\n---\nBody.\n")
	writeFixture(t, dir, "content/blog/2026-01-10-recap-two.md", "---\ntitle: Recap two\n---\nBody.\n")
	writeFixture(t, dir, "content/blog/2024-01-10-recap.md", "---\ntitle: Recap 2024\n---\nBody.\n")
	b := buildLegacyAliasSite(t, dir, legacyAliasConfig())

	owners := 0
	for _, p := range b.lastAllPages {
		for _, a := range p.Aliases {
			if a == "/blog/01-10-recap/" {
				owners++
			}
		}
	}
	if owners != 1 {
		t.Errorf("/blog/01-10-recap/ is an alias of %d pages, want 1", owners)
	}
	if !outputExists(dir, "blog/01-10-recap-two/index.html") {
		t.Error("the other post's 1.5 URL has no redirect")
	}
}

// A translation redirects its own 1.5 URL; a fallback copy adds none.
func TestLegacyDateAlias_Multilingual(t *testing.T) {
	dir := createFixtureSite(t)
	writeFixture(t, dir, "content/blog/2026-03-15-hello.md", "---\ntitle: Hello\n---\nBody.\n")
	writeFixture(t, dir, "content/fr/blog/2026-03-15-hello.md", "---\ntitle: Bonjour\n---\nCorps.\n")
	writeFixture(t, dir, "content/blog/2026-04-02-news.md", "---\ntitle: News\n---\nBody.\n")
	cfg := legacyAliasConfig()
	cfg.I18n.DefaultLanguage = "en"
	cfg.I18n.Languages = map[string]config.LanguageConfig{
		"en": {Name: "English", Weight: 1},
		"fr": {Name: "French", Weight: 2},
	}
	buildLegacyAliasSite(t, dir, cfg)

	rules := readOutput(t, dir, "_redirects")
	for _, want := range []string{
		"/blog/03-15-hello/  /blog/hello/  301!",
		"/fr/blog/03-15-hello/  /fr/blog/hello/  301!",
		"/blog/04-02-news/  /blog/news/  301!",
	} {
		if !strings.Contains(rules, want) {
			t.Errorf("_redirects missing %q:\n%s", want, rules)
		}
	}
	if strings.Contains(rules, "/fr/blog/04-02-news/") {
		t.Errorf("_redirects = %q, want no alias from the untranslated page's fallback", rules)
	}
}

// A collection with a permalink pattern gets no 1.5 redirect: its :slug
// did not change.
func TestLegacyDateAlias_PatternCollection(t *testing.T) {
	dir := createFixtureSite(t)
	writeFixture(t, dir, "content/blog/2026-03-15-hello.md", "---\ntitle: Hello\n---\nBody.\n")
	cfg := legacyAliasConfig()
	cfg.Collections = map[string]*config.CollectionSiteConfig{
		"blog": {Permalink: "/blog/:year/:slug/"},
	}
	buildLegacyAliasSite(t, dir, cfg)

	if !outputExists(dir, "blog/2026/hello/index.html") {
		t.Error("the page is not at /blog/2026/hello/")
	}
	if outputExists(dir, "blog/03-15-hello/index.html") {
		t.Error("a pattern collection got a 1.5 redirect")
	}
}

// With a base path, the stub's target carries it; the alias path is written
// like an author alias.
func TestLegacyDateAlias_BasePath(t *testing.T) {
	dir := createFixtureSite(t)
	writeFixture(t, dir, "content/blog/2026-03-15-hello.md", "---\ntitle: Hello\n---\nBody.\n")
	cfg := legacyAliasConfig()
	cfg.Build.BasePath = "/site/"
	buildLegacyAliasSite(t, dir, cfg)

	if stub := readOutput(t, dir, "blog/03-15-hello/index.html"); !strings.Contains(stub, `url=/site/blog/hello/"`) {
		t.Errorf("stub = %q, want a redirect to /site/blog/hello/", stub)
	}
}
