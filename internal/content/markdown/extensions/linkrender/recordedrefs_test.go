package linkrender

import (
	"testing"

	"github.com/getsarde/sarde/internal/content"
	"github.com/getsarde/sarde/internal/engine"
)

func TestLookupInLaneWithDefaultFallback(t *testing.T) {
	docsCol := &engine.Collection{Name: "docs", Config: &engine.CollectionConfig{}}
	enTarget := &engine.Page{
		PageIdentity:      engine.PageIdentity{RelPermalink: "/docs/guide/auth/", Permalink: "/docs/guide/auth/", Slug: "auth"},
		PageI18n:          engine.PageI18n{Lang: "en", LangRelPath: "docs/guide/auth.md"},
		PageRelationships: engine.PageRelationships{Collection: docsCol},
	}
	idx := content.BuildPageIndex([]*engine.Page{enTarget})
	resolver := &engine.URLResolver{DefaultLang: "en"}

	enPage := &engine.Page{PageI18n: engine.PageI18n{Lang: "en"}}
	if got := LookupInLaneWithDefaultFallback(idx, "/docs/guide/auth/", enPage, resolver); got != enTarget {
		t.Errorf("own-lane lookup = %v, want target page", got)
	}

	frPage := &engine.Page{PageI18n: engine.PageI18n{Lang: "fr"}}
	if got := LookupInLaneWithDefaultFallback(idx, "/docs/guide/auth/", frPage, resolver); got != enTarget {
		t.Errorf("default-lang fallback = %v, want target page", got)
	}

	if got := LookupInLaneWithDefaultFallback(idx, "/docs/guide/nope/", enPage, resolver); got != nil {
		t.Errorf("missing page = %v, want nil", got)
	}
}

func TestRecordedRefs_DrainMirrorsGraph(t *testing.T) {
	r, graph := buildSiteAbsoluteRenderer(t)

	r.ResolveHref("/docs/guide/auth")
	r.ResolveHref("https://example.com/x")

	refs := r.DrainRecordedRefs()
	graphRefs := graph.Refs()
	if len(refs) != 2 || len(graphRefs) != 2 {
		t.Fatalf("recorded %d refs, graph has %d, want 2 and 2", len(refs), len(graphRefs))
	}
	for i := range refs {
		if refs[i].RawDest != graphRefs[i].RawDest || refs[i].Status != graphRefs[i].Status || refs[i].Kind != graphRefs[i].Kind {
			t.Errorf("ref %d: buffer %+v does not mirror graph %+v", i, refs[i], graphRefs[i])
		}
	}

	if got := len(r.DrainRecordedRefs()); got != 0 {
		t.Errorf("second drain returned %d refs, want 0", got)
	}

	r.ResolveHref("/docs/guide/auth")
	r.Reset()
	if got := len(r.DrainRecordedRefs()); got != 0 {
		t.Errorf("drain after Reset returned %d refs, want 0", got)
	}
}

func TestRecordedRefs_AnchorLinksExcluded(t *testing.T) {
	r, graph := buildSiteAbsoluteRenderer(t)

	r.ResolveHref("/docs/guide/auth#api-keys")

	if got := len(r.DrainRecordedRefs()); got != 0 {
		t.Errorf("fragment link recorded %d refs, want 0 (deferred to pending anchors)", got)
	}
	if got := len(r.PendingAnchors); got != 1 {
		t.Errorf("expected 1 pending anchor, got %d", got)
	}
	if got := graph.Len(); got != 0 {
		t.Errorf("graph has %d refs, want 0", got)
	}
}
