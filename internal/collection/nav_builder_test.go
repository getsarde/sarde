package collection

import (
	"path/filepath"
	"testing"

	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/content"
	"github.com/getsarde/sarde/internal/engine"
)

func TestPrimaryLang(t *testing.T) {
	tests := []struct {
		langs       []string
		defaultLang string
		want        string
	}{
		{[]string{"fr", "en"}, "en", "en"},
		{[]string{"fr", "de"}, "en", "fr"},
		{nil, "en", ""},
		{nil, "", ""},
	}
	for _, tt := range tests {
		if got := primaryLang(tt.langs, tt.defaultLang); got != tt.want {
			t.Errorf("primaryLang(%v, %q) = %q, want %q", tt.langs, tt.defaultLang, got, tt.want)
		}
	}
}

// The French page sorts first, so French is the first language in page
// order; the language-free NavTree must still be the default language's.
func TestBuildCollections_NavTreeFollowsDefaultLanguage(t *testing.T) {
	contentDir := filepath.Join(t.TempDir(), "content")
	writeTestFile(t, contentDir, filepath.Join("fr", "docs", "a.md"), "---\ntitle: A\nsidebar:\n  order: 1\n---\n")
	writeTestFile(t, contentDir, filepath.Join("docs", "b.md"), "---\ntitle: B\nsidebar:\n  order: 2\n---\n")

	docs := buildBilingual(t, contentDir, "default", false)["docs"]
	if docs.Pages[0].Lang != "fr" {
		t.Fatalf("precondition: first page language = %q, want fr", docs.Pages[0].Lang)
	}
	if docs.NavTree == nil || docs.NavTree != docs.NavTrees["en"] {
		t.Errorf("NavTree is not the English tree")
	}
}

func TestBuildCollections_VersionedNavTreeFollowsDefaultLanguage(t *testing.T) {
	contentDir := filepath.Join(t.TempDir(), "content")
	writeTestFile(t, contentDir, filepath.Join("fr", "docs", "a.md"), "---\ntitle: A\nsidebar:\n  order: 1\n---\n")
	writeTestFile(t, contentDir, filepath.Join("docs", "b.md"), "---\ntitle: B\nsidebar:\n  order: 2\n---\n")
	writeTestFile(t, contentDir, filepath.Join("docs", "v1", "b.md"), "---\ntitle: B v1\nsidebar:\n  order: 3\n---\n")

	docs := buildBilingual(t, contentDir, "default", true)["docs"]
	if docs.Pages[0].Lang != "fr" {
		t.Fatalf("precondition: first page language = %q, want fr", docs.Pages[0].Lang)
	}
	want := docs.CompositeNavTrees[LangVersionKey("en", "v2")]
	if want == nil || docs.NavTree != want {
		t.Errorf("NavTree is not the English latest-version tree")
	}
}

// A single-language versioned collection has no language codes; its
// NavTree is the latest version's tree.
func TestBuildCollections_SingleLanguageVersionedNavTree(t *testing.T) {
	contentDir := filepath.Join(t.TempDir(), "content")
	writeTestFile(t, contentDir, filepath.Join("docs", "b.md"), "---\ntitle: B\n---\n")
	writeTestFile(t, contentDir, filepath.Join("docs", "v1", "b.md"), "---\ntitle: B v1\n---\n")

	on := true
	siteCfg := config.Defaults()
	siteCfg.Collections = map[string]*config.CollectionSiteConfig{
		"docs": {Versioning: &config.VersioningConfig{Enabled: &on, LastVersion: "v2",
			Versions: []config.VersionEntry{{ID: "v2"}, {ID: "v1"}}}},
	}
	scanner := &content.Scanner{VersionIDs: map[string]map[string]bool{"docs": {"v1": true, "v2": true}}}
	files, err := scanner.DiscoverFiles(contentDir)
	if err != nil {
		t.Fatal(err)
	}
	collections, _, err := BuildCollections(files, siteCfg, contentDir, nil)
	if err != nil {
		t.Fatal(err)
	}
	docs := collections["docs"]
	want := docs.CompositeNavTrees[LangVersionKey("", "v2")]
	if want == nil || docs.NavTree != want {
		t.Errorf("NavTree is not the latest-version tree")
	}
}

