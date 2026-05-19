package navigation

import (
	"testing"

	"github.com/coderoo-dev/coderoo/internal/engine"
)

func docsConfig() *engine.CollectionConfig {
	return &engine.CollectionConfig{
		Layout: engine.LayoutDocs,
		Sidebar: &engine.SidebarConfig{
			MaxDepth: 4,
		},
	}
}

func TestBuildNavTree_BasicStructure(t *testing.T) {
	sec := &engine.Section{
		Title:     "Guides",
		Slug:      "guides",
		Permalink: "/docs/guides/",
		Render:    true,
		IndexPage: &engine.Page{Title: "Guides", Weight: 1, Kind: engine.KindSection},
		Pages: []*engine.Page{
			{Title: "Auth", Slug: "auth", RelPermalink: "/docs/guides/auth/", Weight: 1, Kind: engine.KindPage},
			{Title: "Deploy", Slug: "deploy", RelPermalink: "/docs/guides/deploy/", Weight: 2, Kind: engine.KindPage},
		},
	}
	// Wire Section backrefs.
	for _, p := range sec.Pages {
		p.Section = sec
	}
	sec.IndexPage.Section = sec

	col := &engine.Collection{
		Name:     "docs",
		Title:    "Documentation",
		Config:   docsConfig(),
		Sections: []*engine.Section{sec},
		Pages: []*engine.Page{
			{Title: "Getting Started", Slug: "getting-started", RelPermalink: "/docs/getting-started/", Weight: 0, Kind: engine.KindPage},
			sec.IndexPage,
			sec.Pages[0],
			sec.Pages[1],
		},
	}

	tree := BuildNavTree(col)
	if tree == nil {
		t.Fatal("expected non-nil tree")
	}

	root := tree.Root
	if root.Label != "Documentation" {
		t.Errorf("root label: got %q", root.Label)
	}

	// Should have: Getting Started (root page) + Guides (section group)
	if len(root.Children) != 2 {
		t.Fatalf("expected 2 root children, got %d", len(root.Children))
	}

	// First child should be Getting Started (weight 0)
	if root.Children[0].Label != "Getting Started" {
		t.Errorf("first child: got %q", root.Children[0].Label)
	}

	// Second child should be Guides group
	guides := root.Children[1]
	if guides.Label != "Guides" {
		t.Errorf("guides label: got %q", guides.Label)
	}
	if len(guides.Children) != 2 {
		t.Fatalf("expected 2 guide children, got %d", len(guides.Children))
	}
	if guides.Children[0].Label != "Auth" {
		t.Errorf("first guide child: got %q", guides.Children[0].Label)
	}
}

func TestBuildNavTree_TransparentSection(t *testing.T) {
	sec := &engine.Section{
		Title:       "Internal",
		Slug:        "internal",
		Transparent: true,
		Pages: []*engine.Page{
			{Title: "Page A", Slug: "a", RelPermalink: "/docs/a/", Weight: 1, Kind: engine.KindPage},
			{Title: "Page B", Slug: "b", RelPermalink: "/docs/b/", Weight: 2, Kind: engine.KindPage},
		},
	}
	for _, p := range sec.Pages {
		p.Section = sec
	}

	col := &engine.Collection{
		Name:     "docs",
		Title:    "Docs",
		Config:   docsConfig(),
		Sections: []*engine.Section{sec},
		Pages:    sec.Pages,
	}

	tree := BuildNavTree(col)
	root := tree.Root

	// Transparent section's pages should be hoisted to root.
	if len(root.Children) != 2 {
		t.Fatalf("expected 2 root children (hoisted), got %d", len(root.Children))
	}
	if root.Children[0].Label != "Page A" {
		t.Errorf("got %q", root.Children[0].Label)
	}
}

func TestBuildNavTree_NonRenderingSection(t *testing.T) {
	sec := &engine.Section{
		Title:     "Advanced",
		Slug:      "advanced",
		Permalink: "/docs/advanced/",
		Render:    false,
		IndexPage: &engine.Page{Title: "Advanced", Weight: 10, Kind: engine.KindSection},
		Pages: []*engine.Page{
			{Title: "Internals", Slug: "internals", RelPermalink: "/docs/advanced/internals/", Kind: engine.KindPage},
		},
	}
	for _, p := range sec.Pages {
		p.Section = sec
	}
	sec.IndexPage.Section = sec

	col := &engine.Collection{
		Name:     "docs",
		Title:    "Docs",
		Config:   docsConfig(),
		Sections: []*engine.Section{sec},
		Pages:    append([]*engine.Page{sec.IndexPage}, sec.Pages...),
	}

	tree := BuildNavTree(col)
	group := tree.Root.Children[0]

	// Non-rendering group should have empty URL.
	if group.URL != "" {
		t.Errorf("expected empty URL for non-rendering section, got %q", group.URL)
	}
	if group.Label != "Advanced" {
		t.Errorf("label: got %q", group.Label)
	}
}

