package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/engine"
)

// rebuildParity edits rel with body, runs an incremental rebuild and checks
// it stayed incremental, then runs a full build and checks that every output
// file in compare is byte-identical in both. It returns the files as the
// incremental rebuild wrote them.
func rebuildParity(t *testing.T, builder *SiteBuilder, dir string, fullPages int, rel, body string, compare ...string) map[string]string {
	t.Helper()
	writeFixture(t, dir, rel, body)
	result, err := builder.ContentRebuild([]string{filepath.Join(dir, filepath.FromSlash(rel))})
	if err != nil {
		t.Fatalf("ContentRebuild failed: %v", err)
	}
	if result.PageCount >= fullPages {
		t.Fatalf("rebuild rendered %d pages, the full build %d: want an incremental rebuild", result.PageCount, fullPages)
	}
	dist := filepath.Join(dir, "dist")
	read := func(out string) string {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(dist, filepath.FromSlash(out)))
		if err != nil {
			t.Fatalf("reading %s: %v", out, err)
		}
		return string(data)
	}
	got := make(map[string]string, len(compare))
	for _, out := range compare {
		got[out] = read(out)
	}
	if _, err := builder.Build(); err != nil {
		t.Fatalf("full Build failed: %v", err)
	}
	for _, out := range compare {
		if read(out) != got[out] {
			t.Errorf("%s differs between the incremental rebuild and a full build", out)
		}
	}
	return got
}

// fullBuild runs the first build and returns its page count.
func fullBuild(t *testing.T, builder *SiteBuilder) int {
	t.Helper()
	result, err := builder.Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	return result.PageCount
}

// A body edit on a site with a site-level permalinks pattern keeps writing
// the page at the pattern's URL.
func TestContentRebuild_BodyEditKeepsSitePermalinkPattern(t *testing.T) {
	dir := createIncrementalSite(t)
	cfg := config.Defaults()
	cfg.Permalinks = map[string]string{"blog": "/blog/:year/:month/:slug/"}
	builder := newIncrementalBuilder(dir, cfg)
	full := fullBuild(t, builder)
	dist := filepath.Join(dir, "dist")
	assertFixtureFileExists(t, dist, "blog/2025/01/one/index.html")

	got := rebuildParity(t, builder, dir, full, "content/blog/one.md",
		"---\ntitle: One\ndate: 2025-01-02T00:00:00Z\n---\n# One\nEdited body.\n",
		"blog/2025/01/one/index.html")
	if !strings.Contains(got["blog/2025/01/one/index.html"], "Edited body.") {
		t.Error("the edit did not reach the page at the pattern URL")
	}
	assertFixtureFileNotExists(t, dist, "blog/one/index.html")
}

// writeCascadeSite writes a single-language docs collection whose root
// _index.md cascades a banner and an author.
func writeCascadeSite(t *testing.T, dir string) {
	t.Helper()
	writeFixture(t, dir, "content/_index.md", "---\ntitle: Home\n---\n")
	writeFixture(t, dir, "content/docs/_index.md",
		"---\ntitle: Docs\ncascade:\n  banner:\n    content: Docs notice\n  params:\n    author: Docs Team\n---\n")
	writeFixture(t, dir, "content/docs/a.md", "---\ntitle: A\nsidebar:\n  order: 1\n---\nAlpha.\n")
	writeFixture(t, dir, "content/docs/b.md", "---\ntitle: B\nsidebar:\n  order: 2\n---\nBeta.\n")
}

// A re-parsed page gets the cascade back, after a body edit and after a
// frontmatter edit.
func TestContentRebuild_BodyEditKeepsCascade(t *testing.T) {
	dir := t.TempDir()
	writeCascadeSite(t, dir)
	builder := newIncrementalBuilder(dir, config.Defaults())
	full := fullBuild(t, builder)

	got := rebuildParity(t, builder, dir, full, "content/docs/a.md",
		"---\ntitle: A\nsidebar:\n  order: 1\n---\nAlpha edited.\n", "docs/a/index.html")
	if !strings.Contains(got["docs/a/index.html"], "Docs notice") {
		t.Error("body edit: the cascaded banner is gone")
	}

	got = rebuildParity(t, builder, dir, full, "content/docs/a.md",
		"---\ntitle: A\ndescription: Now described.\nsidebar:\n  order: 1\n---\nAlpha edited.\n", "docs/a/index.html")
	if !strings.Contains(got["docs/a/index.html"], "Docs notice") {
		t.Error("frontmatter edit: the cascaded banner is gone")
	}
}

