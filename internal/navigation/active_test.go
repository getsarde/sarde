package navigation

import (
	"testing"

	"github.com/coderoo-dev/coderoo/internal/engine"
)

func buildTestTree() *engine.NavTree {
	pageA := &engine.Page{Title: "Page A", RelPermalink: "/docs/a/"}
	pageB := &engine.Page{Title: "Page B", RelPermalink: "/docs/guides/b/"}
	pageC := &engine.Page{Title: "Page C", RelPermalink: "/docs/guides/c/"}

	root := &engine.NavNode{Label: "Root", Depth: 0}
	nodeA := &engine.NavNode{Label: "Page A", URL: "/docs/a/", Page: pageA, Depth: 1, Parent: root}
	group := &engine.NavNode{Label: "Guides", Depth: 1, Parent: root}
	nodeB := &engine.NavNode{Label: "Page B", URL: "/docs/guides/b/", Page: pageB, Depth: 2, Parent: group}
	nodeC := &engine.NavNode{Label: "Page C", URL: "/docs/guides/c/", Page: pageC, Depth: 2, Parent: group}

	group.Children = []*engine.NavNode{nodeB, nodeC}
	root.Children = []*engine.NavNode{nodeA, group}

	flat := []*engine.NavNode{nodeA, nodeB, nodeC}
	for i, n := range flat {
		n.Position = i
	}

	return &engine.NavTree{Root: root, Flat: flat, TotalPages: 3, MaxDepth: 2}
}

func TestMarkActive_LeafNode(t *testing.T) {
	tree := buildTestTree()
	page := &engine.Page{RelPermalink: "/docs/a/"}

	marked := MarkActive(tree, page)

	// Original should be unmodified.
	if tree.Root.Children[0].IsActive {
		t.Error("original tree was mutated")
	}

	// Cloned tree should have active node.
	nodeA := marked.Root.Children[0]
	if !nodeA.IsActive {
		t.Error("expected Page A to be active")
	}
}

func TestMarkActive_NestedNode(t *testing.T) {
	tree := buildTestTree()
	page := &engine.Page{RelPermalink: "/docs/guides/b/"}

	marked := MarkActive(tree, page)

	group := marked.Root.Children[1]
	if !group.IsOpen {
		t.Error("expected Guides group to be open")
	}
	if !group.HasActive {
		t.Error("expected Guides group HasActive")
	}

	nodeB := group.Children[0]
	if !nodeB.IsActive {
		t.Error("expected Page B to be active")
	}

	nodeC := group.Children[1]
	if nodeC.IsActive {
		t.Error("expected Page C to not be active")
	}
}

func TestMarkActive_NilTree(t *testing.T) {
	result := MarkActive(nil, &engine.Page{})
	if result != nil {
		t.Error("expected nil")
	}
}

func TestMarkActive_NilPage(t *testing.T) {
	tree := buildTestTree()
	result := MarkActive(tree, nil)
	// Should return original tree unchanged.
	if result != tree {
		t.Error("expected same tree returned for nil page")
	}
}

func TestMarkActive_CloneIndependence(t *testing.T) {
	tree := buildTestTree()

	m1 := MarkActive(tree, &engine.Page{RelPermalink: "/docs/a/"})
	m2 := MarkActive(tree, &engine.Page{RelPermalink: "/docs/guides/b/"})

	// m1 and m2 should be independent.
	if m1.Root.Children[0].IsActive != true {
		t.Error("m1: expected Page A active")
	}
	if m2.Root.Children[0].IsActive != false {
		t.Error("m2: Page A should not be active")
	}
	if m2.Root.Children[1].IsOpen != true {
		t.Error("m2: Guides should be open")
	}
}
