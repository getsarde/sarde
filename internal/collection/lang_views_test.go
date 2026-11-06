package collection

import (
	"testing"
	"time"

	"github.com/getsarde/sarde/internal/engine"
)

func langViewPage(rel, lang, title string, kind engine.NodeKind, day int) *engine.Page {
	return &engine.Page{
		PageIdentity: engine.PageIdentity{
			Title: title, RelPermalink: rel, Kind: kind,
			Date: time.Date(2025, 1, day, 0, 0, 0, 0, time.UTC),
		},
		PageI18n: engine.PageI18n{Lang: lang},
		Params:   map[string]any{},
	}
}

// A blog with an index page and posts in two languages: each language gets
// its own list (fallback copies included), index page and section tree.
func TestBuildLanguageViews_PerLanguageListsAndIndex(t *testing.T) {
	enIndex := langViewPage("/blog/", "en", "Blog", engine.KindSection, 1)
	frIndex := langViewPage("/blog/", "fr", "Carnet", engine.KindSection, 1)
	enPost := langViewPage("/blog/alpha/", "en", "Alpha", engine.KindPage, 2)
	frPost := langViewPage("/blog/beta/", "fr", "Beta", engine.KindPage, 3)
	frCopy := langViewPage("/blog/alpha/", "fr", "Alpha", engine.KindPage, 2)
	frCopy.IsFallback = true
	frPost.Params["featured"] = true

	col := &engine.Collection{
		Name:   "blog",
		Config: &engine.CollectionConfig{SortBy: "date", SortOrder: "desc", Layout: engine.LayoutDefault},
		Pages:  []*engine.Page{enIndex, frIndex, enPost, frPost},
	}
	for _, p := range []*engine.Page{enIndex, frIndex, enPost, frPost, frCopy} {
		p.Collection = col
	}
	cols := map[string]*engine.Collection{"blog": col}
	all := append(append([]*engine.Page(nil), col.Pages...), frCopy)

	for run := 1; run <= 2; run++ { // the pass is repeatable
		BuildLanguageViews(cols, all, []string{"en", "fr"}, "en")

		if got := col.ListPages("en"); len(got) != 1 || got[0] != enPost {
			t.Fatalf("run %d: ListPages(en) = %v, want [Alpha]", run, titles(got))
		}
		if got := col.ListPages("fr"); len(got) != 2 || got[0] != frPost || got[1] != frCopy {
			t.Fatalf("run %d: ListPages(fr) = %v, want [Beta, Alpha copy]", run, titles(got))
		}
		if col.IndexFor("fr") != frIndex || col.TitleFor("fr") != "Carnet" || col.TitleFor("en") != "Blog" {
			t.Errorf("run %d: IndexFor/TitleFor do not follow the language", run)
		}
		if got := col.FeaturedFor("fr"); len(got) != 1 || got[0] != frPost {
			t.Errorf("run %d: FeaturedFor(fr) = %v, want [Beta]", run, titles(got))
		}
		if len(col.FeaturedFor("en")) != 0 {
			t.Errorf("run %d: FeaturedFor(en) lists a French post", run)
		}
		// Each language has its own root section, holding only its pages.
		if frIndex.Section == nil || frIndex.Section == enIndex.Section {
			t.Fatalf("run %d: the French index shares the English section", run)
		}
		if got := frIndex.Section.Pages; len(got) != 2 {
			t.Errorf("run %d: French root section holds %v, want Beta and the Alpha copy", run, titles(got))
		}
		if got := enIndex.Section.Pages; len(got) != 1 || got[0] != enPost {
			t.Errorf("run %d: English root section holds %v, want Alpha", run, titles(got))
		}
	}

	// Under fallback "omit" a language can lack an index page: no list.
	BuildLanguageViews(cols, all, []string{"en", "fr", "de"}, "en")
	if col.IndexFor("de") != nil {
		t.Error("IndexFor(de) should be nil: German has no blog index")
	}
	if len(col.ListPages("de")) != 0 {
		t.Error("ListPages(de) should be empty")
	}

	// A single language clears the views; the accessors read the shared fields.
	BuildLanguageViews(cols, all, []string{"en"}, "en")
	if col.LangPages != nil || col.LangIndex != nil || col.LangFeatured != nil {
		t.Error("a single-language build must leave the views nil")
	}
	if got := col.PagesFor("en"); len(got) != len(col.Pages) {
		t.Errorf("PagesFor without views = %d pages, want Pages (%d)", len(got), len(col.Pages))
	}
}

// In a versioned collection every version's root _index.md has the same
// RelPermalink; the latest version's is the collection's index.
func TestBuildLanguageViews_VersionedIndexIsTheLatest(t *testing.T) {
	v1 := langViewPage("/docs/", "en", "Docs v1", engine.KindSection, 1)
	v1.Version = "v1"
	v2 := langViewPage("/docs/", "en", "Docs", engine.KindSection, 1)
	v2.Version = "v2"
	col := &engine.Collection{
		Name: "docs",
		Config: &engine.CollectionConfig{
			Layout:     engine.LayoutDocs,
			Versioning: &engine.VersionConfig{Enabled: true, LastVersion: "v2"},
		},
		Pages: []*engine.Page{v1, v2},
	}
	BuildLanguageViews(map[string]*engine.Collection{"docs": col}, col.Pages, []string{"en", "fr"}, "en")
	if got := col.IndexFor("en"); got != v2 {
		t.Errorf("IndexFor(en) = %v, want the latest version's index", got)
	}
}

// Only an older version has a root _index.md: it never becomes the
// collection's index in any language.
func TestBuildLanguageViews_VersionedIndexIgnoresOlderVersions(t *testing.T) {
	v1 := langViewPage("/docs/", "en", "Docs v1", engine.KindSection, 1)
	v1.Version = "v1"
	page := langViewPage("/docs/intro/", "en", "Intro", engine.KindPage, 1)
	page.Version = "v2"
	col := &engine.Collection{
		Name:  "docs",
		Title: "Docs",
		Config: &engine.CollectionConfig{
			Layout:     engine.LayoutDocs,
			Versioning: &engine.VersionConfig{Enabled: true, LastVersion: "v2"},
		},
		Pages: []*engine.Page{v1, page},
	}
	BuildLanguageViews(map[string]*engine.Collection{"docs": col}, col.Pages, []string{"en", "fr"}, "en")
	if got := col.IndexFor("en"); got != nil {
		t.Errorf("IndexFor(en) = %q, want nil", got.Title)
	}
	if got := col.TitleFor("en"); got != "Docs" {
		t.Errorf("TitleFor(en) = %q, want Docs", got)
	}
}

func titles(pages []*engine.Page) []string {
	out := make([]string, len(pages))
	for i, p := range pages {
		out[i] = p.Title
	}
	return out
}