// On a bilingual site the re-parsed page gets its own language's cascade,
// and the borrowed default-language one where its language has none.
func TestContentRebuild_BodyEditKeepsPerLanguageCascade(t *testing.T) {
	dir := t.TempDir()
	writeCascadeI18nSite(t, dir)
	builder := newIncrementalBuilder(dir, cascadeI18nConfig())
	full := fullBuild(t, builder)

	got := rebuildParity(t, builder, dir, full, "content/fr/docs/guide/intro.md",
		"---\ntitle: Intro\n---\nBonjour encore.\n", "fr/docs/guide/intro/index.html")
	if !strings.Contains(got["fr/docs/guide/intro/index.html"], "Avis du guide") {
		t.Error("the French cascade is gone after the rebuild")
	}
	got = rebuildParity(t, builder, dir, full, "content/fr/docs/api/endpoints.md",
		"---\ntitle: Points d'accès\n---\nListe revue.\n", "fr/docs/api/endpoints/index.html")
	if !strings.Contains(got["fr/docs/api/endpoints/index.html"], "API notice") {
		t.Error("the borrowed English cascade is gone after the rebuild")
	}
}

// A slide deck keeps the presentation layout after a body edit.
func TestContentRebuild_SlidesDeckKeepsPresentationLayout(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "content/_index.md", "---\ntitle: Home\n---\n")
	writeFixture(t, dir, "content/slides/deck.md", "---\ntitle: Deck\n---\n# One\n\n---\n\n# Two\n")
	writeFixture(t, dir, "content/slides/other.md", "---\ntitle: Other\n---\n# Other\n")
	builder := newIncrementalBuilder(dir, config.Defaults())
	full := fullBuild(t, builder)

	got := rebuildParity(t, builder, dir, full, "content/slides/deck.md",
		"---\ntitle: Deck\n---\n# One edited\n\n---\n\n# Two\n", "slides/deck/index.html")
	if !strings.Contains(got["slides/deck/index.html"], `class="sarde-layout-presentation"`) {
		t.Error("the deck lost the presentation layout")
	}
}

// writeLabsSite writes a multi-course labs collection: course web with labs
// build (two steps) and hello (one page), plus a course-level page.
func writeLabsSite(t *testing.T, dir string) {
	t.Helper()
	writeFixture(t, dir, "content/_index.md", "---\ntitle: Home\n---\n")
	writeFixture(t, dir, "content/labs/_index.md", "---\ntitle: Labs\n---\n")
	writeFixture(t, dir, "content/labs/web/_index.md", "---\ntitle: Web Fundamentals\n---\n")
	writeFixture(t, dir, "content/labs/web/syllabus.md", "---\ntitle: Syllabus\n---\nCourse plan.\n")
	writeFixture(t, dir, "content/labs/web/build/_index.md", "---\ntitle: Build a Webpage\n---\nIntro.\n")
	writeFixture(t, dir, "content/labs/web/build/scaffold.md", "---\ntitle: Scaffold\nsidebar:\n  order: 1\n---\nStep one.\n")
	writeFixture(t, dir, "content/labs/web/build/style.md", "---\ntitle: Style\nsidebar:\n  order: 2\n---\nStep two.\n")
	writeFixture(t, dir, "content/labs/web/hello/_index.md", "---\ntitle: Hello World\n---\nAll in one page.\n")
}

// A lab step keeps the labs layout and its progress bar after a body edit,
// and a course-level page does not gain the labs layout.
func TestContentRebuild_LabsStepKeepsProgress(t *testing.T) {
	dir := t.TempDir()
	writeLabsSite(t, dir)
	builder := newIncrementalBuilder(dir, config.Defaults())
	full := fullBuild(t, builder)

	step := "labs/web/build/scaffold/index.html"
	got := rebuildParity(t, builder, dir, full, "content/labs/web/build/scaffold.md",
		"---\ntitle: Scaffold\nsidebar:\n  order: 1\n---\nStep one, edited.\n", step)
	if !strings.Contains(got[step], `class="sarde-layout-labs"`) || !strings.Contains(got[step], `max="3"`) {
		t.Error("the lab step lost its labs layout or progress bar")
	}

	rebuildParity(t, builder, dir, full, "content/labs/web/syllabus.md",
		"---\ntitle: Syllabus\n---\nCourse plan, edited.\n", "labs/web/syllabus/index.html")
}

