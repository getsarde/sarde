package navigation

import (
	"sort"
	"strings"

	"github.com/frostybee/sarde/internal/content"
	"github.com/frostybee/sarde/internal/engine"
)

// BuildNavTree constructs a NavTree from a collection's section tree.
// For docs-layout collections, this produces the sidebar navigation.
// The tree respects transparent sections (hoisted), non-rendering sections
// (group label only), hidden pages, and weight-based sorting.
func BuildNavTree(collection *engine.Collection) *engine.NavTree {
	if collection == nil {
		return nil
	}

	maxDepth := 4
	if collection.Config != nil && collection.Config.Sidebar != nil && collection.Config.Sidebar.MaxDepth > 0 {
		maxDepth = collection.Config.Sidebar.MaxDepth
	}

	root := &engine.NavNode{
		Label: collection.Title,
		Depth: 0,
	}

	// Add root-level pages (pages not in any section).
	rootPages := findRootPages(collection)
	for _, page := range rootPages {
		if page.SidebarHidden {
			continue
		}
		node := pageToNode(page, 1)
		node.Parent = root
		root.Children = append(root.Children, node)
	}

	// Add sections as group nodes.
	for _, sec := range collection.Sections {
		if sec.Transparent {
			// Hoist transparent section's children directly to root.
			addSectionChildren(root, sec, 1, maxDepth)
		} else {
			node := buildNodeFromSection(sec, 1, maxDepth)
			if node != nil {
				node.Parent = root
				root.Children = append(root.Children, node)
			}
		}
	}

	// Sort all levels.
	sortNodesRecursive(root)

	// Flatten to ordered list (DFS, leaf pages only).
	flat := flattenTree(root)
	for i, node := range flat {
		node.Position = i
	}

	// Compute max depth.
	md := computeMaxDepth(root)

	return &engine.NavTree{
		Root:       root,
		Flat:       flat,
		TotalPages: len(flat),
		MaxDepth:   md,
	}
}

// buildNodeFromSection recursively converts a Section into a NavNode group.
func buildNodeFromSection(sec *engine.Section, depth int, maxDepth int) *engine.NavNode {
	if depth > maxDepth {
		return nil
	}

	group := &engine.NavNode{
		Label: sec.Title,
		Slug:  sec.Slug,
		Depth: depth,
	}

	// Rendering sections with an index page get a URL.
	if sec.Render && sec.IndexPage != nil {
		group.URL = sec.Permalink
		group.Page = sec.IndexPage
		group.Weight = sec.IndexPage.Weight
		if sec.IndexPage.SidebarLabel != "" {
			group.Label = sec.IndexPage.SidebarLabel
		}
	} else if sec.IndexPage != nil {
		// Non-rendering but has index: use its weight/label.
		group.Weight = sec.IndexPage.Weight
		if sec.IndexPage.SidebarLabel != "" {
			group.Label = sec.IndexPage.SidebarLabel
		}
	}

	// Copy sidebar_attrs and DefaultOpen from section index page.
	if sec.IndexPage != nil {
		group.Attrs = copyAttrs(sec.IndexPage.Params)
		if group.Attrs != nil && group.Attrs["open"] == "true" {
			group.DefaultOpen = true
		}
	}

	// Fallback label from directory name.
	if group.Label == "" {
		group.Label = content.FilenameToTitle(sec.Slug + ".md")
	}

	// Add child pages.
	for _, page := range sec.Pages {
		if page.SidebarHidden || page.Kind == engine.KindSection {
			continue
		}
		node := pageToNode(page, depth+1)
		node.Parent = group
		group.Children = append(group.Children, node)
	}

	// Add child sections.
	for _, child := range sec.Sections {
		if child.Transparent {
			addSectionChildren(group, child, depth+1, maxDepth)
		} else {
			childNode := buildNodeFromSection(child, depth+1, maxDepth)
			if childNode != nil {
				childNode.Parent = group
				group.Children = append(group.Children, childNode)
			}
		}
	}

	return group
}

