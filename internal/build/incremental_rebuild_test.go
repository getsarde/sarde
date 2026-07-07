package build

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/getsarde/sarde/embedded"
	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/engine"
)

func newIncrementalBuilder(projectDir string, cfg *config.SiteConfig) *SiteBuilder {
	return NewSiteBuilder(BuildOptions{
		ProjectDir:  projectDir,
		Config:      cfg,
		ThemeConfig: buildThemeConfig(),
		EmbeddedFS:  embedded.ThemeFS(),
		DevMode:     true,
	})
}

func createIncrementalSite(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeFixture(t, dir, "content/_index.md", "---\ntitle: Home\n---\n# Home\n")
	writeFixture(t, dir, "content/about.md", "---\ntitle: About\n---\nAbout page.\n")
	writeFixture(t, dir, "content/blog/_index.md", "---\ntitle: Blog\n---\n")
	writeFixture(t, dir, "content/blog/one.md", "---\ntitle: One\ndate: 2025-01-02T00:00:00Z\n---\n# One\nOriginal body with [about](/about/).\n")
	writeFixture(t, dir, "content/blog/two.md", "---\ntitle: Two\ndate: 2025-01-01T00:00:00Z\n---\n# Two\nSecond body.\n")
	return dir
}

func TestContentRebuild_BodyOnlyUpdatesPageSearchAndValidation(t *testing.T) {
	dir := createIncrementalSite(t)
	cfg := config.Defaults()
	cfg.Plugins.Enabled = []string{"search", "link_validator"}
	cfg.LinkValidation.Enabled = config.BoolPtr(true)

	builder := newIncrementalBuilder(dir, cfg)
	if _, err := builder.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	postPath := filepath.Join(dir, "content", "blog", "one.md")
	writeFixture(t, dir, "content/blog/one.md", "---\ntitle: One\ndate: 2025-01-02T00:00:00Z\n---\n# One\nUpdated body with [missing](/missing/).\n")

	result, err := builder.ContentRebuild([]string{postPath})
	if err != nil {
		t.Fatalf("ContentRebuild failed: %v", err)
	}
	if result.PageCount >= 5 {
		t.Fatalf("expected incremental rebuild, got likely full rebuild page count %d", result.PageCount)
	}

	distDir := filepath.Join(dir, "dist")
	assertFixtureFileContains(t, distDir, "blog/one/index.html", "Updated body")
	assertFixtureFileContains(t, distDir, "search-index.en.json", "Updated body")

	entry, ok := builder.lastValidationData["/blog/one/"]
	if !ok {
		t.Fatal("expected validation data for changed page")
	}
	if len(entry.Links) != 1 || entry.Links[0].Href != "/missing/" {
		t.Fatalf("validation links = %+v, want only /missing/", entry.Links)
	}
	if len(result.Warnings) == 0 {
		t.Fatal("expected link-validator warning for /missing/")
	}
	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w.Message, "/missing/") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("warnings did not mention /missing/: %+v", result.Warnings)
	}
}

func TestContentRebuild_BodyOnlyRendersCollectionPagination(t *testing.T) {
	dir := createIncrementalSite(t)
	cfg := config.Defaults()
	cfg.Collections = map[string]*config.CollectionSiteConfig{
		"blog": {Paginate: 1},
	}

	builder := newIncrementalBuilder(dir, cfg)
	if _, err := builder.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	postPath := filepath.Join(dir, "content", "blog", "one.md")
	writeFixture(t, dir, "content/blog/one.md", "---\ntitle: One\ndate: 2025-01-02T00:00:00Z\n---\n# One\nPagination-safe body edit.\n")

	result, err := builder.ContentRebuild([]string{postPath})
	if err != nil {
		t.Fatalf("ContentRebuild failed: %v", err)
	}
	if result.PageCount < 3 {
		t.Fatalf("expected changed page plus collection list pagination, got %d rendered pages", result.PageCount)
	}
	assertFixtureFileExists(t, filepath.Join(dir, "dist"), "blog/page/2/index.html")
}