// Reordering the steps of a lab rewires prev/next within that lab only.
func TestContentRebuild_LabsOrderChangeWiresWithinLab(t *testing.T) {
	dir := t.TempDir()
	writeLabsSite(t, dir)
	builder := newIncrementalBuilder(dir, config.Defaults())
	full := fullBuild(t, builder)

	rebuildParity(t, builder, dir, full, "content/labs/web/build/style.md",
		"---\ntitle: Style\nsidebar:\n  order: 0\n---\nStep two.\n",
		"labs/web/build/scaffold/index.html", "labs/web/build/style/index.html", "labs/web/hello/index.html")
}

// docsFallbackPages counts the i18n fallback copies of docs pages the last
// build kept.
func docsFallbackPages(pages []*engine.Page) int {
	n := 0
	for _, p := range pages {
		if p.IsFallback && p.Collection != nil && p.Collection.Name == "docs" {
			n++
		}
	}
	return n
}

// A versioned collection's versioning.fallback: omit holds on rebuilds: a
// save does not add the French fallback copies a full build leaves out.
func TestContentRebuild_HonorsVersioningFallback(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "content/_index.md", "---\ntitle: Home\n---\n")
	writeFixture(t, dir, "content/fr/_index.md", "---\ntitle: Accueil\n---\n")
	writeFixture(t, dir, "content/docs/_index.md", "---\ntitle: Docs\n---\n")
	writeFixture(t, dir, "content/docs/guide.md", "---\ntitle: Guide\n---\nLatest.\n")
	writeFixture(t, dir, "content/docs/v1/_index.md", "---\ntitle: Docs v1\n---\n")
	writeFixture(t, dir, "content/docs/v1/guide.md", "---\ntitle: Guide v1\n---\nOld.\n")
	cfg := config.Defaults()
	cfg.I18n.DefaultLanguage = "en"
	cfg.I18n.Fallback = "default"
	cfg.I18n.Languages = map[string]config.LanguageConfig{"en": {Weight: 1}, "fr": {Weight: 2}}
	cfg.Collections = map[string]*config.CollectionSiteConfig{
		"docs": {Versioning: &config.VersioningConfig{
			Enabled: config.BoolPtr(true), LastVersion: "v2", Fallback: "omit",
			Versions: []config.VersionEntry{{ID: "v2"}, {ID: "v1"}},
		}},
	}
	builder := newIncrementalBuilder(dir, cfg)
	full := fullBuild(t, builder)
	if n := docsFallbackPages(builder.lastAllPages); n != 0 {
		t.Fatalf("full build kept %d docs fallback copies, want 0", n)
	}
	dist := filepath.Join(dir, "dist")
	assertFixtureFileNotExists(t, dist, "fr/docs/guide/index.html")

	rebuildParity(t, builder, dir, full, "content/docs/guide.md",
		"---\ntitle: Guide\n---\nLatest, edited.\n", "docs/guide/index.html")
	if n := docsFallbackPages(builder.lastAllPages); n != 0 {
		t.Errorf("after the rebuild %d docs fallback copies, want 0", n)
	}
	assertFixtureFileNotExists(t, dist, "fr/docs/guide/index.html")
}

// The fallback copy of an edited docs page keeps its prev/next links.
func TestContentRebuild_FallbackCopyKeepsPrevNext(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "content/_index.md", "---\ntitle: Home\n---\n")
	writeFixture(t, dir, "content/fr/_index.md", "---\ntitle: Accueil\n---\n")
	writeFixture(t, dir, "content/docs/_index.md", "---\ntitle: Docs\n---\n")
	writeFixture(t, dir, "content/fr/docs/_index.md", "---\ntitle: Docs\n---\n")
	writeFixture(t, dir, "content/docs/a.md", "---\ntitle: Alpha\nsidebar:\n  order: 1\n---\nA.\n")
	writeFixture(t, dir, "content/docs/b.md", "---\ntitle: Bravo\nsidebar:\n  order: 2\n---\nB.\n")
	writeFixture(t, dir, "content/docs/c.md", "---\ntitle: Charlie\nsidebar:\n  order: 3\n---\nC.\n")
	cfg := config.Defaults()
	cfg.I18n.DefaultLanguage = "en"
	cfg.I18n.Fallback = "default"
	cfg.I18n.Languages = map[string]config.LanguageConfig{"en": {Weight: 1}, "fr": {Weight: 2}}
	builder := newIncrementalBuilder(dir, cfg)
	full := fullBuild(t, builder)

	got := rebuildParity(t, builder, dir, full, "content/docs/b.md",
		"---\ntitle: Bravo\nsidebar:\n  order: 2\n---\nB, edited.\n", "fr/docs/b/index.html")
	if !strings.Contains(got["fr/docs/b/index.html"], `href="/fr/docs/c/"`) {
		t.Error("the French copy of Bravo lost its next link to Charlie")
	}
}