func TestBuildNavTree_SidebarHidden(t *testing.T) {
	col := &engine.Collection{
		Name:   "docs",
		Title:  "Docs",
		Config: docsConfig(),
		Pages: []*engine.Page{
			{Title: "Visible", Slug: "visible", RelPermalink: "/docs/visible/", Kind: engine.KindPage},
			{Title: "Hidden", Slug: "hidden", RelPermalink: "/docs/hidden/", SidebarHidden: true, Kind: engine.KindPage},
		},
	}

	tree := BuildNavTree(col)

	if len(tree.Root.Children) != 1 {
		t.Fatalf("expected 1 child (hidden excluded), got %d", len(tree.Root.Children))
	}
	if tree.Root.Children[0].Label != "Visible" {
		t.Errorf("got %q", tree.Root.Children[0].Label)
	}
}

func TestBuildNavTree_SidebarLabel(t *testing.T) {
	col := &engine.Collection{
		Name:   "docs",
		Title:  "Docs",
		Config: docsConfig(),
		Pages: []*engine.Page{
			{Title: "Advanced Configuration Guide", SidebarLabel: "Config", Slug: "config", RelPermalink: "/docs/config/", Kind: engine.KindPage},
		},
	}

	tree := BuildNavTree(col)

	if tree.Root.Children[0].Label != "Config" {
		t.Errorf("expected SidebarLabel, got %q", tree.Root.Children[0].Label)
	}
}

func TestBuildNavTree_WeightSorting(t *testing.T) {
	col := &engine.Collection{
		Name:   "docs",
		Title:  "Docs",
		Config: docsConfig(),
		Pages: []*engine.Page{
			{Title: "Zebra", Slug: "z", RelPermalink: "/docs/z/", Weight: 3, Kind: engine.KindPage},
			{Title: "Apple", Slug: "a", RelPermalink: "/docs/a/", Weight: 1, Kind: engine.KindPage},
			{Title: "Mango", Slug: "m", RelPermalink: "/docs/m/", Weight: 2, Kind: engine.KindPage},
		},
	}

	tree := BuildNavTree(col)

	labels := make([]string, len(tree.Root.Children))
	for i, n := range tree.Root.Children {
		labels[i] = n.Label
	}
	expected := []string{"Apple", "Mango", "Zebra"}
	for i, l := range labels {
		if l != expected[i] {
			t.Errorf("position %d: got %q, want %q", i, l, expected[i])
		}
	}
}

func TestBuildNavTree_FlatList(t *testing.T) {
	sec := &engine.Section{
		Title:     "Section",
		Slug:      "sec",
		Permalink: "/docs/sec/",
		Render:    true,
		IndexPage: &engine.Page{Title: "Section", Kind: engine.KindSection},
		Pages: []*engine.Page{
			{Title: "B", Slug: "b", RelPermalink: "/docs/sec/b/", Weight: 2, Kind: engine.KindPage},
			{Title: "A", Slug: "a", RelPermalink: "/docs/sec/a/", Weight: 1, Kind: engine.KindPage},
		},
	}
	for _, p := range sec.Pages {
		p.Section = sec
	}
	sec.IndexPage.Section = sec

	col := &engine.Collection{
		Name:     "docs",
		Title:    "Docs",
		Config:   docsConfig(),
		Sections: []*engine.Section{sec},
		Pages: []*engine.Page{
			{Title: "Root", Slug: "root", RelPermalink: "/docs/root/", Weight: 0, Kind: engine.KindPage},
			sec.IndexPage,
			sec.Pages[0],
			sec.Pages[1],
		},
	}

	tree := BuildNavTree(col)

	if tree.TotalPages < 3 {
		t.Errorf("expected at least 3 flat pages, got %d", tree.TotalPages)
	}

	// Verify positions are sequential.
	for i, node := range tree.Flat {
		if node.Position != i {
			t.Errorf("node %d has position %d", i, node.Position)
		}
	}
}

func TestBuildNavTree_NilCollection(t *testing.T) {
	tree := BuildNavTree(nil)
	if tree != nil {
		t.Error("expected nil for nil collection")
	}
}

func TestBuildNavTree_SidebarAttrsOnNode(t *testing.T) {
	col := &engine.Collection{
		Name:   "docs",
		Title:  "Docs",
		Config: docsConfig(),
		Pages: []*engine.Page{
			{
				Title:        "API Page",
				Slug:         "api",
				RelPermalink: "/docs/api/",
				Weight:       1,
				Kind:         engine.KindPage,
				Params:       map[string]any{"sidebar_attrs": map[string]string{"icon": "star", "data-new": "true"}},
			},
		},
	}

	tree := BuildNavTree(col)

	if len(tree.Root.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(tree.Root.Children))
	}
	node := tree.Root.Children[0]
	if node.Attrs == nil {
		t.Fatal("expected Attrs to be set on NavNode")
	}
	if node.Attrs["icon"] != "star" {
		t.Errorf("Attrs[icon] = %q, want %q", node.Attrs["icon"], "star")
	}
	if node.Attrs["data-new"] != "true" {
		t.Errorf("Attrs[data-new] = %q, want %q", node.Attrs["data-new"], "true")
	}
}