func TestContentRebuild_StructuralChangesFallBackToFullBuild(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		write      string
		removeFile bool
	}{
		{
			name:  "new file",
			path:  "content/blog/new.md",
			write: "---\ntitle: New\ndate: 2025-01-03T00:00:00Z\n---\nNew post.\n",
		},
		{
			name:       "deleted file",
			path:       "content/blog/one.md",
			removeFile: true,
		},
		{
			name:  "section index",
			path:  "content/blog/_index.md",
			write: "---\ntitle: Blog Updated\n---\n",
		},
		{
			name:  "slug",
			path:  "content/blog/one.md",
			write: "---\ntitle: One\nslug: renamed\ndate: 2025-01-02T00:00:00Z\n---\n# One\nBody.\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := createIncrementalSite(t)
			cfg := config.Defaults()
			builder := newIncrementalBuilder(dir, cfg)
			if _, err := builder.Build(); err != nil {
				t.Fatalf("Build failed: %v", err)
			}

			absPath := filepath.Join(dir, filepath.FromSlash(tt.path))
			if tt.removeFile {
				if err := os.Remove(absPath); err != nil {
					t.Fatalf("removing fixture: %v", err)
				}
			} else {
				writeFixture(t, dir, tt.path, tt.write)
			}

			result, err := builder.ContentRebuild([]string{absPath})
			if err != nil {
				t.Fatalf("ContentRebuild failed: %v", err)
			}
			if result.PageCount <= 2 {
				t.Fatalf("expected full rebuild fallback, got %d rendered pages", result.PageCount)
			}
		})
	}
}

func TestContentRebuild_DeletedFileOutputRemovedFromDist(t *testing.T) {
	dir := createIncrementalSite(t)
	cfg := config.Defaults()
	builder := newIncrementalBuilder(dir, cfg)
	if _, err := builder.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	distDir := filepath.Join(dir, "dist")
	assertFixtureFileExists(t, distDir, "blog/one/index.html")

	postPath := filepath.Join(dir, "content", "blog", "one.md")
	if err := os.Remove(postPath); err != nil {
		t.Fatalf("removing fixture: %v", err)
	}

	result, err := builder.ContentRebuild([]string{postPath})
	if err != nil {
		t.Fatalf("ContentRebuild failed: %v", err)
	}
	if result.PageCount <= 2 {
		t.Fatalf("expected full rebuild fallback, got %d rendered pages", result.PageCount)
	}

	assertFixtureFileNotExists(t, distDir, "blog/one/index.html")
	assertFixtureFileExists(t, distDir, "blog/two/index.html")
}

func TestContentRebuild_NonStructuralChangesStayIncremental(t *testing.T) {
	// maxPages is the highest page count that still indicates an incremental
	// rebuild: the changed page, the blog index, and the home page (always
	// re-rendered for recentEntries), plus taxonomy stubs where terms change.
	// A full rebuild renders at least the 5 content pages.
	tests := []struct {
		name     string
		path     string
		write    string
		maxPages int
	}{
		{
			name:     "alias change",
			path:     "content/blog/one.md",
			write:    "---\ntitle: One\ndate: 2025-01-02T00:00:00Z\naliases: [/old-one/]\n---\n# One\nBody.\n",
			maxPages: 4,
		},
		{
			name:     "taxonomy change",
			path:     "content/blog/one.md",
			write:    "---\ntitle: One\ndate: 2025-01-02T00:00:00Z\ntags: [go]\n---\n# One\nBody.\n",
			maxPages: 5,
		},
		{
			name:     "featured change",
			path:     "content/blog/one.md",
			write:    "---\ntitle: One\ndate: 2025-01-02T00:00:00Z\nfeatured: true\n---\n# One\nBody.\n",
			maxPages: 4,
		},
		{
			name:     "order change (non-sort field for blog)",
			path:     "content/blog/one.md",
			write:    "---\ntitle: One\ndate: 2025-01-02T00:00:00Z\nsidebar:\n  order: 10\n---\n# One\nBody.\n",
			maxPages: 4,
		},
		{
			name:     "render flag",
			path:     "content/blog/one.md",
			write:    "---\ntitle: One\ndate: 2025-01-02T00:00:00Z\nrender: false\n---\n# One\nBody.\n",
			maxPages: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := createIncrementalSite(t)
			cfg := config.Defaults()
			builder := newIncrementalBuilder(dir, cfg)
			if _, err := builder.Build(); err != nil {
				t.Fatalf("Build failed: %v", err)
			}

			absPath := filepath.Join(dir, filepath.FromSlash(tt.path))
			writeFixture(t, dir, tt.path, tt.write)

			result, err := builder.ContentRebuild([]string{absPath})
			if err != nil {
				t.Fatalf("ContentRebuild failed: %v", err)
			}
			if result.PageCount > tt.maxPages {
				t.Fatalf("expected incremental rebuild (<= %d pages), got likely full rebuild page count %d", tt.maxPages, result.PageCount)
			}
		})
	}
}

