package template

import (
	"testing"

	"github.com/coderoo-dev/coderoo/internal/engine"
)

func baseSite() *engine.SiteContext {
	return &engine.SiteContext{
		Title:    "Test Site",
		BaseURL:  "https://example.com",
		Language: "en",
	}
}

func TestBuildRouteData_BlogPage(t *testing.T) {
	col := &engine.Collection{
		Name:   "blog",
		Title:  "Blog",
		Config: &engine.CollectionConfig{Layout: engine.LayoutDefault},
	}
	prev := &engine.Page{Title: "Previous Post", RelPermalink: "/blog/prev/"}
	next := &engine.Page{Title: "Next Post", RelPermalink: "/blog/next/"}
	page := &engine.Page{
		Title:    "My Post",
		Kind:     engine.KindPage,
		PrevPage: prev,
		NextPage: next,
		Collection: col,
	}

	rd := BuildRouteData(page, baseSite(), nil)

	if rd.Template != "blog/single" {
		t.Errorf("Template: got %q, want blog/single", rd.Template)
	}
	if rd.Layout != engine.LayoutDefault {
		t.Errorf("Layout: got %q, want default", rd.Layout)
	}
	if rd.HasSidebar {
		t.Error("HasSidebar should be false for blog")
	}
	if rd.Pagination == nil {
		t.Fatal("expected pagination")
	}
	if rd.Pagination.Prev.Title != "Previous Post" {
		t.Errorf("Prev: got %q", rd.Pagination.Prev.Title)
	}
	if rd.Pagination.Next.URL != "/blog/next/" {
		t.Errorf("Next URL: got %q", rd.Pagination.Next.URL)
	}
}

func TestBuildRouteData_DocsPage(t *testing.T) {
	col := &engine.Collection{
		Name:   "docs",
		Title:  "Documentation",
		Config: &engine.CollectionConfig{Layout: engine.LayoutDocs},
	}
	page := &engine.Page{
		Title:      "Getting Started",
		Kind:       engine.KindPage,
		Collection: col,
	}

	rd := BuildRouteData(page, baseSite(), nil)

	if rd.Template != "docs/single" {
		t.Errorf("Template: got %q", rd.Template)
	}
	if rd.Layout != engine.LayoutDocs {
		t.Errorf("Layout: got %q", rd.Layout)
	}
	if !rd.HasSidebar {
		t.Error("HasSidebar should be true for docs")
	}
	if rd.SidebarType != "nav" {
		t.Errorf("SidebarType: got %q", rd.SidebarType)
	}
}

func TestBuildRouteData_HomePage(t *testing.T) {
	page := &engine.Page{
		Title: "Welcome",
		Kind:  engine.KindHome,
	}

	rd := BuildRouteData(page, baseSite(), nil)

	if rd.Template != "home" {
		t.Errorf("Template: got %q", rd.Template)
	}
	if rd.HasSidebar {
		t.Error("HasSidebar should be false for home")
	}
}

func TestBuildRouteData_StandalonePage(t *testing.T) {
	page := &engine.Page{
		Title: "About",
		Kind:  engine.KindStandalone,
	}

	rd := BuildRouteData(page, baseSite(), nil)

	if rd.Template != "_default/single" {
		t.Errorf("Template: got %q", rd.Template)
	}
}

func TestBuildRouteData_SectionPage(t *testing.T) {
	section := &engine.Section{Title: "Guides"}
	col := &engine.Collection{
		Name:   "docs",
		Title:  "Documentation",
		Config: &engine.CollectionConfig{Layout: engine.LayoutDocs},
	}
	page := &engine.Page{
		Title:      "Guides",
		Kind:       engine.KindSection,
		Collection: col,
		Section:    section,
	}

	rd := BuildRouteData(page, baseSite(), nil)

	if rd.Template != "docs/list" {
		t.Errorf("Template: got %q", rd.Template)
	}
	if !rd.IsSection {
		t.Error("IsSection should be true")
	}
	if rd.Section != section {
		t.Error("Section not set")
	}
}

func TestBuildRouteData_FrontmatterTemplateOverride(t *testing.T) {
	col := &engine.Collection{
		Name:   "blog",
		Title:  "Blog",
		Config: &engine.CollectionConfig{Layout: engine.LayoutDefault},
	}
	page := &engine.Page{
		Title:      "Special Post",
		Kind:       engine.KindPage,
		Collection: col,
		Params:     map[string]any{"template": "blog/featured"},
	}

	rd := BuildRouteData(page, baseSite(), nil)

	if rd.Template != "blog/featured" {
		t.Errorf("Template: got %q, want blog/featured", rd.Template)
	}
}

