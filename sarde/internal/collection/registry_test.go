package collection

import (
	"html/template"
	"os"
	"path/filepath"
	"testing"

	"github.com/frostybee/sarde/internal/config"
	"github.com/frostybee/sarde/internal/content"
	"github.com/frostybee/sarde/internal/engine"
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
	if docs.Config.SortBy != "order" {
		t.Errorf("SortBy = %q, want %q", docs.Config.SortBy, "order")
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
	if contentPages[0].Sidebar.Order > contentPages[1].Sidebar.Order {
		t.Errorf("docs should be sorted by weight asc: got %d, %d", contentPages[0].Sidebar.Order, contentPages[1].Sidebar.Order)
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
			t.Error("first content page should have NextPage")
		}
		// For docs-layout, prev/next is wired from the nav tree flat order
		// which may include the section index page before the first content page.
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

func TestBuildCollections_FeaturedFiltering(t *testing.T) {
	dir := t.TempDir()
	contentDir := filepath.Join(dir, "content")
	writeTestFile(t, contentDir, filepath.Join("blog", "_index.md"), "---\ntitle: Blog\n---\n")
	writeTestFile(t, contentDir, filepath.Join("blog", "a.md"), "---\ntitle: Plain\ndate: 2026-01-01T00:00:00Z\n---\nBody\n")
	writeTestFile(t, contentDir, filepath.Join("blog", "b.md"), "---\ntitle: Featured\ndate: 2026-02-01T00:00:00Z\nfeatured: true\n---\nBody\n")

	scanner := &content.Scanner{}
	files, err := scanner.DiscoverFiles(contentDir)
	if err != nil {
		t.Fatal(err)
	}

	collections, _, err := BuildCollections(files, config.Defaults(), contentDir)
	if err != nil {
		t.Fatal(err)
	}

	blog := collections["blog"]
	if blog == nil {
		t.Fatal("blog collection missing")
	}
	if len(blog.Featured) != 1 {
		t.Fatalf("Featured count = %d, want 1", len(blog.Featured))
	}
	if blog.Featured[0].Title != "Featured" {
		t.Errorf("Featured[0].Title = %q, want %q", blog.Featured[0].Title, "Featured")
	}
}

func TestBuildCollections_RelPathPopulated(t *testing.T) {
	contentDir, files := createTestSite(t)
	siteCfg := config.Defaults()

	collections, _, err := BuildCollections(files, siteCfg, contentDir)
	if err != nil {
		t.Fatal(err)
	}
	blog := collections["blog"]
	if blog == nil {
		t.Fatal("blog collection missing")
	}
	for _, p := range blog.Pages {
		if p.RelPath == "" {
			t.Errorf("page %q has empty RelPath", p.Title)
		}
		// Must be POSIX slashes.
		if filepath.Separator == '\\' && containsBackslash(p.RelPath) {
			t.Errorf("page RelPath %q should use forward slashes", p.RelPath)
		}
	}
}

func containsBackslash(s string) bool {
	for _, r := range s {
		if r == '\\' {
			return true
		}
	}
	return false
}

// unused suppress
var _ = engine.KindPage
var _ template.HTML

// buildSinglePage creates a minimal docs collection with one page and returns it.
func buildSinglePage(t *testing.T, frontmatter, body string) *engine.Page {
	t.Helper()
	dir := t.TempDir()
	contentDir := filepath.Join(dir, "content")
	writeTestFile(t, contentDir, filepath.Join("docs", "_index.md"), "---\ntitle: Docs\n---\n")
	writeTestFile(t, contentDir, filepath.Join("docs", "page.md"), "---\n"+frontmatter+"\n---\n"+body+"\n")

	scanner := &content.Scanner{}
	files, err := scanner.DiscoverFiles(contentDir)
	if err != nil {
		t.Fatal(err)
	}

	collections, _, err := BuildCollections(files, config.Defaults(), contentDir)
	if err != nil {
		t.Fatal(err)
	}

	docs := collections["docs"]
	if docs == nil {
		t.Fatal("docs collection missing")
	}

	for _, p := range docs.Pages {
		if p.Kind != engine.KindSection {
			return p
		}
	}
	t.Fatal("no non-section page found")
	return nil
}

func TestBuildStandalonePages(t *testing.T) {
	contentDir, files := createTestSite(t)

	pages, err := BuildStandalonePages(files, contentDir, 70, "")
	if err != nil {
		t.Fatal(err)
	}

	// Should have _index.md (home) + about.md (standalone)
	if len(pages) != 2 {
		t.Fatalf("standalone pages = %d, want 2", len(pages))
	}
}

func TestBuildPages_ImageTransferred(t *testing.T) {
	page := buildSinglePage(t, "title: Test\nimage: /img/cover.jpg", "Body content.")
	if page.Image != "/img/cover.jpg" {
		t.Errorf("Image = %q, want %q", page.Image, "/img/cover.jpg")
	}
}

func TestBuildPages_SummaryFromFrontmatter(t *testing.T) {
	page := buildSinglePage(t, "title: Test\nsummary: \"Manual summary.\"", "A very long paragraph that would normally be auto-summarized by the transformer.")
	if string(page.Summary) != "Manual summary." {
		t.Errorf("Summary = %q, want %q", page.Summary, "Manual summary.")
	}
}

func TestBuildPages_RenderFalseTransferred(t *testing.T) {
	dir := t.TempDir()
	contentDir := filepath.Join(dir, "content")
	writeTestFile(t, contentDir, filepath.Join("docs", "_index.md"), "---\ntitle: Docs\n---\n")
	writeTestFile(t, contentDir, filepath.Join("docs", "hidden", "_index.md"), "---\ntitle: Hidden Section\nrender: false\n---\n")
	writeTestFile(t, contentDir, filepath.Join("docs", "hidden", "page.md"), "---\ntitle: Inner Page\n---\nContent.\n")

	scanner := &content.Scanner{}
	files, err := scanner.DiscoverFiles(contentDir)
	if err != nil {
		t.Fatal(err)
	}

	collections, _, err := BuildCollections(files, config.Defaults(), contentDir)
	if err != nil {
		t.Fatal(err)
	}

	docs := collections["docs"]
	if docs == nil {
		t.Fatal("docs collection missing")
	}

	for _, p := range docs.Pages {
		if p.Title == "Hidden Section" {
			v, ok := p.Params["render"].(bool)
			if !ok || v != false {
				t.Errorf("expected Params[render] = false, got %v", p.Params["render"])
			}
			return
		}
	}
	t.Error("Hidden Section page not found")
}

func TestBuildPages_TransparentTransferred(t *testing.T) {
	dir := t.TempDir()
	contentDir := filepath.Join(dir, "content")
	writeTestFile(t, contentDir, filepath.Join("docs", "_index.md"), "---\ntitle: Docs\n---\n")
	writeTestFile(t, contentDir, filepath.Join("docs", "flat", "_index.md"), "---\ntitle: Flat\ntransparent: true\n---\n")
	writeTestFile(t, contentDir, filepath.Join("docs", "flat", "page.md"), "---\ntitle: Child\n---\nContent.\n")

	scanner := &content.Scanner{}
	files, err := scanner.DiscoverFiles(contentDir)
	if err != nil {
		t.Fatal(err)
	}

	collections, _, err := BuildCollections(files, config.Defaults(), contentDir)
	if err != nil {
		t.Fatal(err)
	}

	docs := collections["docs"]
	for _, p := range docs.Pages {
		if p.Title == "Flat" {
			v, ok := p.Params["transparent"].(bool)
			if !ok || v != true {
				t.Errorf("expected Params[transparent] = true, got %v", p.Params["transparent"])
			}
			return
		}
	}
	t.Error("Flat section page not found")
}

func TestBuildPages_PrevNextTransferred(t *testing.T) {
	page := buildSinglePage(t, "title: Test\nprev: getting-started\nnext: advanced", "Body.")
	prev, ok := page.Params["prev"].(*engine.NavOverride)
	if !ok || prev.Slug != "getting-started" {
		t.Errorf("Params[prev].Slug = %v, want %q", page.Params["prev"], "getting-started")
	}
	next, ok := page.Params["next"].(*engine.NavOverride)
	if !ok || next.Slug != "advanced" {
		t.Errorf("Params[next].Slug = %v, want %q", page.Params["next"], "advanced")
	}
}

func TestBuildPages_SidebarAttrsTransferred(t *testing.T) {
	page := buildSinglePage(t, "title: Test\nsidebar:\n  attrs:\n    icon: book\n    color: blue", "Body.")
	if page.Sidebar.Attrs == nil {
		t.Fatal("expected Sidebar.Attrs to be set")
	}
	if page.Sidebar.Attrs["icon"] != "book" {
		t.Errorf("Sidebar.Attrs[icon] = %q, want %q", page.Sidebar.Attrs["icon"], "book")
	}
	if page.Sidebar.Attrs["color"] != "blue" {
		t.Errorf("Sidebar.Attrs[color] = %q, want %q", page.Sidebar.Attrs["color"], "blue")
	}
}

func TestBuildPages_TOCFieldsTransferred(t *testing.T) {
	page := buildSinglePage(t, "title: Test\ntoc:\n  enabled: true\n  min_level: 2\n  max_level: 5", "Body.")
	if page.TOC.Enabled == nil || *page.TOC.Enabled != true {
		t.Errorf("TOC.Enabled = %v, want true", page.TOC.Enabled)
	}
	if page.TOC.MinLevel != 2 {
		t.Errorf("TOC.MinLevel = %v, want 2", page.TOC.MinLevel)
	}
	if page.TOC.MaxLevel != 5 {
		t.Errorf("TOC.MaxLevel = %v, want 5", page.TOC.MaxLevel)
	}
}

func TestBuildPages_PagefindTransferred(t *testing.T) {
	page := buildSinglePage(t, "title: Test\npagefind: false", "Body.")
	if page.Params["pagefind"] != false {
		t.Errorf("Params[pagefind] = %v, want false", page.Params["pagefind"])
	}
}

func TestBuildPages_SidebarGroupTransferred(t *testing.T) {
	page := buildSinglePage(t, "title: Test\nsidebar:\n  group: Reference", "Body.")
	if page.Sidebar.Group != "Reference" {
		t.Errorf("Sidebar.Group = %q, want %q", page.Sidebar.Group, "Reference")
	}
}

func TestBuildPages_LayoutTransferred(t *testing.T) {
	page := buildSinglePage(t, "title: Test\nlayout: splash", "Body.")
	if page.Params["layout"] != "splash" {
		t.Errorf("Params[layout] = %v, want %q", page.Params["layout"], "splash")
	}
}

func TestBuildPages_TypeTransferred(t *testing.T) {
	page := buildSinglePage(t, "title: Test\ntype: tutorial", "Body.")
	if page.Params["type"] != "tutorial" {
		t.Errorf("Params[type] = %v, want %q", page.Params["type"], "tutorial")
	}
}

func TestBuildPages_HeroTransferred(t *testing.T) {
	fm := "title: Home\nhero:\n  title: Welcome\n  tagline: Fast sites\n  actions:\n    - text: Get Started\n      link: /docs/\n      variant: primary"
	page := buildSinglePage(t, fm, "Body.")
	raw, ok := page.Params["hero"]
	if !ok {
		t.Fatal("expected Params[hero] to be set")
	}
	hero, ok := raw.(*engine.HeroConfig)
	if !ok {
		t.Fatalf("expected *engine.HeroConfig, got %T", raw)
	}
	if hero.Title != "Welcome" {
		t.Errorf("Hero.Title = %q, want %q", hero.Title, "Welcome")
	}
	if hero.Tagline != "Fast sites" {
		t.Errorf("Hero.Tagline = %q, want %q", hero.Tagline, "Fast sites")
	}
	if len(hero.Actions) != 1 {
		t.Fatalf("Hero.Actions len = %d, want 1", len(hero.Actions))
	}
	if hero.Actions[0].Variant != "primary" {
		t.Errorf("Hero.Actions[0].Variant = %q, want %q", hero.Actions[0].Variant, "primary")
	}
}

func TestNormalizeRootVersionPages(t *testing.T) {
	t.Run("assigns lastVersion to root pages", func(t *testing.T) {
		pages := []*engine.Page{
			{PageIdentity: engine.PageIdentity{RelPermalink: "/docs/guide/intro/"}, PageI18n: engine.PageI18n{LangRelPath: "docs/guide/intro.md"}},
			{PageIdentity: engine.PageIdentity{RelPermalink: "/docs/"}, PageI18n: engine.PageI18n{LangRelPath: "docs/_index.md"}},
		}
		normalizeRootVersionPages(pages, "v2")

		for _, p := range pages {
			if p.Version != "v2" {
				t.Errorf("page %s: Version = %q, want %q", p.RelPermalink, p.Version, "v2")
			}
		}
		if pages[0].VersionRelPath != "guide/intro.md" {
			t.Errorf("VersionRelPath = %q, want %q", pages[0].VersionRelPath, "guide/intro.md")
		}
		if pages[1].VersionRelPath != "_index.md" {
			t.Errorf("VersionRelPath = %q, want %q", pages[1].VersionRelPath, "_index.md")
		}
	})

	t.Run("skips already-versioned pages", func(t *testing.T) {
		pages := []*engine.Page{
			{PageVersioning: engine.PageVersioning{Version: "v1", VersionRelPath: "guide/intro.md"}, PageI18n: engine.PageI18n{LangRelPath: "docs/v1/guide/intro.md"}},
		}
		normalizeRootVersionPages(pages, "v2")

		if pages[0].Version != "v1" {
			t.Errorf("Version = %q, want %q (should not be overwritten)", pages[0].Version, "v1")
		}
		if pages[0].VersionRelPath != "guide/intro.md" {
			t.Errorf("VersionRelPath = %q, want %q", pages[0].VersionRelPath, "guide/intro.md")
		}
	})

	t.Run("no-op with empty lastVersion", func(t *testing.T) {
		pages := []*engine.Page{
			{PageI18n: engine.PageI18n{LangRelPath: "docs/guide/intro.md"}},
		}
		normalizeRootVersionPages(pages, "")

		if pages[0].Version != "" {
			t.Errorf("Version = %q, want empty (no-op)", pages[0].Version)
		}
	})
}