func TestContentRebuild_BodyEditReRendersHomePage(t *testing.T) {
	// Home layouts render recentEntries (title, date, summary) from the site
	// context, so every incremental rebuild must re-render the home page.
	dir := createIncrementalSite(t)
	builder := newIncrementalBuilder(dir, config.Defaults())
	if _, err := builder.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	distDir := filepath.Join(dir, "dist")
	if err := os.Remove(filepath.Join(distDir, "index.html")); err != nil {
		t.Fatalf("removing home output: %v", err)
	}

	postPath := filepath.Join(dir, "content", "blog", "one.md")
	writeFixture(t, dir, "content/blog/one.md", "---\ntitle: One\ndate: 2025-01-02T00:00:00Z\n---\n# One\nRefreshed body.\n")

	result, err := builder.ContentRebuild([]string{postPath})
	if err != nil {
		t.Fatalf("ContentRebuild failed: %v", err)
	}
	if result.PageCount >= 5 {
		t.Fatalf("expected incremental rebuild, got likely full rebuild page count %d", result.PageCount)
	}
	assertFixtureFileExists(t, distDir, "index.html")
}

func TestContentRebuild_H1TitleEditRebuildsCollection(t *testing.T) {
	// A page without a frontmatter title infers its title from the first H1,
	// which the frontmatter digest never sees. An H1-only edit must still be
	// classified as a title change (collection-scoped) so sibling sidebars
	// pick up the new title instead of taking the body-only fast path.
	dir := t.TempDir()
	writeFixture(t, dir, "content/_index.md", "---\ntitle: Home\n---\n# Home\n")
	writeFixture(t, dir, "content/docs/_index.md", "---\ntitle: Docs\n---\n")
	writeFixture(t, dir, "content/docs/alpha.md", "---\ndescription: Alpha page\n---\n# Alpha Original\n\nAlpha body.\n")
	writeFixture(t, dir, "content/docs/beta.md", "---\ntitle: Beta\n---\n# Beta\n\nBeta body.\n")

	builder := newIncrementalBuilder(dir, config.Defaults())
	if _, err := builder.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	distDir := filepath.Join(dir, "dist")
	assertFixtureFileContains(t, distDir, "docs/beta/index.html", "Alpha Original")

	pagePath := filepath.Join(dir, "content", "docs", "alpha.md")
	writeFixture(t, dir, "content/docs/alpha.md", "---\ndescription: Alpha page\n---\n# Alpha Renamed\n\nAlpha body.\n")

	if _, err := builder.ContentRebuild([]string{pagePath}); err != nil {
		t.Fatalf("ContentRebuild failed: %v", err)
	}
	assertFixtureFileContains(t, distDir, "docs/alpha/index.html", "Alpha Renamed")
	// The sibling's sidebar proves the collection nav was actually rebuilt.
	assertFixtureFileContains(t, distDir, "docs/beta/index.html", "Alpha Renamed")
	assertFixtureFileNotContains(t, distDir, "docs/beta/index.html", "Alpha Original")
}