// writeTaggedBilingualBlog writes an en/fr blog whose posts share a tag.
func writeTaggedBilingualBlog(t *testing.T, dir string) {
	t.Helper()
	writeFixture(t, dir, "content/_index.md", "---\ntitle: Home\n---\n")
	writeFixture(t, dir, "content/fr/_index.md", "---\ntitle: Accueil\n---\n")
	writeFixture(t, dir, "content/blog/_index.md", "---\ntitle: Blog\n---\n")
	writeFixture(t, dir, "content/fr/blog/_index.md", "---\ntitle: Carnet\n---\n")
	writeFixture(t, dir, "content/blog/one.md", "---\ntitle: One\ndate: 2025-01-02T00:00:00Z\ntags: [golang]\n---\nOne.\n")
	writeFixture(t, dir, "content/blog/two.md", "---\ntitle: Two\ndate: 2025-01-01T00:00:00Z\ntags: [golang]\n---\nTwo.\n")
	writeFixture(t, dir, "content/fr/blog/un.md", "---\ntitle: Un\ndate: 2025-01-03T00:00:00Z\ntags: [golang]\n---\nUn.\n")
}

// Taxonomy and term pages re-rendered by a rebuild keep the language
// switcher and hreflang links a full build gives them.
func TestContentRebuild_TermPageKeepsLanguageSwitcher(t *testing.T) {
	dir := t.TempDir()
	writeTaggedBilingualBlog(t, dir)
	cfg := cascadeI18nConfig()
	cfg.Site.URL = "https://example.com"
	cfg.Taxonomies = map[string]config.TaxonomyConfig{"tags": {}}
	builder := newIncrementalBuilder(dir, cfg)
	full := fullBuild(t, builder)

	got := rebuildParity(t, builder, dir, full, "content/blog/two.md",
		"---\ntitle: Two\ndate: 2025-01-01T00:00:00Z\ntags: [golang, testing]\n---\nTwo.\n",
		"tags/golang/index.html", "fr/tags/golang/index.html", "tags/index.html")
	for _, out := range []string{"tags/golang/index.html", "fr/tags/golang/index.html"} {
		if !strings.Contains(got[out], `class="sarde-lang-switcher"`) {
			t.Errorf("%s has no language switcher after the rebuild", out)
		}
	}
	if !strings.Contains(got["tags/golang/index.html"], `hreflang="fr"`) {
		t.Error("tags/golang/ has no hreflang link to French after the rebuild")
	}
}

// A term page re-rendered by a rebuild on a single-language site gets the
// site's stylesheets and the plugin output of a full build.
func TestContentRebuild_TermPageKeepsCustomCSS(t *testing.T) {
	dir := createIncrementalSite(t)
	writeFixture(t, dir, "content/blog/one.md", "---\ntitle: One\ndate: 2025-01-02T00:00:00Z\ntags: [golang]\n---\nOne.\n")
	writeFixture(t, dir, "assets/css/custom.css", ".brand { color: rebeccapurple; }\n")
	cfg := config.Defaults()
	cfg.Taxonomies = map[string]config.TaxonomyConfig{"tags": {}}
	builder := newIncrementalBuilder(dir, cfg)
	full := fullBuild(t, builder)

	rebuildParity(t, builder, dir, full, "content/blog/two.md",
		"---\ntitle: Two\ndate: 2025-01-01T00:00:00Z\ntags: [golang]\n---\nTwo.\n",
		"tags/golang/index.html", "tags/index.html")
}
