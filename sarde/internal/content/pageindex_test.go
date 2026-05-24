package content

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/frostybee/sarde/internal/engine"
)

func TestBuildPageIndex(t *testing.T) {
	pages := []*engine.Page{
		{Slug: "hello", RelPermalink: "/blog/hello/", Permalink: "/blog/hello/"},
		{Slug: "guide", RelPermalink: "/docs/guide/", Permalink: "/docs/guide/"},
		{Slug: "about", RelPermalink: "/about/", Permalink: "/about/"},
	}
	idx := BuildPageIndex(pages)

	if idx.PageCount() != 3 {
		t.Errorf("PageCount() = %d, want 3", idx.PageCount())
	}

	for _, p := range pages {
		if !idx.HasPage(p.RelPermalink) {
			t.Errorf("HasPage(%q) = false, want true", p.RelPermalink)
		}
		got := idx.LookupByPermalink(p.RelPermalink)
		if got != p {
			t.Errorf("LookupByPermalink(%q) returned wrong page", p.RelPermalink)
		}
		got = idx.LookupBySlug(p.Slug)
		if got != p {
			t.Errorf("LookupBySlug(%q) returned wrong page", p.Slug)
		}
	}

	if idx.HasPage("/nonexistent/") {
		t.Error("HasPage(/nonexistent/) = true, want false")
	}
	if idx.LookupByPermalink("/nonexistent/") != nil {
		t.Error("LookupByPermalink(/nonexistent/) != nil, want nil")
	}
	if idx.LookupBySlug("nonexistent") != nil {
		t.Error("LookupBySlug(nonexistent) != nil, want nil")
	}
}

func TestPageIndexSlugCollision(t *testing.T) {
	first := &engine.Page{Slug: "intro", RelPermalink: "/docs/intro/", Permalink: "/docs/intro/"}
	second := &engine.Page{Slug: "intro", RelPermalink: "/blog/intro/", Permalink: "/blog/intro/"}
	idx := BuildPageIndex([]*engine.Page{first, second})

	got := idx.LookupBySlug("intro")
	if got != first {
		t.Error("LookupBySlug(intro) should return first-indexed page")
	}

	if !idx.HasPage("/docs/intro/") {
		t.Error("HasPage(/docs/intro/) = false, want true")
	}
	if !idx.HasPage("/blog/intro/") {
		t.Error("HasPage(/blog/intro/) = false, want true")
	}
}

func TestPageIndexEmptySlugOrPermalink(t *testing.T) {
	orphan := &engine.Page{Slug: "orphan", RelPermalink: ""}
	pages := []*engine.Page{
		{Slug: "", RelPermalink: "/page/"},
		orphan,
	}
	idx := BuildPageIndex(pages)

	if idx.PageCount() != 1 {
		t.Errorf("PageCount() = %d, want 1 (only page with non-empty permalink)", idx.PageCount())
	}
	// A page with a slug is still findable by slug even without a permalink.
	if idx.LookupBySlug("orphan") != orphan {
		t.Error("LookupBySlug(orphan) should return the page even when permalink is empty")
	}
	// A page with no slug but a permalink is findable by permalink.
	if !idx.HasPage("/page/") {
		t.Error("HasPage(/page/) = false, want true")
	}
	if idx.LookupBySlug("") != nil {
		t.Error("LookupBySlug('') should return nil")
	}
}

func TestPageIndexHeadings(t *testing.T) {
	idx := BuildPageIndex([]*engine.Page{
		{Slug: "post", RelPermalink: "/blog/post/"},
	})

	idx.SetHeadings("/blog/post/", []string{"intro", "setup", "conclusion"})

	if !idx.HasHeading("/blog/post/", "_top") {
		t.Error("HasHeading(_top) = false, want true (_top is always prepended)")
	}
	if !idx.HasHeading("/blog/post/", "intro") {
		t.Error("HasHeading(intro) = false, want true")
	}
	if !idx.HasHeading("/blog/post/", "setup") {
		t.Error("HasHeading(setup) = false, want true")
	}
	if idx.HasHeading("/blog/post/", "nonexistent") {
		t.Error("HasHeading(nonexistent) = true, want false")
	}
	if idx.HasHeading("/unknown/", "intro") {
		t.Error("HasHeading on unknown page = true, want false")
	}

	headings := idx.HeadingsFor("/blog/post/")
	if len(headings) != 4 {
		t.Errorf("HeadingsFor() returned %d headings, want 4", len(headings))
	}
	if headings[0] != "_top" {
		t.Errorf("first heading = %q, want _top", headings[0])
	}
}

func TestPageIndexHeadingsEmpty(t *testing.T) {
	idx := BuildPageIndex([]*engine.Page{
		{Slug: "post", RelPermalink: "/blog/post/"},
	})

	idx.SetHeadings("/blog/post/", nil)

	if !idx.HasHeading("/blog/post/", "_top") {
		t.Error("HasHeading(_top) = false after SetHeadings(nil), want true")
	}
}

func TestPageIndexAssets(t *testing.T) {
	dir := t.TempDir()
	writeStaticFile(t, dir, "favicon.ico", "icon")
	writeStaticFile(t, dir, filepath.Join("images", "hero.png"), "img")
	writeStaticFile(t, dir, filepath.Join("docs", "spec.pdf"), "pdf")

	idx := BuildPageIndex(nil)
	idx.AddAssets(dir)

	tests := []struct {
		path string
		want bool
	}{
		{"/favicon.ico", true},
		{"/images/hero.png", true},
		{"/docs/spec.pdf", true},
		{"/nonexistent.txt", false},
		{"/images/", false},
	}
	for _, tt := range tests {
		if got := idx.HasAsset(tt.path); got != tt.want {
			t.Errorf("HasAsset(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestPageIndexAssetsNonexistentDir(t *testing.T) {
	idx := BuildPageIndex(nil)
	idx.AddAssets(filepath.Join(t.TempDir(), "does-not-exist"))

	if idx.HasAsset("/anything") {
		t.Error("HasAsset should return false when static dir doesn't exist")
	}
}

func TestPageIndexEmpty(t *testing.T) {
	idx := BuildPageIndex(nil)

	if idx.PageCount() != 0 {
		t.Errorf("PageCount() = %d, want 0", idx.PageCount())
	}
	if idx.HasPage("/anything/") {
		t.Error("HasPage on empty index = true, want false")
	}
	if idx.HasHeading("/anything/", "foo") {
		t.Error("HasHeading on empty index = true, want false")
	}
	if idx.HasAsset("/foo.png") {
		t.Error("HasAsset on empty index = true, want false")
	}
}

func TestNormalizePermalink(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"/", "/"},
		{"/blog/post", "/blog/post/"},
		{"/blog/post/", "/blog/post/"},
		{"/file.xml", "/file.xml"},
		{"/feed/atom.xml", "/feed/atom.xml"},
	}
	for _, tt := range tests {
		if got := NormalizePermalink(tt.input); got != tt.want {
			t.Errorf("NormalizePermalink(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func writeStaticFile(t *testing.T, base, rel, body string) {
	t.Helper()
	path := filepath.Join(base, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