func TestContentRebuild_BodyEditWithTaxonomyReusesTermPageContent(t *testing.T) {
	// The body-only fast path reuses the last build's taxonomy structures
	// instead of rebuilding them. Term pages render member-page summaries, so
	// the reused term lists must be re-pointed at the re-parsed page objects
	// or the term page would show the previous save's content.
	dir := createIncrementalSite(t)
	writeFixture(t, dir, "content/blog/one.md", "---\ntitle: One\ndate: 2025-01-02T00:00:00Z\ntags: [go]\n---\n# One\nOriginal distinctive prose for the term page.\n")

	builder := newIncrementalBuilder(dir, config.Defaults())
	if _, err := builder.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	distDir := filepath.Join(dir, "dist")
	assertFixtureFileContains(t, distDir, "tags/go/index.html", "Original distinctive prose")

	postPath := filepath.Join(dir, "content", "blog", "one.md")
	writeFixture(t, dir, "content/blog/one.md", "---\ntitle: One\ndate: 2025-01-02T00:00:00Z\ntags: [go]\n---\n# One\nRefreshed distinctive prose for the term page.\n")

	result, err := builder.ContentRebuild([]string{postPath})
	if err != nil {
		t.Fatalf("ContentRebuild failed: %v", err)
	}
	if result.PageCount >= 6 {
		t.Fatalf("expected incremental rebuild, got likely full rebuild page count %d", result.PageCount)
	}
	assertFixtureFileContains(t, distDir, "tags/go/index.html", "Refreshed distinctive prose")
	assertFixtureFileNotContains(t, distDir, "tags/go/index.html", "Original distinctive prose")
}

func TestContentRebuild_TitleChangeRebuildsCollection(t *testing.T) {
	dir := createIncrementalSite(t)
	cfg := config.Defaults()
	builder := newIncrementalBuilder(dir, cfg)
	if _, err := builder.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	postPath := filepath.Join(dir, "content", "blog", "one.md")
	writeFixture(t, dir, "content/blog/one.md", "---\ntitle: One Updated\ndate: 2025-01-02T00:00:00Z\n---\n# One Updated\nBody.\n")

	result, err := builder.ContentRebuild([]string{postPath})
	if err != nil {
		t.Fatalf("ContentRebuild failed: %v", err)
	}
	// Title change triggers collection-scoped rebuild: all blog pages re-rendered.
	if result.PageCount < 2 {
		t.Fatalf("expected collection-scoped rebuild, got %d rendered pages", result.PageCount)
	}
}

func TestContentRebuild_ContentDigestSkipsUnchangedFile(t *testing.T) {
	dir := createIncrementalSite(t)
	cfg := config.Defaults()
	builder := newIncrementalBuilder(dir, cfg)
	if _, err := builder.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// "Touch" the file without changing content — ContentRebuild should skip it.
	postPath := filepath.Join(dir, "content", "blog", "one.md")
	result, err := builder.ContentRebuild([]string{postPath})
	if err != nil {
		t.Fatalf("ContentRebuild failed: %v", err)
	}
	if result.PageCount != 0 {
		t.Fatalf("expected 0 pages rendered for unchanged file, got %d", result.PageCount)
	}
}

