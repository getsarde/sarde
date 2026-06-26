package links

import (
	"testing"

	"github.com/getsarde/sarde/internal/engine"
)

// mockAnchorIndex implements AnchorLookup for testing.
type mockAnchorIndex struct {
	headings map[string][]string // permalink → heading IDs
}

func (m *mockAnchorIndex) HasHeading(permalink, headingID string) bool {
	ids, ok := m.headings[permalink]
	if !ok {
		return false
	}
	for _, id := range ids {
		if id == headingID {
			return true
		}
	}
	return false
}

func newMockIndex(entries map[string][]string) *mockAnchorIndex {
	return &mockAnchorIndex{headings: entries}
}

func TestValidateAnchors_ValidAnchor(t *testing.T) {
	graph := NewLinkGraph()
	index := newMockIndex(map[string][]string{
		"/docs/guide/": {"_top", "setup", "usage"},
	})

	pending := []PendingAnchorCheck{
		{
			SourceFile:      "content/docs/intro.md",
			TargetPermalink: "/docs/guide/",
			Fragment:        "setup",
			RawHref:         "./guide.md#setup",
			FromPage:        &engine.Page{PageIdentity: engine.PageIdentity{FilePath: "content/docs/intro.md"}},
			Dim:             DimKey{Collection: "docs", Lang: "en"},
			Kind:            KindRelative,
			Resolved:        "/docs/guide/#setup",
		},
	}

	broken := ValidateAnchors(graph, pending, index)

	if len(broken) != 0 {
		t.Fatalf("expected 0 broken, got %d", len(broken))
	}
	if graph.Len() != 1 {
		t.Fatalf("expected 1 graph entry, got %d", graph.Len())
	}
	refs := graph.Refs()
	if refs[0].Status != StatusOK {
		t.Errorf("expected StatusOK, got %d", refs[0].Status)
	}
	if refs[0].Fragment != "setup" {
		t.Errorf("expected fragment 'setup', got %q", refs[0].Fragment)
	}
}

func TestValidateAnchors_BrokenAnchor(t *testing.T) {
	graph := NewLinkGraph()
	index := newMockIndex(map[string][]string{
		"/docs/guide/": {"_top", "setup"},
	})

	pending := []PendingAnchorCheck{
		{
			SourceFile:      "content/docs/intro.md",
			TargetPermalink: "/docs/guide/",
			Fragment:        "nonexistent",
			RawHref:         "./guide.md#nonexistent",
			FromPage:        &engine.Page{PageIdentity: engine.PageIdentity{FilePath: "content/docs/intro.md"}},
			Dim:             DimKey{Collection: "docs", Lang: "en"},
			Kind:            KindRelative,
			Resolved:        "/docs/guide/#nonexistent",
		},
	}

	broken := ValidateAnchors(graph, pending, index)

	if len(broken) != 1 {
		t.Fatalf("expected 1 broken, got %d", len(broken))
	}
	if broken[0].Fragment != "nonexistent" {
		t.Errorf("expected fragment 'nonexistent', got %q", broken[0].Fragment)
	}

	refs := graph.Refs()
	if refs[0].Status != StatusBrokenAnchor {
		t.Errorf("expected StatusBrokenAnchor, got %d", refs[0].Status)
	}
}

func TestValidateAnchors_SamePageValid(t *testing.T) {
	graph := NewLinkGraph()
	page := &engine.Page{
		PageIdentity: engine.PageIdentity{
			FilePath:  "content/docs/guide.md",
			Permalink: "/docs/guide/",
		},
	}
	index := newMockIndex(map[string][]string{
		"/docs/guide/": {"_top", "overview", "details"},
	})

	pending := []PendingAnchorCheck{
		{
			SourceFile:      page.FilePath,
			TargetPermalink: page.Permalink,
			Fragment:        "overview",
			RawHref:         "#overview",
			FromPage:        page,
			TargetPage:      page,
			Dim:             DimKey{Collection: "docs", Lang: "en"},
			Kind:            KindAnchorOnly,
			Resolved:        "/docs/guide/#overview",
		},
	}

	broken := ValidateAnchors(graph, pending, index)

	if len(broken) != 0 {
		t.Fatalf("expected 0 broken for same-page anchor, got %d", len(broken))
	}
	if graph.Refs()[0].Status != StatusOK {
		t.Error("expected StatusOK for same-page anchor")
	}
}

