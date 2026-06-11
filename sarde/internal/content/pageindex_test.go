package content

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/frostybee/sarde/internal/engine"
)

func TestBuildPageIndex(t *testing.T) {
	pages := []*engine.Page{
		{PageIdentity: engine.PageIdentity{Slug: "hello", RelPermalink: "/blog/hello/", Permalink: "/blog/hello/"}},
		{PageIdentity: engine.PageIdentity{Slug: "guide", RelPermalink: "/docs/guide/", Permalink: "/docs/guide/"}},
		{PageIdentity: engine.PageIdentity{Slug: "about", RelPermalink: "/about/", Permalink: "/about/"}},
	}
	idx := BuildPageIndex(pages)

	if idx.PageCount() != 3 {
		t.Errorf("PageCount() = %d, want 3", idx.PageCount())
	}

	for _, p := range pages {
		if !idx.HasPage(p.Permalink) {
			t.Errorf("HasPage(%q) = false, want true", p.Permalink)
		}
		got := idx.LookupByPermalink(p.Permalink)
		if got != p {
			t.Errorf("LookupByPermalink(%q) returned wrong page", p.Permalink)
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
	first := &engine.Page{PageIdentity: engine.PageIdentity{Slug: "intro", RelPermalink: "/docs/intro/", Permalink: "/docs/intro/"}}
	second := &engine.Page{PageIdentity: engine.PageIdentity{Slug: "intro", RelPermalink: "/blog/intro/", Permalink: "/blog/intro/"}}
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

func TestBuildPageIndex_RecordsPermalinkCollision(t *testing.T) {
	// Two distinct pages claiming the same Permalink. Because they also share
	// RelPermalink within one lane, this is simultaneously a byLane dup — it must
	// be recorded exactly once (via byPermalink), never double-counted.
	first := &engine.Page{PageIdentity: engine.PageIdentity{
		RelPermalink: "/docs/guide/", Permalink: "/docs/guide/", FilePath: "content/docs/a.md",
	}}
	second := &engine.Page{PageIdentity: engine.PageIdentity{
		RelPermalink: "/docs/guide/", Permalink: "/docs/guide/", FilePath: "content/docs/b.md",
	}}
	idx := BuildPageIndex([]*engine.Page{first, second})

	cols := idx.Collisions()
	if len(cols) != 1 {
		t.Fatalf("Collisions() returned %d entries, want 1 (recorded once via byPermalink)", len(cols))
	}
	got := cols[0]
	if got.Permalink != "/docs/guide/" {
		t.Errorf("Collision.Permalink = %q, want /docs/guide/", got.Permalink)
	}
	if got.KeptFile != first.FilePath {
		t.Errorf("Collision.KeptFile = %q, want %q (first registered)", got.KeptFile, first.FilePath)
	}
	if got.DroppedFile != second.FilePath {
		t.Errorf("Collision.DroppedFile = %q, want %q (later page)", got.DroppedFile, second.FilePath)
	}
	// First-match resolution preserved: the first page wins the URL.
	if idx.LookupByPermalink("/docs/guide/") != first {
		t.Error("LookupByPermalink should return the first-registered page")
	}
}

func TestBuildPageIndex_NoCollisionWhenDistinct(t *testing.T) {
	pages := []*engine.Page{
		{PageIdentity: engine.PageIdentity{RelPermalink: "/blog/hello/", Permalink: "/blog/hello/", FilePath: "a.md"}},
		{PageIdentity: engine.PageIdentity{RelPermalink: "/docs/guide/", Permalink: "/docs/guide/", FilePath: "b.md"}},
	}
	idx := BuildPageIndex(pages)
	if got := len(idx.Collisions()); got != 0 {
		t.Errorf("Collisions() returned %d entries, want 0 for a clean index", got)
	}
}

func TestPageIndexEmptySlugOrPermalink(t *testing.T) {
	orphan := &engine.Page{PageIdentity: engine.PageIdentity{Slug: "orphan", RelPermalink: "", Permalink: ""}}
	pages := []*engine.Page{
		{PageIdentity: engine.PageIdentity{Slug: "", RelPermalink: "/page/", Permalink: "/page/"}},
		orphan,
	}
	idx := BuildPageIndex(pages)

	if idx.PageCount() != 1 {
		t.Errorf("PageCount() = %d, want 1 (only page with non-empty permalink)", idx.PageCount())
	}
	if idx.LookupBySlug("orphan") != orphan {
		t.Error("LookupBySlug(orphan) should return the page even when permalink is empty")
	}
	if !idx.HasPage("/page/") {
		t.Error("HasPage(/page/) = false, want true")
	}
	if idx.LookupBySlug("") != nil {
		t.Error("LookupBySlug('') should return nil")
	}
}

func TestPageIndexHeadings(t *testing.T) {
	idx := BuildPageIndex([]*engine.Page{
		{PageIdentity: engine.PageIdentity{Slug: "post", RelPermalink: "/blog/post/"}},
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
		{PageIdentity: engine.PageIdentity{Slug: "post", RelPermalink: "/blog/post/"}},
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

func TestLookupInLane(t *testing.T) {
	enLatest := &engine.Page{
		PageIdentity: engine.PageIdentity{Slug: "auth", RelPermalink: "/docs/guides/auth/", Permalink: "/docs/guides/auth/"},
		PageI18n:     engine.PageI18n{Lang: "en"},
		PageVersioning: engine.PageVersioning{Version: "v2"},
	}
	enV1 := &engine.Page{
		PageIdentity: engine.PageIdentity{Slug: "auth", RelPermalink: "/docs/guides/auth/", Permalink: "/docs/v1/guides/auth/"},
		PageI18n:     engine.PageI18n{Lang: "en"},
		PageVersioning: engine.PageVersioning{Version: "v1"},
	}
	frLatest := &engine.Page{
		PageIdentity: engine.PageIdentity{Slug: "auth", RelPermalink: "/docs/guides/auth/", Permalink: "/fr/docs/guides/auth/"},
		PageI18n:     engine.PageI18n{Lang: "fr"},
		PageVersioning: engine.PageVersioning{Version: "v2"},
	}
	blogPost := &engine.Page{
		PageIdentity: engine.PageIdentity{Slug: "hello", RelPermalink: "/blog/hello/", Permalink: "/blog/hello/"},
		PageI18n:     engine.PageI18n{Lang: "en"},
	}

	idx := BuildPageIndex([]*engine.Page{enLatest, enV1, frLatest, blogPost})

	tests := []struct {
		name         string
		relPermalink string
		lang         string
		version      string
		want         *engine.Page
	}{
		{"en latest", "/docs/guides/auth/", "en", "v2", enLatest},
		{"en v1", "/docs/guides/auth/", "en", "v1", enV1},
		{"fr latest", "/docs/guides/auth/", "fr", "v2", frLatest},
		{"blog en", "/blog/hello/", "en", "", blogPost},
		{"wrong lang", "/docs/guides/auth/", "de", "v2", nil},
		{"wrong version", "/docs/guides/auth/", "en", "v3", nil},
		{"wrong path", "/docs/guides/missing/", "en", "v2", nil},
		{"empty lane", "/anything/", "xx", "yy", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := idx.LookupInLane(tt.relPermalink, tt.lang, tt.version)
			if got != tt.want {
				t.Errorf("LookupInLane(%q, %q, %q) = %v, want %v", tt.relPermalink, tt.lang, tt.version, got, tt.want)
			}
		})
	}
}

func TestLookupInLaneUnversioned(t *testing.T) {
	page := &engine.Page{
		PageIdentity: engine.PageIdentity{Slug: "post", RelPermalink: "/blog/post/", Permalink: "/blog/post/"},
		PageI18n:     engine.PageI18n{Lang: "en"},
	}
	idx := BuildPageIndex([]*engine.Page{page})

	got := idx.LookupInLane("/blog/post/", "en", "")
	if got != page {
		t.Error("LookupInLane for unversioned page should return the page")
	}

	got = idx.LookupInLane("/blog/post/", "en", "v1")
	if got != nil {
		t.Error("LookupInLane with wrong version should return nil")
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
		// Digit-only "extensions" are versioned page paths, not files.
		{"/docs/v1.2", "/docs/v1.2/"},
		{"/docs/v1.2/", "/docs/v1.2/"},
		{"/release-2.0", "/release-2.0/"},
		// Real extensions (containing letters) stay file-like.
		{"/jquery.min.js", "/jquery.min.js"},
		{"/archive.tar.gz", "/archive.tar.gz"},
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
