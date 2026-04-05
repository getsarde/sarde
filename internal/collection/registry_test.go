package collection

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/coderoo-dev/coderoo/internal/config"
	"github.com/coderoo-dev/coderoo/internal/content"
	"github.com/coderoo-dev/coderoo/internal/engine"
)

func createTestSite(t *testing.T) (string, []content.ContentFile) {
	t.Helper()
	dir := t.TempDir()
	contentDir := filepath.Join(dir, "content")

	writeTestFile(t, contentDir, "_index.md", "---\ntitle: Home\n---\n# Welcome\n")
	writeTestFile(t, contentDir, "about.md", "---\ntitle: About\n---\nAbout page.\n")

	// Blog
	writeTestFile(t, contentDir, filepath.Join("blog", "_index.md"), "---\ntitle: Blog\n---\n")
	writeTestFile(t, contentDir, filepath.Join("blog", "2024-post.md"), "---\ntitle: Old Post\ndate: 2024-01-01T00:00:00Z\n---\nOld content.\n")
	writeTestFile(t, contentDir, filepath.Join("blog", "2026-post.md"), "---\ntitle: New Post\ndate: 2026-06-01T00:00:00Z\n---\nNew content.\n")
	writeTestFile(t, contentDir, filepath.Join("blog", "draft-post.md"), "---\ntitle: Draft\ndraft: true\n---\nDraft content.\n")

	// Docs
	writeTestFile(t, contentDir, filepath.Join("docs", "_index.md"), "---\ntitle: Documentation\n---\n")
	writeTestFile(t, contentDir, filepath.Join("docs", "01-getting-started.md"), "---\ntitle: Getting Started\n---\nIntro.\n")
	writeTestFile(t, contentDir, filepath.Join("docs", "02-advanced.md"), "---\ntitle: Advanced\n---\nAdvanced.\n")

	scanner := &content.Scanner{}
	files, err := scanner.DiscoverFiles(contentDir)
	if err != nil {
		t.Fatal(err)
	}
	return contentDir, files
}

func writeTestFile(t *testing.T, base, rel, body string) {
	t.Helper()
	path := filepath.Join(base, rel)
	os.MkdirAll(filepath.Dir(path), 0755)
	os.WriteFile(path, []byte(body), 0644)
}

func TestBuildCollections_BlogDefaults(t *testing.T) {
	contentDir, files := createTestSite(t)
	siteCfg := config.Defaults()

	collections, _, err := BuildCollections(files, siteCfg, contentDir)
	if err != nil {
		t.Fatalf("BuildCollections error: %v", err)
	}

	blog, ok := collections["blog"]
	if !ok {
		t.Fatal("blog collection not found")
	}
	if blog.Config.SortBy != "date" {
		t.Errorf("SortBy = %q, want %q", blog.Config.SortBy, "date")
	}
	if blog.Config.Layout != engine.LayoutDefault {
		t.Errorf("Layout = %q, want %q", blog.Config.Layout, engine.LayoutDefault)
	}
	if !blog.Config.Feed {
		t.Error("Feed should be true")
	}
}

func TestBuildCollections_DocsDefaults(t *testing.T) {
	contentDir, files := createTestSite(t)
	siteCfg := config.Defaults()

	collections, _, err := BuildCollections(files, siteCfg, contentDir)
	if err != nil {
		t.Fatalf("BuildCollections error: %v", err)
	}

	docs, ok := collections["docs"]
	if !ok {
		t.Fatal("docs collection not found")
	}
	if docs.Config.SortBy != "weight" {
		t.Errorf("SortBy = %q, want %q", docs.Config.SortBy, "weight")
	}
	if docs.Config.Layout != engine.LayoutDocs {
		t.Errorf("Layout = %q, want %q", docs.Config.Layout, engine.LayoutDocs)
	}
}

func TestBuildCollections_DraftFiltering(t *testing.T) {
	contentDir, files := createTestSite(t)
	siteCfg := config.Defaults()
	// Defaults have drafts=false

	collections, _, err := BuildCollections(files, siteCfg, contentDir)
	if err != nil {
		t.Fatal(err)
	}

	blog := collections["blog"]
	for _, p := range blog.Pages {
		if p.Draft {
			t.Errorf("draft page %q should have been filtered", p.Title)
		}
	}
}

