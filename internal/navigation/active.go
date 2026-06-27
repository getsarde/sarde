package navigation

import (
	"github.com/getsarde/sarde/internal/engine"
)

// MarkActive deep-clones the NavTree and marks the active node and its
// ancestor chain for the current page. The clone ensures the shared
// collection-level NavTree is not mutated during parallel rendering.
func MarkActive(tree *engine.NavTree, currentPage *engine.Page) *engine.NavTree {
	if tree == nil || currentPage == nil {
		return tree
	}

	cloned := cloneNavTree(tree)
	markActiveRecursive(cloned.Root, currentPage)

	return cloned
}

// markActiveRecursive walks the tree to find and mark the active node.
// Returns true if any descendant is active.
func markActiveRecursive(node *engine.NavNode, currentPage *engine.Page) bool {
	// Check if this node is the active page.
	if node.Page != nil && node.Page.RelPermalink == currentPage.RelPermalink {
		node.IsActive = true
		return true
	}

	// Check children.
	anyActive := false
	for _, child := range node.Children {
		if markActiveRecursive(child, currentPage) {
			anyActive = true
		}
	}

	if anyActive {
		node.IsOpen = true
		node.HasActive = true
	}

	return anyActive || node.IsActive
}

// cloneNavTree creates a deep copy of a NavTree.
func cloneNavTree(tree *engine.NavTree) *engine.NavTree {
	if tree == nil {
		return nil
	}

	newRoot := cloneNode(tree.Root, nil)

	// Rebuild flat list from cloned tree.
	flat := flattenTree(newRoot)
	for i, node := range flat {
		node.Position = i
	}

	return &engine.NavTree{
		Root:       newRoot,
		Flat:       flat,
		TotalPages: tree.TotalPages,
		MaxDepth:   tree.MaxDepth,
		Hash:       tree.Hash,
	}
}

// cloneNode recursively clones a NavNode and all its children.
func cloneNode(node *engine.NavNode, parent *engine.NavNode) *engine.NavNode {
	if node == nil {
		return nil
	}

	clone := &engine.NavNode{
		Label:       node.Label,
		URL:         node.URL,
		Slug:        node.Slug,
		Order:       node.Order,
		Position:    node.Position,
		Depth:       node.Depth,
		Page:        node.Page, // share Page pointer (immutable during render)
		Parent:      parent,
		Icon:        node.Icon,
		DefaultOpen: node.DefaultOpen,
		GroupIndex:  node.GroupIndex,
	}

	if node.Attrs != nil {
		clone.Attrs = make(map[string]string, len(node.Attrs))
		for k, v := range node.Attrs {
			clone.Attrs[k] = v
		}
	}

	if len(node.Children) > 0 {
		clone.Children = make([]*engine.NavNode, len(node.Children))
		for i, child := range node.Children {
			clone.Children[i] = cloneNode(child, clone)
		}
	}

	return clone
}