func TestContentRebuild_AliasChangeCreatesNewAlias(t *testing.T) {
	dir := createIncrementalSite(t)
	writeFixture(t, dir, "content/blog/one.md", "---\ntitle: One\ndate: 2025-01-02T00:00:00Z\naliases: [/old-one/]\n---\n# One\nBody.\n")

	cfg := config.Defaults()
	builder := newIncrementalBuilder(dir, cfg)
	if _, err := builder.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	distDir := filepath.Join(dir, "dist")
	assertFixtureFileExists(t, distDir, "old-one/index.html")

	postPath := filepath.Join(dir, "content", "blog", "one.md")
	writeFixture(t, dir, "content/blog/one.md", "---\ntitle: One\ndate: 2025-01-02T00:00:00Z\naliases: [/new-one/]\n---\n# One\nBody.\n")
	result, err := builder.ContentRebuild([]string{postPath})
	if err != nil {
		t.Fatalf("ContentRebuild failed: %v", err)
	}
	if result.PageCount >= 5 {
		t.Fatalf("expected incremental rebuild for alias change, got likely full rebuild page count %d", result.PageCount)
	}
	assertFixtureFileExists(t, distDir, "new-one/index.html")
}

func TestContentRebuild_RemovedCustomTaxonomyTermUpdatesTermPage(t *testing.T) {
	// Removed-term dirty marking must cover custom taxonomies (authors,
	// series, ...), not just the built-in tags/categories fields.
	dir := createIncrementalSite(t)
	writeFixture(t, dir, "content/blog/one.md", "---\ntitle: One\ndate: 2025-01-02T00:00:00Z\nauthors: [jane, bob]\n---\n# One\nBody.\n")
	writeFixture(t, dir, "content/blog/two.md", "---\ntitle: Two\ndate: 2025-01-01T00:00:00Z\nauthors: [bob]\n---\n# Two\nSecond body.\n")

	cfg := config.Defaults()
	cfg.Taxonomies = map[string]config.TaxonomyConfig{
		"authors": {Singular: "author"},
	}
	builder := newIncrementalBuilder(dir, cfg)
	if _, err := builder.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	distDir := filepath.Join(dir, "dist")
	assertFixtureFileContains(t, distDir, "authors/bob/index.html", "/blog/one/")

	// Drop bob from post one; bob still has post two, so his term page must
	// be re-rendered without post one.
	postPath := filepath.Join(dir, "content", "blog", "one.md")
	writeFixture(t, dir, "content/blog/one.md", "---\ntitle: One\ndate: 2025-01-02T00:00:00Z\nauthors: [jane]\n---\n# One\nBody.\n")

	result, err := builder.ContentRebuild([]string{postPath})
	if err != nil {
		t.Fatalf("ContentRebuild failed: %v", err)
	}
	if result.PageCount >= 7 {
		t.Fatalf("expected incremental rebuild, got likely full rebuild page count %d", result.PageCount)
	}
	assertFixtureFileNotContains(t, distDir, "authors/bob/index.html", "/blog/one/")
	assertFixtureFileContains(t, distDir, "authors/bob/index.html", "/blog/two/")
	assertFixtureFileContains(t, distDir, "authors/jane/index.html", "/blog/one/")
}

func TestContentRebuild_LastTermRemovedDeletesOrphanedTermPage(t *testing.T) {
	dir := createIncrementalSite(t)
	writeFixture(t, dir, "content/blog/one.md", "---\ntitle: One\ndate: 2025-01-02T00:00:00Z\ntags: [golang]\n---\n# One\nBody.\n")
	writeFixture(t, dir, "content/blog/two.md", "---\ntitle: Two\ndate: 2025-01-01T00:00:00Z\ntags: [rust]\n---\n# Two\nBody.\n")

	cfg := config.Defaults()
	builder := newIncrementalBuilder(dir, cfg)
	if _, err := builder.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	distDir := filepath.Join(dir, "dist")
	assertFixtureFileExists(t, distDir, "tags/golang/index.html")
	assertFixtureFileExists(t, distDir, "tags/rust/index.html")
	assertFixtureFileExists(t, distDir, "tags/index.html")

	postPath := filepath.Join(dir, "content", "blog", "one.md")
	writeFixture(t, dir, "content/blog/one.md", "---\ntitle: One\ndate: 2025-01-02T00:00:00Z\ntags: []\n---\n# One\nBody.\n")

	result, err := builder.ContentRebuild([]string{postPath})
	if err != nil {
		t.Fatalf("ContentRebuild failed: %v", err)
	}
	if result.PageCount >= 7 {
		t.Fatalf("expected incremental rebuild, got likely full rebuild page count %d", result.PageCount)
	}

	assertFixtureFileNotExists(t, distDir, "tags/golang/index.html")
	assertFixtureFileExists(t, distDir, "tags/rust/index.html")
	assertFixtureFileExists(t, distDir, "tags/index.html")
}

