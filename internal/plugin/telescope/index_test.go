package telescope

import (
	"testing"

	"github.com/getsarde/sarde/internal/engine"
)

func page(kind engine.NodeKind, rel, title, lang string) *engine.Page {
	p := &engine.Page{}
	p.Kind = kind
	p.RelPermalink = rel
	p.Title = title
	p.Lang = lang
	return p
}

func TestBuildIndex_SkipsDraftsAndStructuralNodes(t *testing.T) {
	draft := page(engine.KindPage, "/docs/wip/", "WIP", "")
	draft.Draft = true

	pages := []*engine.Page{
		page(engine.KindPage, "/docs/intro/", "Intro", ""),
		page(engine.KindBundle, "/docs/bundle/", "Bundle", ""),
		page(engine.KindHome, "/", "Home", ""),
		page(engine.KindStandalone, "/about/", "About", ""),
		page(engine.KindSection, "/docs/", "Docs Section", ""),
		page(engine.KindTaxonomy, "/tags/", "Tags", ""),
		page(engine.KindTerm, "/tags/go/", "Go", ""),
		draft,
		nil,
	}

	entries := buildIndex(pages, nil, nil)
	if len(entries) != 4 {
		t.Fatalf("got %d entries, want 4: %+v", len(entries), entries)
	}
	for _, e := range entries {
		switch e.Path {
		case "/docs/intro/", "/docs/bundle/", "/", "/about/":
		default:
			t.Errorf("unexpected entry %q", e.Path)
		}
	}
}

func TestBuildIndex_ExcludeGlobs(t *testing.T) {
	pages := []*engine.Page{
		page(engine.KindPage, "/docs/intro/", "Intro", ""),
		page(engine.KindPage, "/internal/secret/", "Secret", ""),
	}
	entries := buildIndex(pages, []string{"/internal/*"}, nil)
	if len(entries) != 1 || entries[0].Path != "/docs/intro/" {
		t.Fatalf("exclude glob not applied: %+v", entries)
	}
}

func TestBuildIndex_DedupesByResolvedURL(t *testing.T) {
	pages := []*engine.Page{
		page(engine.KindPage, "/docs/intro/", "Intro", ""),
		page(engine.KindPage, "/docs/intro/", "Intro Copy", ""),
	}
	entries := buildIndex(pages, nil, nil)
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1 after dedupe", len(entries))
	}
}

func TestBuildIndex_TranslationsAreNotDeduped(t *testing.T) {
	// Translations share a lang-free RelPermalink; the resolver produces
	// distinct URLs, so both must survive.
	pages := []*engine.Page{
		page(engine.KindPage, "/docs/intro/", "Intro", "en"),
		page(engine.KindPage, "/docs/intro/", "Introduction", "fr"),
	}
	resolve := func(relPath, lang, _ string) string {
		if lang == "fr" {
			return "/fr" + relPath
		}
		return relPath
	}
	entries := buildIndex(pages, nil, resolve)
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2 (one per language): %+v", len(entries), entries)
	}
	if entries[0].Path != "/docs/intro/" || entries[1].Path != "/fr/docs/intro/" {
		t.Errorf("resolved paths wrong: %+v", entries)
	}
}

func TestBuildIndex_SortedAndPopulated(t *testing.T) {
	b := page(engine.KindPage, "/b/", "B", "en")
	b.Description = "second page"
	b.Tags = []string{"beta"}
	b.Collection = &engine.Collection{Name: "docs", Title: "Documentation"}
	pages := []*engine.Page{
		b,
		page(engine.KindPage, "/a/", "A", "en"),
	}
	entries := buildIndex(pages, nil, nil)
	if len(entries) != 2 || entries[0].Path != "/a/" || entries[1].Path != "/b/" {
		t.Fatalf("entries not sorted by path: %+v", entries)
	}
	if entries[1].Description != "second page" || len(entries[1].Tags) != 1 || entries[1].Lang != "en" {
		t.Errorf("entry fields not populated: %+v", entries[1])
	}
	if entries[1].Collection != "Documentation" {
		t.Errorf("Collection = %q, want collection Title", entries[1].Collection)
	}
	if entries[0].Collection != "" {
		t.Errorf("page without collection should have empty Collection, got %q", entries[0].Collection)
	}
}

func TestBuildIndex_CollectionTitleFallsBackToName(t *testing.T) {
	p := page(engine.KindPage, "/blog/post/", "Post", "en")
	p.Collection = &engine.Collection{Name: "blog"}
	entries := buildIndex([]*engine.Page{p}, nil, nil)
	if len(entries) != 1 || entries[0].Collection != "blog" {
		t.Fatalf("Collection = %+v, want fallback to Name", entries)
	}
}

func TestBuildIndex_ExcludeMatchesLaneFreePath(t *testing.T) {
	// Exclude patterns match RelPermalink before URL resolution, so one
	// pattern covers every language of a page.
	pages := []*engine.Page{
		page(engine.KindPage, "/private/", "Private EN", "en"),
		page(engine.KindPage, "/private/", "Private FR", "fr"),
	}
	resolve := func(relPath, lang, _ string) string { return "/" + lang + relPath }
	entries := buildIndex(pages, []string{"/private"}, resolve)
	if len(entries) != 0 {
		t.Fatalf("expected all translations excluded, got %+v", entries)
	}
}