func TestBuildRouteData_NoPagination(t *testing.T) {
	col := &engine.Collection{
		Name:   "blog",
		Config: &engine.CollectionConfig{Layout: engine.LayoutDefault},
	}
	page := &engine.Page{
		Kind:       engine.KindPage,
		Collection: col,
	}

	rd := BuildRouteData(page, baseSite(), nil)

	if rd.Pagination != nil {
		t.Error("expected nil pagination when no prev/next")
	}
}

func TestBuildRouteData_PaginatorOnListPage(t *testing.T) {
	// 12 pages, paginate by 5 → 3 pagination pages total.
	pages := make([]*engine.Page, 12)
	for i := range pages {
		pages[i] = &engine.Page{Kind: engine.KindPage, Title: "post", RelPermalink: "/blog/post/"}
	}
	col := &engine.Collection{
		Name:   "blog",
		Pages:  pages,
		Config: &engine.CollectionConfig{Layout: engine.LayoutDefault, Paginate: 5},
	}
	indexPage := &engine.Page{
		Title:        "Blog",
		Kind:         engine.KindSection,
		Collection:   col,
		RelPermalink: "/blog/",
	}
	col.IndexPage = indexPage

	rd := BuildRouteData(indexPage, baseSite(), nil)

	if rd.Paginator == nil {
		t.Fatal("Paginator should be populated for paginated list page")
	}
	if rd.Paginator.Total != 3 {
		t.Errorf("Total = %d, want 3", rd.Paginator.Total)
	}
	if rd.Paginator.Current != 1 {
		t.Errorf("Current = %d, want 1", rd.Paginator.Current)
	}
	if len(rd.Paginator.CurrentPages) != 5 {
		t.Errorf("CurrentPages len = %d, want 5", len(rd.Paginator.CurrentPages))
	}
	if !rd.Paginator.HasNext {
		t.Error("HasNext should be true on page 1 of 3")
	}
	if rd.Paginator.HasPrev {
		t.Error("HasPrev should be false on page 1")
	}
	if rd.Paginator.NextURL != "/blog/page/2/" {
		t.Errorf("NextURL = %q, want /blog/page/2/", rd.Paginator.NextURL)
	}
}

func TestBuildRouteData_PaginatorCurrentFromParams(t *testing.T) {
	pages := make([]*engine.Page, 12)
	for i := range pages {
		pages[i] = &engine.Page{Kind: engine.KindPage, RelPermalink: "/blog/post/"}
	}
	col := &engine.Collection{
		Name:   "blog",
		Pages:  pages,
		Config: &engine.CollectionConfig{Layout: engine.LayoutDefault, Paginate: 5},
	}
	indexPage := &engine.Page{
		Kind:         engine.KindSection,
		Collection:   col,
		RelPermalink: "/blog/",
		Params:       map[string]any{paginationCurrentKey: 2},
	}
	col.IndexPage = indexPage

	rd := BuildRouteData(indexPage, baseSite(), nil)

	if rd.Paginator == nil {
		t.Fatal("Paginator not set")
	}
	if rd.Paginator.Current != 2 {
		t.Errorf("Current = %d, want 2", rd.Paginator.Current)
	}
	if !rd.Paginator.HasPrev || !rd.Paginator.HasNext {
		t.Error("Page 2 of 3 should have both HasPrev and HasNext")
	}
	if rd.Paginator.PrevURL != "/blog/" {
		t.Errorf("PrevURL = %q, want /blog/", rd.Paginator.PrevURL)
	}
	if rd.Paginator.NextURL != "/blog/page/3/" {
		t.Errorf("NextURL = %q, want /blog/page/3/", rd.Paginator.NextURL)
	}
}

func TestBuildRouteData_NoPaginatorWhenPaginateZero(t *testing.T) {
	col := &engine.Collection{
		Name:   "blog",
		Pages:  []*engine.Page{{Kind: engine.KindPage}},
		Config: &engine.CollectionConfig{Layout: engine.LayoutDefault},
	}
	page := &engine.Page{Kind: engine.KindSection, Collection: col}
	rd := BuildRouteData(page, baseSite(), nil)
	if rd.Paginator != nil {
		t.Error("Paginator should be nil when Paginate == 0")
	}
}

func TestBuildRouteData_Lang(t *testing.T) {
	site := baseSite()
	site.Language = "fr"
	page := &engine.Page{Kind: engine.KindStandalone}

	rd := BuildRouteData(page, site, nil)

	if rd.Lang != "fr" {
		t.Errorf("Lang: got %q", rd.Lang)
	}
	if rd.Dir != "ltr" {
		t.Errorf("Dir: got %q", rd.Dir)
	}
}