func TestRebuildNavTreesWithFallbacks_NavTreeIsDefaultLanguage(t *testing.T) {
	pages := []*engine.Page{
		{PageIdentity: engine.PageIdentity{Title: "Un", Slug: "un", Kind: engine.KindPage, RelPermalink: "/docs/un/"}, PageI18n: engine.PageI18n{Lang: "fr"}},
		{PageIdentity: engine.PageIdentity{Title: "One", Slug: "one", Kind: engine.KindPage, RelPermalink: "/docs/one/"}, PageI18n: engine.PageI18n{Lang: "en"}},
	}
	col := &engine.Collection{
		Name:   "docs",
		Config: &engine.CollectionConfig{Layout: engine.LayoutDocs},
		Pages:  pages,
	}
	for _, p := range pages {
		p.Collection = col
	}

	RebuildNavTreesWithFallbacks(map[string]*engine.Collection{"docs": col}, pages, []string{"fr", "en"}, t.TempDir(), "en")

	if col.NavTree == nil || col.NavTree != col.NavTrees["en"] {
		t.Errorf("NavTree is not the English tree")
	}
}

// After fallback copies exist, an untranslated tab in another language takes
// its labels from the fallback copy of the default language's _index.md.
func TestRebuildNavTreesWithFallbacks_TabsFromFallbackCopies(t *testing.T) {
	enAPI := i18nTabPage("en", "API Reference", "/docs/api/", engine.KindSection, 2)
	pages := []*engine.Page{
		i18nTabPage("en", "User Guide", "/docs/guide/", engine.KindSection, 1),
		i18nTabPage("en", "Intro", "/docs/guide/intro/", engine.KindPage, 1),
		enAPI,
		i18nTabPage("en", "Ref", "/docs/api/ref/", engine.KindPage, 1),
		i18nTabPage("fr", "Guide de l'utilisateur", "/docs/guide/", engine.KindSection, 1),
		i18nTabPage("fr", "Intro", "/docs/guide/intro/", engine.KindPage, 1),
	}
	col := &engine.Collection{
		Name:     "docs",
		Config:   &engine.CollectionConfig{Layout: engine.LayoutDocs},
		Pages:    pages,
		Sections: BuildSectionTree(pages, "docs"),
		IsTabbed: true,
	}
	for _, p := range pages {
		p.Collection = col
	}
	col.Tabs, col.LangTabs = BuildTabsI18n(col, pages, "", []string{"en", "fr"}, "en")

	fbAPI := i18nTabPage("fr", "API Reference", "/docs/api/", engine.KindSection, 2)
	fbAPI.IsFallback = true
	fbRef := i18nTabPage("fr", "Ref", "/docs/api/ref/", engine.KindPage, 1)
	fbRef.IsFallback = true
	allPages := append(append([]*engine.Page{}, pages...), fbAPI, fbRef)
	for _, p := range allPages {
		p.Collection = col
	}

	RebuildNavTreesWithFallbacks(map[string]*engine.Collection{"docs": col}, allPages, []string{"en", "fr"}, "", "en")

	api := findTab(t, col.TabsFor("fr"), "api")
	if api.IndexPage != fbAPI {
		t.Errorf("fr api IndexPage = %v, want the fallback copy", api.IndexPage)
	}
	if got := findTab(t, col.TabsFor("fr"), "guide").Title; got != "Guide de l'utilisateur" {
		t.Errorf("fr guide title = %q, want the French title", got)
	}
	if col.Tabs[0] != col.LangTabs["en"][0] {
		t.Error("Tabs is not the English list")
	}
}
