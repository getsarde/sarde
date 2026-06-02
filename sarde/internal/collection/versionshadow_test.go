package collection

import (
	"testing"

	"github.com/frostybee/sarde/internal/engine"
)

func loosePage(lang, rel, file string) *engine.Page {
	return &engine.Page{
		PageIdentity: engine.PageIdentity{RelPermalink: rel, FilePath: file, Kind: engine.KindPage},
		PageI18n:     engine.PageI18n{Lang: lang},
	}
}

func looseSection(lang, rel, file string) *engine.Page {
	return &engine.Page{
		PageIdentity: engine.PageIdentity{RelPermalink: rel, FilePath: file, Kind: engine.KindSection},
		PageI18n:     engine.PageI18n{Lang: lang},
	}
}

func versionedPage(lang, rel, version, file string) *engine.Page {
	return &engine.Page{
		PageIdentity:   engine.PageIdentity{RelPermalink: rel, FilePath: file},
		PageI18n:       engine.PageI18n{Lang: lang},
		PageVersioning: engine.PageVersioning{Version: version},
	}
}

func TestResolveVersionShadowing(t *testing.T) {
	loose := loosePage("en", "/docs/guide/x/", "content/docs/guide/x.md")
	latest := versionedPage("en", "/docs/guide/x/", "v2", "content/docs/v2/guide/x.md")
	// v1 shares the version-free RelPermalink but is neither loose nor latest:
	// its URL keeps the /v1/ segment, so it never collides and must be kept.
	v1 := versionedPage("en", "/docs/guide/x/", "v1", "content/docs/v1/guide/x.md")
	// fr lane: loose + latest at the same RelPermalink → fr loose dropped.
	frLatest := versionedPage("fr", "/docs/guide/y/", "v2", "content/fr/docs/v2/guide/y.md")
	frLoose := loosePage("fr", "/docs/guide/y/", "content/fr/docs/guide/y.md")
	// en loose at /docs/guide/y/ where only a *fr* latest exists → kept (lang-scoped).
	enLooseVsFr := loosePage("en", "/docs/guide/y/", "content/docs/guide/y.md")
	// loose page with no latest twin (version-independent) → kept.
	standalone := loosePage("en", "/docs/changelog/", "content/docs/changelog.md")

	pages := []*engine.Page{loose, latest, v1, frLatest, frLoose, enLooseVsFr, standalone}
	kept, drops := ResolveVersionShadowing(pages, "v2")

	if len(drops) != 2 {
		t.Fatalf("got %d drops, want 2 (en loose + fr loose)", len(drops))
	}
	dropped := map[*engine.Page]*engine.Page{} // dropped → owner
	for _, d := range drops {
		dropped[d.Dropped] = d.Owner
	}
	if owner, ok := dropped[loose]; !ok || owner != latest {
		t.Errorf("en loose page should be dropped with owner=latest; got ok=%v owner=%v", ok, owner)
	}
	if owner, ok := dropped[frLoose]; !ok || owner != frLatest {
		t.Errorf("fr loose page should be dropped with owner=frLatest; got ok=%v owner=%v", ok, owner)
	}

	keptSet := map[*engine.Page]bool{}
	for _, p := range kept {
		keptSet[p] = true
	}
	for _, p := range []*engine.Page{latest, v1, frLatest, enLooseVsFr, standalone} {
		if !keptSet[p] {
			t.Errorf("page %q should be kept", p.FilePath)
		}
	}
	if keptSet[loose] || keptSet[frLoose] {
		t.Error("dropped pages must not appear in kept")
	}
	if len(kept) != 5 {
		t.Errorf("kept has %d pages, want 5", len(kept))
	}
}

func TestResolveVersionShadowing_SectionPagesKept(t *testing.T) {
	// Section pages (_index.md) are structural — they define the section tree,
	// IndexPage, and navigation. Even if a loose _index.md shadows the latest
	// version's _index.md at the same URL, the loose one must be kept.
	section := looseSection("en", "/docs/", "content/docs/_index.md")
	v2Section := versionedPage("en", "/docs/", "v2", "content/docs/v2/_index.md")
	// A leaf page at the same collection SHOULD still be dropped.
	leaf := loosePage("en", "/docs/guide/x/", "content/docs/guide/x.md")
	v2Leaf := versionedPage("en", "/docs/guide/x/", "v2", "content/docs/v2/guide/x.md")

	pages := []*engine.Page{section, v2Section, leaf, v2Leaf}
	kept, drops := ResolveVersionShadowing(pages, "v2")

	if len(drops) != 1 {
		t.Fatalf("got %d drops, want 1 (only the leaf)", len(drops))
	}
	if drops[0].Dropped != leaf {
		t.Errorf("expected the loose leaf to be dropped, got %q", drops[0].Dropped.FilePath)
	}
	keptSet := map[*engine.Page]bool{}
	for _, p := range kept {
		keptSet[p] = true
	}
	if !keptSet[section] {
		t.Error("loose section page must be kept (structural)")
	}
}

func TestResolveVersionShadowing_NoOp(t *testing.T) {
	pages := []*engine.Page{
		loosePage("en", "/docs/a/", "a.md"),
		versionedPage("en", "/docs/a/", "v2", "v2/a.md"),
	}

	// Empty lastVersion → no-op (collection not effectively versioned).
	kept, drops := ResolveVersionShadowing(pages, "")
	if len(drops) != 0 || len(kept) != len(pages) {
		t.Errorf("lastVersion=\"\" should be a no-op; got %d drops, %d kept", len(drops), len(kept))
	}

	// No pages → no-op.
	kept, drops = ResolveVersionShadowing(nil, "v2")
	if len(drops) != 0 || len(kept) != 0 {
		t.Errorf("nil pages should be a no-op; got %d drops, %d kept", len(drops), len(kept))
	}
}
