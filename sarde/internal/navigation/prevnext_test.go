package navigation

import (
	"testing"

	"github.com/frostybee/sarde/internal/engine"
)

func TestWirePrevNextFromTree_CrossSection(t *testing.T) {
	pageA := &engine.Page{Title: "A", Slug: "a"}
	pageB := &engine.Page{Title: "B", Slug: "b"}
	pageC := &engine.Page{Title: "C", Slug: "c"}

	tree := &engine.NavTree{
		Flat: []*engine.NavNode{
			{Page: pageA, Position: 0},
			{Page: pageB, Position: 1},
			{Page: pageC, Position: 2},
		},
		TotalPages: 3,
	}

	WirePrevNextFromTree(tree)

	if pageA.PrevPage != nil {
		t.Error("A should have no prev")
	}
	if pageA.NextPage != pageB {
		t.Error("A.Next should be B")
	}
	if pageB.PrevPage != pageA {
		t.Error("B.Prev should be A")
	}
	if pageB.NextPage != pageC {
		t.Error("B.Next should be C")
	}
	if pageC.PrevPage != pageB {
		t.Error("C.Prev should be B")
	}
	if pageC.NextPage != nil {
		t.Error("C should have no next")
	}
}

func TestWirePrevNextFromTree_ManualOverride(t *testing.T) {
	pageA := &engine.Page{Title: "A", Slug: "a"}
	pageB := &engine.Page{Title: "B", Slug: "b", Params: map[string]any{"prev": "c"}}
	pageC := &engine.Page{Title: "C", Slug: "c"}

	tree := &engine.NavTree{
		Flat: []*engine.NavNode{
			{Page: pageA, Position: 0},
			{Page: pageB, Position: 1},
			{Page: pageC, Position: 2},
		},
		TotalPages: 3,
	}

	WirePrevNextFromTree(tree)

	// B's prev should be overridden to C (via frontmatter).
	if pageB.PrevPage != pageC {
		t.Errorf("B.Prev should be C (manual override), got %v", pageB.PrevPage)
	}
}

func TestWirePrevNextFromTree_NilTree(t *testing.T) {
	WirePrevNextFromTree(nil) // Should not panic.
}

func TestWirePrevNextFromTree_SinglePage(t *testing.T) {
	page := &engine.Page{Title: "Only", Slug: "only"}
	tree := &engine.NavTree{
		Flat:       []*engine.NavNode{{Page: page}},
		TotalPages: 1,
	}

	WirePrevNextFromTree(tree)

	if page.PrevPage != nil || page.NextPage != nil {
		t.Error("single page should have no prev/next")
	}
}

func TestWirePrevNextFromTree_NextOverride(t *testing.T) {
	pageA := &engine.Page{Title: "A", Slug: "a"}
	pageB := &engine.Page{Title: "B", Slug: "b", Params: map[string]any{"next": "a"}}
	pageC := &engine.Page{Title: "C", Slug: "c"}

	tree := &engine.NavTree{
		Flat: []*engine.NavNode{
			{Page: pageA, Position: 0},
			{Page: pageB, Position: 1},
			{Page: pageC, Position: 2},
		},
		TotalPages: 3,
	}

	WirePrevNextFromTree(tree)

	if pageB.NextPage != pageA {
		t.Errorf("B.Next should be A (manual override), got %v", pageB.NextPage)
	}
}

func TestWirePrevNextFromTree_BothOverrides(t *testing.T) {
	pageA := &engine.Page{Title: "A", Slug: "a"}
	pageB := &engine.Page{Title: "B", Slug: "b", Params: map[string]any{"prev": "c", "next": "a"}}
	pageC := &engine.Page{Title: "C", Slug: "c"}

	tree := &engine.NavTree{
		Flat: []*engine.NavNode{
			{Page: pageA, Position: 0},
			{Page: pageB, Position: 1},
			{Page: pageC, Position: 2},
		},
		TotalPages: 3,
	}

	WirePrevNextFromTree(tree)

	if pageB.PrevPage != pageC {
		t.Errorf("B.Prev should be C (override), got %v", pageB.PrevPage)
	}
	if pageB.NextPage != pageA {
		t.Errorf("B.Next should be A (override), got %v", pageB.NextPage)
	}
}

func TestWirePrevNextFromTree_MissingSlug(t *testing.T) {
	pageA := &engine.Page{Title: "A", Slug: "a"}
	pageB := &engine.Page{Title: "B", Slug: "b", Params: map[string]any{"prev": "nonexistent"}}

	tree := &engine.NavTree{
		Flat: []*engine.NavNode{
			{Page: pageA, Position: 0},
			{Page: pageB, Position: 1},
		},
		TotalPages: 2,
	}

	WirePrevNextFromTree(tree)

	// Non-existent slug should not override the auto-wired prev.
	if pageB.PrevPage != pageA {
		t.Errorf("B.Prev should remain A when override slug is missing, got %v", pageB.PrevPage)
	}
}