func TestContentRebuild_AllTermsRemovedDeletesTaxonomyIndex(t *testing.T) {
	dir := createIncrementalSite(t)
	writeFixture(t, dir, "content/blog/one.md", "---\ntitle: One\ndate: 2025-01-02T00:00:00Z\ntags: [golang]\n---\n# One\nBody.\n")

	cfg := config.Defaults()
	builder := newIncrementalBuilder(dir, cfg)
	if _, err := builder.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	distDir := filepath.Join(dir, "dist")
	assertFixtureFileExists(t, distDir, "tags/golang/index.html")
	assertFixtureFileExists(t, distDir, "tags/index.html")

	postPath := filepath.Join(dir, "content", "blog", "one.md")
	writeFixture(t, dir, "content/blog/one.md", "---\ntitle: One\ndate: 2025-01-02T00:00:00Z\n---\n# One\nBody.\n")

	result, err := builder.ContentRebuild([]string{postPath})
	if err != nil {
		t.Fatalf("ContentRebuild failed: %v", err)
	}
	if result.PageCount >= 7 {
		t.Fatalf("expected incremental rebuild, got likely full rebuild page count %d", result.PageCount)
	}

	assertFixtureFileNotExists(t, distDir, "tags/golang/index.html")
	assertFixtureFileNotExists(t, distDir, "tags/index.html")
}

func TestRebuildFallback_HardErrorResetsBuilt(t *testing.T) {
	// A hard (non-fallback) error aborts mid-rebuild after in-place mutations
	// (patched collections, site context, template engine); the builder must
	// not offer that state to the next incremental rebuild.
	builder := newIncrementalBuilder(t.TempDir(), config.Defaults())
	builder.built = true

	if _, err := builder.rebuildFallback(errors.New("disk full")); err == nil {
		t.Fatal("expected the hard error to be returned")
	}
	if builder.built {
		t.Error("built must be cleared on hard incremental errors")
	}
}

// rebuildCollectionNav cannot correctly rebuild tabbed, versioned, or
// multi-language sidebars, so a collection-scoped change in those cases must
// escalate to a full rebuild.
func TestCollectionNeedsFullNavRebuild(t *testing.T) {
	base := func() *SiteBuilder {
		return newIncrementalBuilder(t.TempDir(), config.Defaults())
	}

	// Plain single-language, non-tabbed, non-versioned: incremental is fine.
	if b := base(); b.collectionNeedsFullNavRebuild(&engine.Collection{Name: "blog"}) {
		t.Error("plain collection should not need a full nav rebuild")
	}

	// Tabbed collection.
	if b := base(); !b.collectionNeedsFullNavRebuild(&engine.Collection{Name: "docs", IsTabbed: true}) {
		t.Error("tabbed collection must escalate to a full rebuild")
	}

	// Versioned collection.
	if b := base(); !b.collectionNeedsFullNavRebuild(&engine.Collection{
		Name:       "docs",
		Versioning: &engine.VersionConfig{Enabled: true},
	}) {
		t.Error("versioned collection must escalate to a full rebuild")
	}

	// Multi-language site.
	cfg := config.Defaults()
	cfg.I18n.Languages = map[string]config.LanguageConfig{"en": {}, "fr": {}}
	b := newIncrementalBuilder(t.TempDir(), cfg)
	if !b.collectionNeedsFullNavRebuild(&engine.Collection{Name: "blog"}) {
		t.Error("multi-language site must escalate any collection-scoped nav change")
	}
}
