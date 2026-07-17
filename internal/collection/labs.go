package collection

import (
	"github.com/getsarde/sarde/internal/engine"
	"github.com/getsarde/sarde/internal/navigation"
)

// BuildLabsNav builds lab-scoped sidebar trees and wires prev/next
// navigation within each lab. Supports both 2-level (labs directly under
// collection root) and 3-level (courses containing labs) structures.
func BuildLabsNav(col *engine.Collection) {
	col.LabNavTrees = make(map[string]*engine.NavTree)
	root := findCollectionRootSection(col.Sections)
	if root == nil {
		return
	}

	col.IsMultiCourse = detectMultiCourse(root)

	if col.IsMultiCourse {
		for _, course := range root.Sections {
			buildLabTreesForSection(col, course)
		}
	} else {
		buildLabTreesForSection(col, root)
	}
}

// detectMultiCourse returns true if any depth-1 section has child sub-sections,
// indicating a 3-level (course > lab > step) structure.
func detectMultiCourse(root *engine.Section) bool {
	for _, sec := range root.Sections {
		if len(sec.Sections) > 0 {
			return true
		}
	}
	return false
}

// buildLabTreesForSection iterates labs under a parent section, builds a
// scoped sidebar tree for each, wires prev/next within the lab, and stamps
// progress metadata onto each page.
func buildLabTreesForSection(col *engine.Collection, parent *engine.Section) {
	for ordinal, lab := range parent.Sections {
		labTree := buildLabScopedTree(col.Name, col.Config, lab)
		if labTree == nil {
			continue
		}
		navigation.WirePrevNextFromTree(labTree)
		col.LabNavTrees[lab.Permalink] = labTree

		stampLabProgress(lab, ordinal+1, labTree, col.Config)
	}
}

// buildLabScopedTree creates a nav tree scoped to a single lab's steps.
// It reuses BuildNavTree by wrapping the lab section in a minimal Collection
// with its permalink remapped to look like the collection root, triggering
// BuildNavTree's root-hoisting logic.
func buildLabScopedTree(collName string, cfg *engine.CollectionConfig, lab *engine.Section) *engine.NavTree {
	fakeRoot := *lab
	fakeRoot.Permalink = "/" + collName + "/"
	fakeRoot.Parent = nil

	var sidebarCfg *engine.SidebarConfig
	if cfg != nil {
		sidebarCfg = cfg.Sidebar
	}

	labCol := &engine.Collection{
		Name:   collName,
		Config: &engine.CollectionConfig{Sidebar: sidebarCfg},
		Pages:  collectLabPages(lab),
		Sections: []*engine.Section{&fakeRoot},
	}
	tree := navigation.BuildNavTree(labCol)
	if tree == nil {
		return nil
	}

	if lab.IndexPage != nil {
		overview := &engine.NavNode{
			Label: "Overview",
			URL:   lab.Permalink,
			Slug:  lab.Slug,
			Depth: 1,
			Page:  lab.IndexPage,
			Order: -1,
		}
		overview.Parent = tree.Root
		tree.Root.Children = append([]*engine.NavNode{overview}, tree.Root.Children...)

		flat := make([]*engine.NavNode, 0, len(tree.Flat)+1)
		flat = append(flat, overview)
		flat = append(flat, tree.Flat...)
		for i, node := range flat {
			node.Position = i
		}
		tree.Flat = flat
		tree.TotalPages = len(flat)
	}

	return tree
}

// collectLabPages returns all non-section pages belonging to a lab section.
func collectLabPages(lab *engine.Section) []*engine.Page {
	pages := make([]*engine.Page, 0, len(lab.Pages))
	for _, p := range lab.Pages {
		if p.Kind != engine.KindSection {
			pages = append(pages, p)
		}
	}
	return pages
}

// stampLabProgress writes lab number and step progress metadata into page.Params
// for every page in the lab (intro + steps).
func stampLabProgress(lab *engine.Section, ordinal int, tree *engine.NavTree, cfg *engine.CollectionConfig) {
	total := tree.TotalPages

	stamp := func(p *engine.Page, stepIndex int) {
		if p.Params == nil {
			p.Params = make(map[string]any)
		}
		p.Params["labs_number"] = ordinal
		p.Params["labs_step_index"] = stepIndex
		p.Params["labs_step_total"] = total
	}

	for _, node := range tree.Flat {
		if node.Page != nil {
			stamp(node.Page, node.Position+1)
		}
	}
}

// findCollectionRootSection returns the root section (Parent == nil) from
// the collection's section tree. Returns nil if no root exists.
func findCollectionRootSection(sections []*engine.Section) *engine.Section {
	for _, sec := range sections {
		if sec.Parent == nil {
			return sec
		}
	}
	return nil
}
