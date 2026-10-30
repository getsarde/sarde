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
