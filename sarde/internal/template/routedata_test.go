package template

import (
	"testing"

	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/consts"
	"github.com/getsarde/sarde/internal/engine"
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
	prev := &engine.Page{PageIdentity: engine.PageIdentity{Title: "Previous Post", RelPermalink: "/blog/prev/", Permalink: "/blog/prev/"}}
	next := &engine.Page{PageIdentity: engine.PageIdentity{Title: "Next Post", RelPermalink: "/blog/next/", Permalink: "/blog/next/"}}
	page := &engine.Page{
		PageIdentity:      engine.PageIdentity{Title: "My Post", Kind: engine.KindPage},
		PageRelationships: engine.PageRelationships{PrevPage: prev, NextPage: next, Collection: col},
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
		PageIdentity:      engine.PageIdentity{Title: "Getting Started", Kind: engine.KindPage},
		PageRelationships: engine.PageRelationships{Collection: col},
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
		PageIdentity: engine.PageIdentity{Title: "Welcome", Kind: engine.KindHome},
	}

	rd := BuildRouteData(page, baseSite(), nil)

	if rd.Template != "home" {
		t.Errorf("Template: got %q", rd.Template)
	}
	if rd.HasSidebar {
		t.Error("HasSidebar should be false for home")
	}
}

func TestBuildRouteData_HomePageHeroOptionalFields(t *testing.T) {
	page := &engine.Page{
		PageIdentity: engine.PageIdentity{Title: "Welcome", Kind: engine.KindHome},
	}
	site := baseSite()
	site.Config = &config.SiteConfig{
		Homepage: config.HomepageSettings{
			Template: "hero",
			Hero: config.HeroSettings{
				Eyebrow: "Go HTTP Router",
				Title:   "Velox",
				CTA: &config.HeroCTA{
					Label: "Get Started",
					URL:   "/docs/",
				},
				SecondaryCTA: &config.HeroCTA{
					Label: "GitHub",
					URL:   "https://github.com/example/velox",
				},
				Stats: []config.HeroStat{
					{Value: "0", Label: "heap allocations"},
					{Value: "<1us", Label: "route matching"},
				},
				Code: &config.HeroCode{
					Title:    "Quick start",
					Language: "go",
					Body:     "r := velox.New()",
				},
			},
		},
	}

	rd := BuildRouteData(page, site, nil)

	if rd.Homepage == nil {
		t.Fatal("Homepage should be populated")
	}
	hero := rd.Homepage.Hero
	if hero.Eyebrow != "Go HTTP Router" || hero.Title != "Velox" {
		t.Fatalf("Hero = %#v, want mapped eyebrow and title", hero)
	}
	if hero.CTA == nil || hero.CTA.Label != "Get Started" {
		t.Fatalf("CTA = %#v, want primary CTA", hero.CTA)
	}
	if hero.SecondaryCTA == nil || hero.SecondaryCTA.Label != "GitHub" {
		t.Fatalf("SecondaryCTA = %#v, want GitHub CTA", hero.SecondaryCTA)
	}
	if len(hero.Stats) != 2 || hero.Stats[1].Value != "<1us" {
		t.Fatalf("Stats = %#v, want mapped stats", hero.Stats)
	}
	if hero.Code == nil || hero.Code.Body != "r := velox.New()" {
		t.Fatalf("Code = %#v, want mapped code panel", hero.Code)
	}
}

