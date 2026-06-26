package links

import (
	"sync"
	"testing"

	"github.com/getsarde/sarde/internal/engine"
)

func TestNewLinkGraph(t *testing.T) {
	g := NewLinkGraph()
	if g == nil {
		t.Fatal("expected non-nil graph")
	}
	if g.Len() != 0 {
		t.Fatalf("expected 0 refs, got %d", g.Len())
	}
}

func TestRecord(t *testing.T) {
	g := NewLinkGraph()

	page := &engine.Page{
		PageIdentity: engine.PageIdentity{FilePath: "content/docs/guide.md"},
	}

	g.Record(LinkRef{
		FromPage: page,
		FromFile: page.FilePath,
		RawDest:  "./auth.md",
		Dim:      DimKey{Collection: "docs", Lang: "en", Version: ""},
		Kind:     KindRelative,
		Resolved: "/docs/auth/",
		Status:   StatusOK,
	})

	if g.Len() != 1 {
		t.Fatalf("expected 1 ref, got %d", g.Len())
	}

	refs := g.Refs()
	if len(refs) != 1 {
		t.Fatalf("expected 1 ref, got %d", len(refs))
	}
	if refs[0].RawDest != "./auth.md" {
		t.Errorf("expected RawDest './auth.md', got %q", refs[0].RawDest)
	}
	if refs[0].Status != StatusOK {
		t.Errorf("expected StatusOK, got %d", refs[0].Status)
	}
	if refs[0].Dim.Collection != "docs" {
		t.Errorf("expected collection 'docs', got %q", refs[0].Dim.Collection)
	}
}

func TestBrokenRefs(t *testing.T) {
	g := NewLinkGraph()

	g.Record(LinkRef{RawDest: "./ok.md", Status: StatusOK})
	g.Record(LinkRef{RawDest: "./missing.md", Status: StatusBrokenTarget})
	g.Record(LinkRef{RawDest: "./bad-anchor.md#nope", Status: StatusBrokenAnchor})
	g.Record(LinkRef{RawDest: "https://example.com", Status: StatusExternal})

	broken := g.BrokenRefs()
	if len(broken) != 2 {
		t.Fatalf("expected 2 broken refs, got %d", len(broken))
	}
	if broken[0].RawDest != "./missing.md" {
		t.Errorf("expected './missing.md', got %q", broken[0].RawDest)
	}
	if broken[1].RawDest != "./bad-anchor.md#nope" {
		t.Errorf("expected './bad-anchor.md#nope', got %q", broken[1].RawDest)
	}
}

func TestExternalRefs(t *testing.T) {
	g := NewLinkGraph()

	g.Record(LinkRef{RawDest: "./ok.md", Status: StatusOK})
	g.Record(LinkRef{RawDest: "https://example.com", Status: StatusExternal})
	g.Record(LinkRef{RawDest: "https://other.com", Status: StatusExternal})

	ext := g.ExternalRefs()
	if len(ext) != 2 {
		t.Fatalf("expected 2 external refs, got %d", len(ext))
	}
}

func TestRefsCopy(t *testing.T) {
	g := NewLinkGraph()
	g.Record(LinkRef{RawDest: "./a.md", Status: StatusOK})

	refs := g.Refs()
	refs[0].RawDest = "mutated"

	original := g.Refs()
	if original[0].RawDest != "./a.md" {
		t.Error("Refs() should return a copy; mutation should not affect original")
	}
}

func TestConcurrentRecord(t *testing.T) {
	g := NewLinkGraph()
	var wg sync.WaitGroup
	n := 100

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			g.Record(LinkRef{RawDest: "./page.md", Status: StatusOK})
		}()
	}

	wg.Wait()
	if g.Len() != n {
		t.Fatalf("expected %d refs after concurrent writes, got %d", n, g.Len())
	}
}

func TestDimKey(t *testing.T) {
	g := NewLinkGraph()

	g.Record(LinkRef{
		RawDest: "./guide.md",
		Dim:     DimKey{Collection: "docs", Lang: "en", Version: "v2"},
		Status:  StatusOK,
	})
	g.Record(LinkRef{
		RawDest: "./guide.md",
		Dim:     DimKey{Collection: "docs", Lang: "fr", Version: "v1"},
		Status:  StatusBrokenTarget,
	})

	refs := g.Refs()
	if len(refs) != 2 {
		t.Fatalf("expected 2 refs, got %d", len(refs))
	}
	if refs[0].Dim.Lang != "en" || refs[0].Dim.Version != "v2" {
		t.Error("first ref should be en/v2")
	}
	if refs[1].Dim.Lang != "fr" || refs[1].Dim.Version != "v1" {
		t.Error("second ref should be fr/v1")
	}
	if refs[1].Status != StatusBrokenTarget {
		t.Error("second ref should be BrokenTarget")
	}
}
