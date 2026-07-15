package navigation

import (
	"fmt"
	"sort"
	"strings"

	"github.com/getsarde/sarde/internal/content"
	"github.com/getsarde/sarde/internal/engine"
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

	ctx := sidebarCtx{collName: collection.Name}
	if collection.Config != nil {
		ctx.sidebar = collection.Config.Sidebar
	}

	// Add root-level pages (pages not in any section).
	rootPages := findRootPages(collection)
	for _, page := range rootPages {
		if page.Sidebar.Hidden {
			continue
		}
		node := pageToNode(page, 1, ctx)
		if node == nil {
			continue
		}
		node.Parent = root
		root.Children = append(root.Children, node)
	}

	// Add sections as group nodes.
	collectionRootPermalink := "/" + collection.Name + "/"
	for _, sec := range collection.Sections {
		isCollectionRoot := sec.Permalink == collectionRootPermalink
		if sec.Transparent || isCollectionRoot {
			// Hoist transparent/root section's children directly to root.
			addSectionChildren(root, sec, 1, maxDepth, ctx)
		} else {
			node := buildNodeFromSection(sec, 1, maxDepth, ctx)
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

	assignGroupIndices(root)
	hash := computeSidebarHash(root)

	return &engine.NavTree{
		Root:       root,
		Flat:       flat,
		TotalPages: len(flat),
		MaxDepth:   md,
		Hash:       hash,
	}
}

// buildNodeFromSection recursively converts a Section into a NavNode group.
func buildNodeFromSection(sec *engine.Section, depth int, maxDepth int, ctx sidebarCtx) *engine.NavNode {
	if depth > maxDepth {
		return nil
	}

	ov := lookupOverride(ctx, collectionRelPath(sec.Permalink, ctx.collName))
	if ov != nil && ov.Hidden {
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
		group.Order = sec.IndexPage.Sidebar.Order
		if sec.IndexPage.Sidebar.Label != "" {
			group.Label = sec.IndexPage.Sidebar.Label
		}
	} else if sec.IndexPage != nil {
		// Non-rendering but has index: use its weight/label.
		group.Order = sec.IndexPage.Sidebar.Order
		if sec.IndexPage.Sidebar.Label != "" {
			group.Label = sec.IndexPage.Sidebar.Label
		}
	}

	// Copy sidebar attrs, icon, and DefaultOpen from section index page.
	if sec.IndexPage != nil {
		group.Attrs = cloneStringMap(sec.IndexPage.Sidebar.Attrs)
		group.Icon = sec.IndexPage.Sidebar.Icon
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
		if page.Sidebar.Hidden || page.Kind == engine.KindSection {
			continue
		}
		node := pageToNode(page, depth+1, ctx)
		if node == nil {
			continue
		}
		node.Parent = group
		group.Children = append(group.Children, node)
	}

	// Add child sections.
	for _, child := range sec.Sections {
		if child.Transparent {
			addSectionChildren(group, child, depth+1, maxDepth, ctx)
		} else {
			childNode := buildNodeFromSection(child, depth+1, maxDepth, ctx)
			if childNode != nil {
				childNode.Parent = group
				group.Children = append(group.Children, childNode)
			}
		}
	}

	// Depth-based default expansion (collapse_level).
	if ctx.sidebar != nil && ctx.sidebar.CollapseLevel > 0 && depth <= ctx.sidebar.CollapseLevel {
		group.DefaultOpen = true
	}

	// sidebar.yaml override wins over frontmatter and collapse_level.
	applyOverride(group, ov)

	return group
}

// addSectionChildren adds a transparent section's pages and sub-sections
// directly to the parent node (hoisting).
func addSectionChildren(parent *engine.NavNode, sec *engine.Section, depth int, maxDepth int, ctx sidebarCtx) {
	if ov := lookupOverride(ctx, collectionRelPath(sec.Permalink, ctx.collName)); ov != nil && ov.Hidden {
		return
	}
	for _, page := range sec.Pages {
		if page.Sidebar.Hidden || page.Kind == engine.KindSection {
			continue
		}
		node := pageToNode(page, depth, ctx)
		if node == nil {
			continue
		}
		node.Parent = parent
		parent.Children = append(parent.Children, node)
	}
	for _, child := range sec.Sections {
		if child.Transparent {
			addSectionChildren(parent, child, depth, maxDepth, ctx)
		} else {
			childNode := buildNodeFromSection(child, depth, maxDepth, ctx)
			if childNode != nil {
				childNode.Parent = parent
				parent.Children = append(parent.Children, childNode)
			}
		}
	}
}

// pageToNode creates a leaf NavNode from a Page.
func pageToNode(page *engine.Page, depth int, ctx sidebarCtx) *engine.NavNode {
	ov := lookupOverride(ctx, collectionRelPath(page.RelPermalink, ctx.collName))
	if ov != nil && ov.Hidden {
		return nil
	}
	label := page.Title
	if page.Sidebar.Label != "" {
		label = page.Sidebar.Label
	}
	node := &engine.NavNode{
		Label: label,
		URL:   page.RelPermalink,
		Slug:  page.Slug,
		Order: page.Sidebar.Order,
		Depth: depth,
		Page:  page,
		Attrs: cloneStringMap(page.Sidebar.Attrs),
		Icon:  page.Sidebar.Icon,
		Badge: page.Sidebar.Badge,
	}
	applyOverride(node, ov)
	return node
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
		if a.Order != b.Order {
			return a.Order < b.Order
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

// cloneStringMap returns a shallow copy of a string map, or nil if empty.
func cloneStringMap(m map[string]string) map[string]string {
	if len(m) == 0 {
		return nil
	}
	result := make(map[string]string, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

// assignGroupIndices assigns sequential DFS indexes to group nodes (nodes
// with children). The order matches DOM rendering order in Sidebar.html.
func assignGroupIndices(root *engine.NavNode) {
	counter := 0
	var walk func(*engine.NavNode)
	walk = func(node *engine.NavNode) {
		for _, child := range node.Children {
			if len(child.Children) > 0 {
				child.GroupIndex = counter
				counter++
				walk(child)
			}
		}
	}
	walk(root)
}

// ---------------------------------------------------------------------------
// sidebar.yaml overrides
// ---------------------------------------------------------------------------

// sidebarCtx carries the collection name and sidebar config through tree
// building so sidebar.yaml overrides can be consulted per node. Threaded as a
// parameter (not package state) so future structural sidebar building can
// reuse the same helpers scoped to a subtree.
type sidebarCtx struct {
	collName string
	sidebar  *engine.SidebarConfig // may be nil
}

// collectionRelPath converts a permalink to the collection-relative path key
// used by sidebar.yaml overrides ("/docs/guide/advanced/" -> "guide/advanced").
// Permalinks are version-free and lang-free by construction, so the same key
// matches in every (lang, version) lane. Mirrors buildPageLookup in navyaml.go.
func collectionRelPath(permalink, collectionName string) string {
	p := strings.TrimPrefix(permalink, "/"+collectionName+"/")
	return strings.TrimSuffix(p, "/")
}

// lookupOverride returns the sidebar.yaml override for key, marking the key
// matched so unmatched keys can be reported once all lanes have built.
func lookupOverride(ctx sidebarCtx, key string) *engine.SidebarOverride {
	if ctx.sidebar == nil || len(ctx.sidebar.Overrides) == 0 || key == "" {
		return nil
	}
	ov, ok := ctx.sidebar.Overrides[key]
	if !ok {
		return nil
	}
	ctx.sidebar.MarkOverrideMatched(key)
	return ov
}

// applyOverride applies sidebar.yaml node properties on top of the values
// derived from frontmatter and inference. Config wins; unset falls through.
func applyOverride(node *engine.NavNode, ov *engine.SidebarOverride) {
	if ov == nil {
		return
	}
	if ov.Label != "" {
		node.Label = ov.Label
	}
	if ov.Description != "" {
		node.Description = ov.Description
	}
	if ov.Order != nil {
		node.Order = *ov.Order
	}
	if ov.Icon != "" {
		node.Icon = ov.Icon
	}
	if !ov.Badge.IsEmpty() {
		node.Badge = ov.Badge
	}
	// collapsed: false forces a group open; collapsed: true clears DefaultOpen,
	// which only closes the group when the collection collapses groups by
	// default (collapsed_by_default or collapse_level). It cannot force-close
	// against a blanket-open sidebar, matching nav.yaml semantics.
	if ov.Collapsed != nil {
		node.DefaultOpen = !*ov.Collapsed
	}
	if len(ov.Attrs) > 0 {
		if node.Attrs == nil {
			node.Attrs = make(map[string]string, len(ov.Attrs))
		}
		for k, v := range ov.Attrs {
			node.Attrs[k] = v
		}
	}
}

// computeSidebarHash produces a DJB2 hash of the sidebar's group structure
// (labels and child counts). Used by client JS to invalidate stale state.
func computeSidebarHash(root *engine.NavNode) string {
	h := uint32(5381)
	var walk func(*engine.NavNode)
	walk = func(node *engine.NavNode) {
		for _, child := range node.Children {
			if len(child.Children) > 0 {
				for _, b := range []byte(child.Label) {
					h = h*33 ^ uint32(b)
				}
				h = h*33 ^ uint32(len(child.Children))
				walk(child)
			}
		}
	}
	walk(root)
	return fmt.Sprintf("%08x", h)
}