func TestValidateAnchors_SamePageBroken(t *testing.T) {
	graph := NewLinkGraph()
	page := &engine.Page{
		PageIdentity: engine.PageIdentity{
			FilePath:  "content/docs/guide.md",
			Permalink: "/docs/guide/",
		},
	}
	index := newMockIndex(map[string][]string{
		"/docs/guide/": {"_top", "overview"},
	})

	pending := []PendingAnchorCheck{
		{
			SourceFile:      page.FilePath,
			TargetPermalink: page.Permalink,
			Fragment:        "missing-section",
			RawHref:         "#missing-section",
			FromPage:        page,
			TargetPage:      page,
			Dim:             DimKey{Collection: "docs", Lang: "en"},
			Kind:            KindAnchorOnly,
			Resolved:        "/docs/guide/#missing-section",
		},
	}

	broken := ValidateAnchors(graph, pending, index)

	if len(broken) != 1 {
		t.Fatalf("expected 1 broken for same-page broken anchor, got %d", len(broken))
	}
	if graph.Refs()[0].Status != StatusBrokenAnchor {
		t.Error("expected StatusBrokenAnchor for same-page broken anchor")
	}
}

func TestValidateAnchors_TopSentinel(t *testing.T) {
	graph := NewLinkGraph()
	index := newMockIndex(map[string][]string{
		"/docs/guide/": {"_top", "setup"},
	})

	pending := []PendingAnchorCheck{
		{
			SourceFile:      "content/docs/intro.md",
			TargetPermalink: "/docs/guide/",
			Fragment:        "_top",
			RawHref:         "./guide.md#_top",
			FromPage:        &engine.Page{},
			Dim:             DimKey{Collection: "docs", Lang: "en"},
		},
	}

	broken := ValidateAnchors(graph, pending, index)

	if len(broken) != 0 {
		t.Fatal("_top should always be valid")
	}
	if graph.Refs()[0].Status != StatusOK {
		t.Error("expected StatusOK for _top anchor")
	}
}

func TestValidateAnchors_DuplicateSuffix(t *testing.T) {
	graph := NewLinkGraph()
	index := newMockIndex(map[string][]string{
		"/docs/guide/": {"_top", "overview", "overview-1", "overview-2"},
	})

	pending := []PendingAnchorCheck{
		{
			SourceFile:      "content/docs/intro.md",
			TargetPermalink: "/docs/guide/",
			Fragment:        "overview-1",
			RawHref:         "./guide.md#overview-1",
			FromPage:        &engine.Page{},
			Dim:             DimKey{Collection: "docs", Lang: "en"},
		},
	}

	broken := ValidateAnchors(graph, pending, index)

	if len(broken) != 0 {
		t.Fatal("overview-1 should be valid (duplicate suffix)")
	}
}

func TestValidateAnchors_MixedResults(t *testing.T) {
	graph := NewLinkGraph()
	index := newMockIndex(map[string][]string{
		"/docs/guide/": {"_top", "setup", "usage"},
		"/docs/api/":   {"_top", "endpoints"},
	})

	pending := []PendingAnchorCheck{
		{
			SourceFile: "a.md", TargetPermalink: "/docs/guide/",
			Fragment: "setup", RawHref: "./guide.md#setup",
			FromPage: &engine.Page{}, Dim: DimKey{Collection: "docs", Lang: "en"},
		},
		{
			SourceFile: "a.md", TargetPermalink: "/docs/guide/",
			Fragment: "missing", RawHref: "./guide.md#missing",
			FromPage: &engine.Page{}, Dim: DimKey{Collection: "docs", Lang: "en"},
		},
		{
			SourceFile: "b.md", TargetPermalink: "/docs/api/",
			Fragment: "endpoints", RawHref: "./api.md#endpoints",
			FromPage: &engine.Page{}, Dim: DimKey{Collection: "docs", Lang: "en"},
		},
		{
			SourceFile: "b.md", TargetPermalink: "/docs/api/",
			Fragment: "auth", RawHref: "./api.md#auth",
			FromPage: &engine.Page{}, Dim: DimKey{Collection: "docs", Lang: "en"},
		},
	}

	broken := ValidateAnchors(graph, pending, index)

	if len(broken) != 2 {
		t.Fatalf("expected 2 broken, got %d", len(broken))
	}
	if broken[0].Fragment != "missing" {
		t.Errorf("first broken should be 'missing', got %q", broken[0].Fragment)
	}
	if broken[1].Fragment != "auth" {
		t.Errorf("second broken should be 'auth', got %q", broken[1].Fragment)
	}

	if graph.Len() != 4 {
		t.Fatalf("expected 4 graph entries, got %d", graph.Len())
	}

	brokenRefs := graph.BrokenRefs()
	if len(brokenRefs) != 2 {
		t.Fatalf("expected 2 BrokenRefs, got %d", len(brokenRefs))
	}
}

func TestValidateAnchors_EmptyPending(t *testing.T) {
	graph := NewLinkGraph()
	index := newMockIndex(nil)

	broken := ValidateAnchors(graph, nil, index)

	if len(broken) != 0 {
		t.Fatalf("expected 0 broken for empty pending, got %d", len(broken))
	}
	if graph.Len() != 0 {
		t.Fatalf("expected 0 graph entries, got %d", graph.Len())
	}
}
