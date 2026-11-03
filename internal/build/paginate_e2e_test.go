package build

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/getsarde/sarde/internal/config"
)

// writePaginateSite writes a blog of 12 posts, more than the inferred 10 per
// page, whose index uses the given list template.
func writePaginateSite(t *testing.T, tmpl string) string {
	t.Helper()
	dir := t.TempDir()
	writeFixture(t, dir, "content/_index.md", "---\ntitle: Home\n---\nWelcome.\n")
	index := "---\ntitle: Field Notes\n"
	if tmpl != "" {
		index += "template: " + tmpl + "\n"
	}
	writeFixture(t, dir, "content/blog/_index.md", index+"---\n")
	for i := 1; i <= 12; i++ {
		writeFixture(t, dir, fmt.Sprintf("content/blog/post-%02d.md", i),
			fmt.Sprintf("---\ntitle: Post %02d\ndate: 2025-01-%02dT00:00:00Z\n---\nNotes from day %d.\n", i, i, i))
	}
	return dir
}

// countListEntries counts the post entries a blog list template rendered.
func countListEntries(html, tmpl string) int {
	if tmpl == "blog/list-minimal" {
		_, list, _ := strings.Cut(html, `<ul class="sarde-post-list-minimal">`)
		list, _, _ = strings.Cut(list, "</ul>")
		return strings.Count(list, "<li>")
	}
	return strings.Count(html, `class="sarde-post-card-title"`)
}

// paginate: 0 turns pagination off: every post on one page, no numbered
// pages, no pagination nav, and no card for the blog's own _index.md.
func TestBuild_PaginateZeroListsEveryPost(t *testing.T) {
	for _, tc := range []struct{ name, tmpl, marker string }{
		{"blog/list", "", `<div class="sarde-post-grid">`},
		{"blog/list-grid", "blog/list-grid", "sarde-post-grid-cards"},
		{"blog/list-minimal", "blog/list-minimal", "sarde-post-list-minimal"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := writePaginateSite(t, tc.tmpl)
			cfg := config.Defaults()
			cfg.Collections = map[string]*config.CollectionSiteConfig{"blog": {Paginate: config.IntPtr(0)}}
			read := buildImageSite(t, dir, cfg)

			if _, err := os.Stat(filepath.Join(dir, "dist", "blog", "page", "2", "index.html")); err == nil {
				t.Error("blog/page/2/ must not be built with paginate: 0")
			}
			html := read("blog/index.html")
			if !strings.Contains(html, tc.marker) {
				t.Fatalf("expected the %s template (marker %q)", tc.name, tc.marker)
			}
			if got := countListEntries(html, tc.name); got != 12 {
				t.Errorf("list shows %d entries, want the 12 posts and no section page", got)
			}
			if strings.Contains(html, "sarde-pagination-info") {
				t.Error("unpaginated list must not render the pagination nav")
			}
		})
	}

	t.Run("unset paginate keeps 10 per page", func(t *testing.T) {
		dir := writePaginateSite(t, "")
		read := buildImageSite(t, dir, config.Defaults())
		if got := countListEntries(read("blog/index.html"), "blog/list"); got != 10 {
			t.Errorf("first page shows %d posts, want 10", got)
		}
		if _, err := os.Stat(filepath.Join(dir, "dist", "blog", "page", "2", "index.html")); err != nil {
			t.Errorf("blog/page/2/ must exist: %v", err)
		}
	})
}

// writeBilingualBlog writes 12 English posts and 3 French ones with their own
// filenames (so none of the French posts translates an English one), each
// language with its own blog index, paginated 5 per page.
func writeBilingualBlog(t *testing.T) (string, *config.SiteConfig) {
	t.Helper()
	dir := writePaginateSite(t, "")
	writeFixture(t, dir, "content/fr/_index.md", "---\ntitle: Accueil\n---\nBienvenue.\n")
	writeFixture(t, dir, "content/fr/blog/_index.md", "---\ntitle: Carnet\n---\n")
	for i := 1; i <= 3; i++ {
		writeFixture(t, dir, fmt.Sprintf("content/fr/blog/billet-%02d.md", i),
			fmt.Sprintf("---\ntitle: Billet %02d\ndate: 2025-02-%02dT00:00:00Z\n---\nNotes du jour %d.\n", i, i, i))
	}
	cfg := config.Defaults()
	cfg.I18n.DefaultLanguage = "en"
	cfg.I18n.Languages = map[string]config.LanguageConfig{
		"en": {Name: "English", Weight: 1, Dir: "ltr"},
		"fr": {Name: "Français", Weight: 2, Dir: "ltr"},
	}
	cfg.Collections = map[string]*config.CollectionSiteConfig{"blog": {Paginate: config.IntPtr(5)}}
	return dir, cfg
}