func TestBuildRouteData_StandalonePage(t *testing.T) {
	page := &engine.Page{
		PageIdentity: engine.PageIdentity{Title: "About", Kind: engine.KindStandalone},
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
		PageIdentity:      engine.PageIdentity{Title: "Guides", Kind: engine.KindSection},
		PageRelationships: engine.PageRelationships{Collection: col, Section: section},
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
		PageIdentity:      engine.PageIdentity{Title: "Special Post", Kind: engine.KindPage},
		PageRelationships: engine.PageRelationships{Collection: col},
		Params:            map[string]any{"template": "blog/featured"},
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
		PageIdentity:      engine.PageIdentity{Kind: engine.KindPage},
		PageRelationships: engine.PageRelationships{Collection: col},
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
		pages[i] = &engine.Page{PageIdentity: engine.PageIdentity{Kind: engine.KindPage, Title: "post", RelPermalink: "/blog/post/"}}
	}
	col := &engine.Collection{
		Name:   "blog",
		Pages:  pages,
		Config: &engine.CollectionConfig{Layout: engine.LayoutDefault, Paginate: 5},
	}
	indexPage := &engine.Page{
		PageIdentity:      engine.PageIdentity{Title: "Blog", Kind: engine.KindSection, RelPermalink: "/blog/"},
		PageRelationships: engine.PageRelationships{Collection: col},
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
	if rd.Paginator.TotalItems != 12 {
		t.Errorf("TotalItems = %d, want 12", rd.Paginator.TotalItems)
	}
	if rd.Paginator.BaseURL != "/blog/" {
		t.Errorf("BaseURL = %q, want /blog/", rd.Paginator.BaseURL)
	}
	if rd.Paginator.FirstURL != "/blog/" {
		t.Errorf("FirstURL = %q, want /blog/", rd.Paginator.FirstURL)
	}
	if rd.Paginator.LastURL != "/blog/page/3/" {
		t.Errorf("LastURL = %q, want /blog/page/3/", rd.Paginator.LastURL)
	}
}

func TestBuildRouteData_PaginatorCurrentFromParams(t *testing.T) {
	pages := make([]*engine.Page, 12)
	for i := range pages {
		pages[i] = &engine.Page{PageIdentity: engine.PageIdentity{Kind: engine.KindPage, RelPermalink: "/blog/post/"}}
	}
	col := &engine.Collection{
		Name:   "blog",
		Pages:  pages,
		Config: &engine.CollectionConfig{Layout: engine.LayoutDefault, Paginate: 5},
	}
	indexPage := &engine.Page{
		PageIdentity:      engine.PageIdentity{Kind: engine.KindSection, RelPermalink: "/blog/"},
		PageRelationships: engine.PageRelationships{Collection: col},
		Params:            map[string]any{consts.PaginationCurrentKey: 2},
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
		Pages:  []*engine.Page{{PageIdentity: engine.PageIdentity{Kind: engine.KindPage}}},
		Config: &engine.CollectionConfig{Layout: engine.LayoutDefault},
	}
	page := &engine.Page{
		PageIdentity:      engine.PageIdentity{Kind: engine.KindSection},
		PageRelationships: engine.PageRelationships{Collection: col},
	}
	rd := BuildRouteData(page, baseSite(), nil)
	if rd.Paginator != nil {
		t.Error("Paginator should be nil when Paginate == 0")
	}
}

func TestBuildRouteData_Lang(t *testing.T) {
	site := baseSite()
	site.Language = "fr"
	page := &engine.Page{PageIdentity: engine.PageIdentity{Kind: engine.KindStandalone}}

	rd := BuildRouteData(page, site, nil)

	if rd.Lang != "fr" {
		t.Errorf("Lang: got %q", rd.Lang)
	}
	if rd.Dir != "ltr" {
		t.Errorf("Dir: got %q", rd.Dir)
	}
}

func TestBuildRouteData_TranslationsAndAllTranslations(t *testing.T) {
	site := baseSite()
	site.Languages = []engine.Language{
		{Code: "en", Name: "English", Dir: "ltr"},
		{Code: "fr", Name: "Français", Dir: "ltr"},
		{Code: "ar", Name: "العربية", Dir: "rtl"},
	}

	frReal := &engine.Page{
		PageIdentity: engine.PageIdentity{Title: "Guide FR", Permalink: "/fr/docs/guide/"},
		PageI18n:     engine.PageI18n{Lang: "fr"},
	}
	arFallback := &engine.Page{
		PageIdentity: engine.PageIdentity{Title: "Guide", Permalink: "/ar/docs/guide/"},
		PageI18n:     engine.PageI18n{Lang: "ar", IsFallback: true},
	}

	page := &engine.Page{
		PageIdentity: engine.PageIdentity{Title: "Guide", Kind: engine.KindStandalone, Permalink: "/docs/guide/"},
		PageI18n: engine.PageI18n{
			Lang:            "en",
			Translations:    []*engine.Page{frReal},
			AllTranslations: []*engine.Page{frReal, arFallback},
		},
	}

	rd := BuildRouteData(page, site, nil)

	// Translations: self + 1 real = 2
	if len(rd.Translations) != 2 {
		t.Fatalf("Translations len = %d, want 2", len(rd.Translations))
	}
	if rd.Translations[1].Lang != "fr" || rd.Translations[1].IsFallback {
		t.Errorf("Translations[1] = %+v, want fr real", rd.Translations[1])
	}

	// AllTranslations: self + 2 (real + fallback) = 3
	if len(rd.AllTranslations) != 3 {
		t.Fatalf("AllTranslations len = %d, want 3", len(rd.AllTranslations))
	}
	if rd.AllTranslations[0].Lang != "en" {
		t.Errorf("AllTranslations[0] (self) Lang = %q, want en", rd.AllTranslations[0].Lang)
	}
	if rd.AllTranslations[2].Lang != "ar" || !rd.AllTranslations[2].IsFallback {
		t.Errorf("AllTranslations[2] = %+v, want ar fallback", rd.AllTranslations[2])
	}
}

// Regression: col.Config.Versioning was read without the col.Config nil guard
// used by the other config accesses in the same block.
func TestBuildRouteData_NilCollectionConfig(t *testing.T) {
	col := &engine.Collection{Name: "docs", Title: "Docs"} // Config deliberately nil
	page := &engine.Page{
		PageIdentity:      engine.PageIdentity{Title: "P", Kind: engine.KindPage},
		PageRelationships: engine.PageRelationships{Collection: col},
	}

	rd := BuildRouteData(page, baseSite(), nil) // must not panic

	if rd.Version != "" || rd.Versions != nil {
		t.Errorf("no versioning config → version fields must stay empty, got Version=%q Versions=%v", rd.Version, rd.Versions)
	}
}

// Regression: the headerLinks block read site.Config without the site nil
// guard used by the other site accesses in the same function.
func TestBuildRouteData_NilSite(t *testing.T) {
	page := &engine.Page{
		PageIdentity: engine.PageIdentity{Title: "P", Kind: engine.KindPage},
	}

	rd := BuildRouteData(page, nil, nil) // must not panic

	if rd.GlobalNav != nil {
		t.Errorf("nil site → GlobalNav should be nil, got %+v", rd.GlobalNav)
	}
}
