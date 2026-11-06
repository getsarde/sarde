package collection

import (
	"strings"
	"testing"

	"github.com/getsarde/sarde/internal/engine"
)

// A top-level directory without an _index.md in an explicitly-tabbed collection
// must not crash BuildTabs (it previously dereferenced sec.IndexPage). Such a
// phantom section becomes a directory-named tab with no IndexPage, keeping its
// pages grouped under their own sidebar rather than orphaning them under tab[0].
func TestBuildTabs_PhantomTopLevelSection(t *testing.T) {
	pages := []*engine.Page{
		{PageIdentity: engine.PageIdentity{Title: "Docs", Slug: "docs", Kind: engine.KindSection, RelPermalink: "/docs/"}},
		{PageIdentity: engine.PageIdentity{Title: "Guide", Slug: "guide", Kind: engine.KindSection, RelPermalink: "/docs/guide/"}},
		{PageIdentity: engine.PageIdentity{Title: "Intro", Slug: "intro", Kind: engine.KindPage, RelPermalink: "/docs/guide/intro/"}},
		// "extra" has no _index.md → phantom top-level section.
		{PageIdentity: engine.PageIdentity{Title: "Note", Slug: "note", Kind: engine.KindPage, RelPermalink: "/docs/extra/note/"}},
	}
	col := &engine.Collection{
		Name:     "docs",
		Title:    "Docs",
		Config:   &engine.CollectionConfig{Layout: "docs"},
		Pages:    pages,
		Sections: BuildSectionTree(pages, "docs"),
	}

	tabs := BuildTabs(col, "") // must not panic on the phantom section

	var guide, extra *engine.DocsTab
	for _, tab := range tabs {
		switch tab.Slug {
		case "guide":
			guide = tab
		case "extra":
			extra = tab
		}
	}
	if guide == nil {
		t.Fatal("guide tab missing")
	}
	if guide.IndexPage == nil {
		t.Error("real guide tab should keep its IndexPage")
	}
	if extra == nil {
		t.Fatal("phantom extra tab missing")
	}
	if extra.IndexPage != nil {
		t.Error("phantom extra tab should have nil IndexPage")
	}
	if extra.Title != "Extra" {
		t.Errorf("extra.Title = %q, want %q", extra.Title, "Extra")
	}
	if len(extra.Pages) != 1 || extra.Pages[0].Slug != "note" {
		t.Errorf("extra.Pages = %v, want [note]", extra.Pages)
	}
}

// tabTestCollection builds a tabbed "docs" collection with a "guide" tab
// (index, two pages, an "advanced" subsection) and a phantom "extra" tab.
func tabTestCollection(guideIndex *engine.Page) *engine.Collection {
	pages := []*engine.Page{
		{PageIdentity: engine.PageIdentity{Title: "Docs", Slug: "docs", Kind: engine.KindSection, RelPermalink: "/docs/"}},
		guideIndex,
		{PageIdentity: engine.PageIdentity{Title: "Intro", Slug: "intro", Kind: engine.KindPage, RelPermalink: "/docs/guide/intro/"}, Sidebar: engine.PageSidebar{Order: 1}},
		{PageIdentity: engine.PageIdentity{Title: "Setup", Slug: "setup", Kind: engine.KindPage, RelPermalink: "/docs/guide/setup/"}, Sidebar: engine.PageSidebar{Order: 2}},
		{PageIdentity: engine.PageIdentity{Title: "Advanced", Slug: "advanced", Kind: engine.KindSection, RelPermalink: "/docs/guide/advanced/"}, Sidebar: engine.PageSidebar{Order: 3}},
		{PageIdentity: engine.PageIdentity{Title: "Tuning", Slug: "tuning", Kind: engine.KindPage, RelPermalink: "/docs/guide/advanced/tuning/"}},
		{PageIdentity: engine.PageIdentity{Title: "Note", Slug: "note", Kind: engine.KindPage, RelPermalink: "/docs/extra/note/"}},
	}
	return &engine.Collection{
		Name:     "docs",
		Title:    "Docs",
		Config:   &engine.CollectionConfig{Layout: "docs"},
		Pages:    pages,
		Sections: BuildSectionTree(pages, "docs"),
	}
}

