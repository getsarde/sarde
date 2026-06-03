package collection

import (
	"testing"

	"github.com/frostybee/sarde/internal/engine"
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