func TestBuildCollections_DraftIncluded(t *testing.T) {
	contentDir, files := createTestSite(t)
	siteCfg := config.Defaults()
	siteCfg.Build.Drafts = config.BoolPtr(true)

	collections, _, err := BuildCollections(files, siteCfg, contentDir)
	if err != nil {
		t.Fatal(err)
	}

	blog := collections["blog"]
	hasDraft := false
	for _, p := range blog.Pages {
		if p.Draft {
			hasDraft = true
		}
	}
	if !hasDraft {
		t.Error("draft page should be included when Drafts=true")
	}
}

func TestBuildCollections_BlogSortDateDesc(t *testing.T) {
	contentDir, files := createTestSite(t)
	siteCfg := config.Defaults()

	collections, _, err := BuildCollections(files, siteCfg, contentDir)
	if err != nil {
		t.Fatal(err)
	}

	blog := collections["blog"]
	// Filter to non-section, non-draft pages
	var posts []*engine.Page
	for _, p := range blog.Pages {
		if p.Kind != engine.KindSection && !p.Draft {
			posts = append(posts, p)
		}
	}
	if len(posts) < 2 {
		t.Fatalf("expected at least 2 posts, got %d", len(posts))
	}
	// First should be newest
	if posts[0].Title != "New Post" {
		t.Errorf("first post = %q, want %q (newest first)", posts[0].Title, "New Post")
	}
}

func TestBuildCollections_DocsSortWeightAsc(t *testing.T) {
	contentDir, files := createTestSite(t)
	siteCfg := config.Defaults()

	collections, _, err := BuildCollections(files, siteCfg, contentDir)
	if err != nil {
		t.Fatal(err)
	}

	docs := collections["docs"]
	var contentPages []*engine.Page
	for _, p := range docs.Pages {
		if p.Kind != engine.KindSection {
			contentPages = append(contentPages, p)
		}
	}
	if len(contentPages) < 2 {
		t.Fatalf("expected at least 2 docs pages, got %d", len(contentPages))
	}
	if contentPages[0].Weight > contentPages[1].Weight {
		t.Errorf("docs should be sorted by weight asc: got %d, %d", contentPages[0].Weight, contentPages[1].Weight)
	}
}

func TestBuildCollections_PrevNext(t *testing.T) {
	contentDir, files := createTestSite(t)
	siteCfg := config.Defaults()

	collections, _, err := BuildCollections(files, siteCfg, contentDir)
	if err != nil {
		t.Fatal(err)
	}

	docs := collections["docs"]
	var contentPages []*engine.Page
	for _, p := range docs.Pages {
		if p.Kind != engine.KindSection {
			contentPages = append(contentPages, p)
		}
	}
	if len(contentPages) >= 2 {
		if contentPages[0].NextPage == nil {
			t.Error("first page should have NextPage")
		}
		if contentPages[0].PrevPage != nil {
			t.Error("first page should not have PrevPage")
		}
		if contentPages[len(contentPages)-1].PrevPage == nil {
			t.Error("last page should have PrevPage")
		}
	}
}

func TestBuildCollections_CollectionBackref(t *testing.T) {
	contentDir, files := createTestSite(t)
	siteCfg := config.Defaults()

	collections, _, err := BuildCollections(files, siteCfg, contentDir)
	if err != nil {
		t.Fatal(err)
	}

	for name, col := range collections {
		for _, p := range col.Pages {
			if p.Collection != col {
				t.Errorf("%s: page %q Collection backref not set", name, p.Title)
			}
		}
	}
}

func TestBuildCollections_IndexPage(t *testing.T) {
	contentDir, files := createTestSite(t)
	siteCfg := config.Defaults()

	collections, _, err := BuildCollections(files, siteCfg, contentDir)
	if err != nil {
		t.Fatal(err)
	}

	docs := collections["docs"]
	if docs.IndexPage == nil {
		t.Error("docs IndexPage should be set")
	} else if docs.IndexPage.Title != "Documentation" {
		t.Errorf("docs IndexPage.Title = %q, want %q", docs.IndexPage.Title, "Documentation")
	}

	if docs.Title != "Documentation" {
		t.Errorf("docs.Title = %q, want %q", docs.Title, "Documentation")
	}
}

func TestBuildStandalonePages(t *testing.T) {
	contentDir, files := createTestSite(t)

	pages, err := BuildStandalonePages(files, contentDir, 70)
	if err != nil {
		t.Fatal(err)
	}

	// Should have _index.md (home) + about.md (standalone)
	if len(pages) != 2 {
		t.Fatalf("standalone pages = %d, want 2", len(pages))
	}
}
