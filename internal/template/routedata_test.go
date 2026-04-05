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