// Each language's blog lists and pages through its own posts: the English
// list never shows a French post, the French list holds the French posts plus
// the fallback copies of the untranslated English ones (or only the French
// posts under fallback "omit"), and each language gets its own numbered pages.
func TestBuild_PaginatePerLanguage(t *testing.T) {
	exists := func(dir, rel string) bool {
		_, err := os.Stat(filepath.Join(dir, "dist", filepath.FromSlash(rel)))
		return err == nil
	}

	t.Run("fallback default", func(t *testing.T) {
		dir, cfg := writeBilingualBlog(t)
		read := buildImageSite(t, dir, cfg)

		// English: 12 posts, 5 + 5 + 2.
		for page, want := range map[string]int{"blog/index.html": 5, "blog/page/2/index.html": 5, "blog/page/3/index.html": 2} {
			html := read(page)
			if got := countListEntries(html, "blog/list"); got != want {
				t.Errorf("%s: %d entries, want %d", page, got, want)
			}
			if strings.Contains(html, "Billet") {
				t.Errorf("%s: the English list shows a French post", page)
			}
		}
		if exists(dir, "blog/page/4/index.html") {
			t.Error("blog/page/4/ built, but 12 posts fill three pages")
		}

		// French: 3 posts plus 12 fallback copies, 5 + 5 + 5, linked under /fr/.
		for _, page := range []string{"fr/blog/index.html", "fr/blog/page/2/index.html", "fr/blog/page/3/index.html"} {
			html := read(page)
			if got := countListEntries(html, "blog/list"); got != 5 {
				t.Errorf("%s: %d entries, want 5", page, got)
			}
			if strings.Contains(html, `href="/blog/post-`) {
				t.Errorf("%s: links an English post instead of its /fr/ fallback copy", page)
			}
		}
		if !strings.Contains(read("fr/blog/index.html"), "Billet 03") {
			t.Error("fr/blog/: the newest French post is missing from the first page")
		}
		if !strings.Contains(read("fr/blog/page/2/index.html"), `href="/fr/blog/post-`) {
			t.Error("fr/blog/page/2/: want fallback copies linked under /fr/")
		}
		if exists(dir, "fr/blog/page/4/index.html") {
			t.Error("fr/blog/page/4/ built, but 15 entries fill three pages")
		}
	})

	t.Run("fallback omit", func(t *testing.T) {
		dir, cfg := writeBilingualBlog(t)
		cfg.I18n.Fallback = "omit"
		read := buildImageSite(t, dir, cfg)

		html := read("fr/blog/index.html")
		if got := countListEntries(html, "blog/list"); got != 3 {
			t.Errorf("fr/blog/: %d entries, want the 3 French posts", got)
		}
		if strings.Contains(html, "Post 0") || strings.Contains(html, "Post 1") {
			t.Error("fr/blog/ lists an English post under omit")
		}
		if exists(dir, "fr/blog/page/2/index.html") {
			t.Error("fr/blog/page/2/ built, but 3 posts fit on one page")
		}
		if !exists(dir, "blog/page/3/index.html") {
			t.Error("blog/page/3/ missing: the English list has 12 posts")
		}
	})
}

// Taxonomy paginate_by: 0 turns term pagination off, like a collection's
// paginate: 0: the term page lists every post and no numbered term pages are
// built. Unset keeps 10 per page.
func TestBuild_TaxonomyPaginateByZero(t *testing.T) {
	writeTagged := func(t *testing.T) string {
		dir := t.TempDir()
		writeFixture(t, dir, "content/_index.md", "---\ntitle: Home\n---\nWelcome.\n")
		writeFixture(t, dir, "content/blog/_index.md", "---\ntitle: Field Notes\n---\n")
		for i := 1; i <= 12; i++ {
			writeFixture(t, dir, fmt.Sprintf("content/blog/post-%02d.md", i),
				fmt.Sprintf("---\ntitle: Post %02d\ndate: 2025-01-%02dT00:00:00Z\ntags: [botany]\n---\nNotes from day %d.\n", i, i, i))
		}
		return dir
	}
	exists := func(dir, rel string) bool {
		_, err := os.Stat(filepath.Join(dir, "dist", filepath.FromSlash(rel)))
		return err == nil
	}

	t.Run("paginate_by 0", func(t *testing.T) {
		dir := writeTagged(t)
		cfg := config.Defaults()
		cfg.Taxonomies = map[string]config.TaxonomyConfig{"tags": {Singular: "tag", PaginateBy: config.IntPtr(0)}}
		read := buildImageSite(t, dir, cfg)
		html := read("tags/botany/index.html")
		if got := strings.Count(html, `class="sarde-post-card-title"`); got != 12 {
			t.Errorf("term page lists %d posts, want all 12", got)
		}
		if !strings.Contains(html, `sarde-taxonomy-term-count">12 `) {
			t.Error("term page does not show the post count")
		}
		if strings.Contains(html, "sarde-pagination-info") {
			t.Error("unpaginated term page renders the pagination nav")
		}
		if exists(dir, "tags/botany/page/2/index.html") {
			t.Error("tags/botany/page/2/ must not be built with paginate_by: 0")
		}
	})

	t.Run("unset keeps 10 per page", func(t *testing.T) {
		dir := writeTagged(t)
		cfg := config.Defaults()
		cfg.Taxonomies = map[string]config.TaxonomyConfig{"tags": {Singular: "tag"}}
		read := buildImageSite(t, dir, cfg)
		if got := strings.Count(read("tags/botany/index.html"), `class="sarde-post-card-title"`); got != 10 {
			t.Errorf("term page lists %d posts, want 10", got)
		}
		if !exists(dir, "tags/botany/page/2/index.html") {
			t.Error("tags/botany/page/2/ must exist")
		}
	})
}