func guideIndexPage() *engine.Page {
	return &engine.Page{
		PageIdentity: engine.PageIdentity{Title: "Guide", Slug: "guide", Kind: engine.KindSection, RelPermalink: "/docs/guide/"},
		Sidebar:      engine.PageSidebar{Badge: engine.Badge{Text: "Beginner"}},
	}
}

func findTab(t *testing.T, tabs []*engine.DocsTab, slug string) *engine.DocsTab {
	t.Helper()
	for _, tab := range tabs {
		if tab.Slug == slug {
			return tab
		}
	}
	t.Fatalf("tab %q missing", slug)
	return nil
}

func nodeLabels(nodes []*engine.NavNode) []string {
	labels := make([]string, len(nodes))
	for i, n := range nodes {
		labels[i] = n.Label
	}
	return labels
}

// The tab's own directory must not appear as a wrapper group: its pages sit
// at the top level, led by an Overview entry for the tab's index page.
func TestBuildTabs_TreeLeadsWithOverview(t *testing.T) {
	index := guideIndexPage()
	guide := findTab(t, BuildTabs(tabTestCollection(index), ""), "guide")
	root := guide.NavTree.Root

	got := nodeLabels(root.Children)
	want := []string{"Overview", "Intro", "Setup", "Advanced"}
	if len(got) != len(want) {
		t.Fatalf("root children = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("root children = %v, want %v", got, want)
		}
	}

	overview := root.Children[0]
	if overview.LabelKey != "nav.overview" || overview.Page != index || overview.URL != "/docs/guide/" {
		t.Errorf("overview = %+v, want nav.overview entry for the guide index", overview)
	}
	if !overview.Badge.IsEmpty() {
		t.Errorf("overview badge = %+v, want none (the tab badge is not copied)", overview.Badge)
	}
	for _, n := range root.Children {
		if n.Depth != 1 || n.Parent != root {
			t.Errorf("%s: depth %d parent %p, want depth 1 under root", n.Label, n.Depth, n.Parent)
		}
	}
	advanced := root.Children[3]
	if len(advanced.Children) != 1 || advanced.Children[0].Depth != 2 {
		t.Errorf("advanced group should keep its child at depth 2, got %+v", advanced.Children)
	}
	if guide.NavTree.Flat[0] != overview {
		t.Errorf("Flat[0] = %q, want Overview so prev/next starts there", guide.NavTree.Flat[0].Label)
	}
}

func TestBuildTabs_OverviewKeepsSidebarLabel(t *testing.T) {
	index := guideIndexPage()
	index.Sidebar.Label = "Introduction"
	guide := findTab(t, BuildTabs(tabTestCollection(index), ""), "guide")

	overview := guide.NavTree.Root.Children[0]
	if overview.Label != "Introduction" || overview.LabelKey != "" {
		t.Errorf("overview label = %q key %q, want Introduction with no key", overview.Label, overview.LabelKey)
	}
}

func TestBuildTabs_HiddenIndexHasNoOverview(t *testing.T) {
	index := guideIndexPage()
	index.Sidebar.Hidden = true
	guide := findTab(t, BuildTabs(tabTestCollection(index), ""), "guide")

	if got := nodeLabels(guide.NavTree.Root.Children); len(got) != 3 || got[0] != "Intro" {
		t.Errorf("root children = %v, want [Intro Setup Advanced]", got)
	}
}

// A phantom tab (no _index.md) is unwrapped too, but has no page to link.
func TestBuildTabs_PhantomTabHasNoOverview(t *testing.T) {
	extra := findTab(t, BuildTabs(tabTestCollection(guideIndexPage()), ""), "extra")

	if got := nodeLabels(extra.NavTree.Root.Children); len(got) != 1 || got[0] != "Note" {
		t.Errorf("root children = %v, want [Note]", got)
	}
}