// addSectionChildren adds a transparent section's pages and sub-sections
// directly to the parent node (hoisting).
func addSectionChildren(parent *engine.NavNode, sec *engine.Section, depth int, maxDepth int) {
	for _, page := range sec.Pages {
		if page.SidebarHidden || page.Kind == engine.KindSection {
			continue
		}
		node := pageToNode(page, depth)
		node.Parent = parent
		parent.Children = append(parent.Children, node)
	}
	for _, child := range sec.Sections {
		if child.Transparent {
			addSectionChildren(parent, child, depth, maxDepth)
		} else {
			childNode := buildNodeFromSection(child, depth, maxDepth)
			if childNode != nil {
				childNode.Parent = parent
				parent.Children = append(parent.Children, childNode)
			}
		}
	}
}

// pageToNode creates a leaf NavNode from a Page.
func pageToNode(page *engine.Page, depth int) *engine.NavNode {
	label := page.Title
	if page.SidebarLabel != "" {
		label = page.SidebarLabel
	}
	return &engine.NavNode{
		Label:  label,
		URL:    page.RelPermalink,
		Slug:   page.Slug,
		Weight: page.Weight,
		Depth:  depth,
		Page:   page,
		Attrs:  copyAttrs(page.Params),
	}
}

// findRootPages returns pages that belong to the collection but aren't
// in any section (root-level content files).
func findRootPages(col *engine.Collection) []*engine.Page {
	var pages []*engine.Page
	for _, p := range col.Pages {
		if p.Section == nil && p.Kind == engine.KindPage {
			pages = append(pages, p)
		}
	}
	return pages
}

// sortNodesRecursive sorts children at each level by weight then title.
func sortNodesRecursive(node *engine.NavNode) {
	if len(node.Children) == 0 {
		return
	}
	sort.SliceStable(node.Children, func(i, j int) bool {
		a, b := node.Children[i], node.Children[j]
		if a.Weight != b.Weight {
			return a.Weight < b.Weight
		}
		return strings.ToLower(a.Label) < strings.ToLower(b.Label)
	})
	for _, child := range node.Children {
		sortNodesRecursive(child)
	}
}

// flattenTree returns a DFS-ordered list of leaf nodes (pages only).
func flattenTree(root *engine.NavNode) []*engine.NavNode {
	var flat []*engine.NavNode
	flattenDFS(root, &flat)
	return flat
}

func flattenDFS(node *engine.NavNode, flat *[]*engine.NavNode) {
	if len(node.Children) == 0 && node.Page != nil {
		*flat = append(*flat, node)
		return
	}
	// Also include group nodes that are clickable (have a Page reference).
	if node.Page != nil && len(node.Children) > 0 {
		*flat = append(*flat, node)
	}
	for _, child := range node.Children {
		flattenDFS(child, flat)
	}
}

// computeMaxDepth returns the maximum depth in the tree.
func computeMaxDepth(root *engine.NavNode) int {
	max := root.Depth
	for _, child := range root.Children {
		d := computeMaxDepth(child)
		if d > max {
			max = d
		}
	}
	return max
}

// copyAttrs extracts sidebar_attrs from page Params if present.
func copyAttrs(params map[string]any) map[string]string {
	if params == nil {
		return nil
	}
	raw, ok := params["sidebar_attrs"]
	if !ok {
		return nil
	}
	m, ok := raw.(map[string]string)
	if ok {
		result := make(map[string]string, len(m))
		for k, v := range m {
			result[k] = v
		}
		return result
	}
	// Handle map[string]any from YAML parsing.
	ma, ok := raw.(map[string]any)
	if ok {
		result := make(map[string]string, len(ma))
		for k, v := range ma {
			if s, ok := v.(string); ok {
				result[k] = s
			}
		}
		return result
	}
	return nil
}