// i18nTabPage builds a page of a bilingual tabbed "docs" collection.
func i18nTabPage(lang, title, rel string, kind engine.NodeKind, order int) *engine.Page {
	segs := strings.Split(strings.Trim(rel, "/"), "/")
	p := &engine.Page{
		PageIdentity: engine.PageIdentity{Title: title, Slug: segs[len(segs)-1], Kind: kind, RelPermalink: rel},
		PageI18n:     engine.PageI18n{Lang: lang},
		Sidebar:      engine.PageSidebar{Order: order},
		Params:       map[string]any{},
	}
	return p
}

// Each language's tabs carry that language's labels. The French api tab has
// no _index.md, so it borrows the English title, description, icon and
// order, and sorts after the guide as in English. Tabs that share a slug
// share one NavTrees map.
func TestBuildTabsI18n_LabelsPerLanguage(t *testing.T) {
	enGuide := i18nTabPage("en", "User Guide", "/docs/guide/", engine.KindSection, 1)
	enGuide.Params["icon"] = "book"
	frGuide := i18nTabPage("fr", "Guide de l'utilisateur", "/docs/guide/", engine.KindSection, 1)
	enAPI := i18nTabPage("en", "API Reference", "/docs/api/", engine.KindSection, 2)
	enAPI.Description = "Every endpoint"
	enAPI.Params["icon"] = "code"
	pages := []*engine.Page{
		i18nTabPage("fr", "Docs", "/docs/", engine.KindSection, 0),
		frGuide,
		i18nTabPage("fr", "Intro", "/docs/guide/intro/", engine.KindPage, 1),
		i18nTabPage("fr", "Réf", "/docs/api/ref/", engine.KindPage, 1),
		i18nTabPage("en", "Docs", "/docs/", engine.KindSection, 0),
		enGuide,
		i18nTabPage("en", "Intro", "/docs/guide/intro/", engine.KindPage, 1),
		enAPI,
		i18nTabPage("en", "Ref", "/docs/api/ref/", engine.KindPage, 1),
	}
	col := &engine.Collection{
		Name:     "docs",
		Config:   &engine.CollectionConfig{Layout: engine.LayoutDocs},
		Pages:    pages,
		Sections: BuildSectionTree(pages, "docs"),
	}

	tabs, byLang := BuildTabsI18n(col, pages, "", []string{"fr", "en"}, "en")

	if len(tabs) == 0 || len(byLang["en"]) == 0 || tabs[0] != byLang["en"][0] {
		t.Fatal("the returned tabs are not the English list")
	}
	en, fr := byLang["en"], byLang["fr"]
	if got := findTab(t, en, "guide").Title; got != "User Guide" {
		t.Errorf("en guide title = %q, want User Guide", got)
	}
	frGuideTab := findTab(t, fr, "guide")
	if frGuideTab.Title != "Guide de l'utilisateur" || frGuideTab.IndexPage != frGuide {
		t.Errorf("fr guide = %q (index %v), want the French _index.md", frGuideTab.Title, frGuideTab.IndexPage)
	}
	frAPI := findTab(t, fr, "api")
	if frAPI.Title != "API Reference" || frAPI.Description != "Every endpoint" || frAPI.Icon != "code" || frAPI.Order != 2 {
		t.Errorf("fr api = %+v, want the English labels and order", frAPI)
	}
	if fr[0].Slug != "guide" || fr[1].Slug != "api" {
		t.Errorf("fr tab order = %s, %s, want guide, api", fr[0].Slug, fr[1].Slug)
	}
	enGuideTab := findTab(t, en, "guide")
	if enGuideTab.NavTree == frGuideTab.NavTree {
		t.Error("en and fr guide tabs share one sidebar tree")
	}
	if enGuideTab.NavTrees["fr"] != frGuideTab.NavTree || frGuideTab.NavTrees["en"] != enGuideTab.NavTree {
		t.Error("same-slug tabs do not share NavTrees")
	}
	for lang, list := range byLang {
		for _, tab := range list {
			for _, p := range tab.Pages {
				if p.Lang != lang {
					t.Errorf("%s tab %s holds a %s page", lang, tab.Slug, p.Lang)
				}
			}
		}
	}
}
