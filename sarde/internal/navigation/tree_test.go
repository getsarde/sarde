package navigation

import (
	"testing"

	"github.com/frostybee/sarde/internal/engine"
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
		IndexPage: &engine.Page{PageIdentity: engine.PageIdentity{Title: "Guides", Kind: engine.KindSection}, PageMeta: engine.PageMeta{Weight: 1}},
		Pages: []*engine.Page{
			{PageIdentity: engine.PageIdentity{Title: "Auth", Slug: "auth", RelPermalink: "/docs/guides/auth/", Kind: engine.KindPage}, PageMeta: engine.PageMeta{Weight: 1}},
			{PageIdentity: engine.PageIdentity{Title: "Deploy", Slug: "deploy", RelPermalink: "/docs/guides/deploy/", Kind: engine.KindPage}, PageMeta: engine.PageMeta{Weight: 2}},
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
			{PageIdentity: engine.PageIdentity{Title: "Getting Started", Slug: "getting-started", RelPermalink: "/docs/getting-started/", Kind: engine.KindPage}, PageMeta: engine.PageMeta{Weight: 0}},
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
			{PageIdentity: engine.PageIdentity{Title: "Page A", Slug: "a", RelPermalink: "/docs/a/", Kind: engine.KindPage}, PageMeta: engine.PageMeta{Weight: 1}},
			{PageIdentity: engine.PageIdentity{Title: "Page B", Slug: "b", RelPermalink: "/docs/b/", Kind: engine.KindPage}, PageMeta: engine.PageMeta{Weight: 2}},
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
		IndexPage: &engine.Page{PageIdentity: engine.PageIdentity{Title: "Advanced", Kind: engine.KindSection}, PageMeta: engine.PageMeta{Weight: 10}},
		Pages: []*engine.Page{
			{PageIdentity: engine.PageIdentity{Title: "Internals", Slug: "internals", RelPermalink: "/docs/advanced/internals/", Kind: engine.KindPage}},
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
			{PageIdentity: engine.PageIdentity{Title: "Visible", Slug: "visible", RelPermalink: "/docs/visible/", Kind: engine.KindPage}},
			{PageIdentity: engine.PageIdentity{Title: "Hidden", Slug: "hidden", RelPermalink: "/docs/hidden/", Kind: engine.KindPage}, PageSidebar: engine.PageSidebar{SidebarHidden: true}},
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
			{PageIdentity: engine.PageIdentity{Title: "Advanced Configuration Guide", Slug: "config", RelPermalink: "/docs/config/", Kind: engine.KindPage}, PageSidebar: engine.PageSidebar{SidebarLabel: "Config"}},
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
			{PageIdentity: engine.PageIdentity{Title: "Zebra", Slug: "z", RelPermalink: "/docs/z/", Kind: engine.KindPage}, PageMeta: engine.PageMeta{Weight: 3}},
			{PageIdentity: engine.PageIdentity{Title: "Apple", Slug: "a", RelPermalink: "/docs/a/", Kind: engine.KindPage}, PageMeta: engine.PageMeta{Weight: 1}},
			{PageIdentity: engine.PageIdentity{Title: "Mango", Slug: "m", RelPermalink: "/docs/m/", Kind: engine.KindPage}, PageMeta: engine.PageMeta{Weight: 2}},
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
		IndexPage: &engine.Page{PageIdentity: engine.PageIdentity{Title: "Section", Kind: engine.KindSection}},
		Pages: []*engine.Page{
			{PageIdentity: engine.PageIdentity{Title: "B", Slug: "b", RelPermalink: "/docs/sec/b/", Kind: engine.KindPage}, PageMeta: engine.PageMeta{Weight: 2}},
			{PageIdentity: engine.PageIdentity{Title: "A", Slug: "a", RelPermalink: "/docs/sec/a/", Kind: engine.KindPage}, PageMeta: engine.PageMeta{Weight: 1}},
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
			{PageIdentity: engine.PageIdentity{Title: "Root", Slug: "root", RelPermalink: "/docs/root/", Kind: engine.KindPage}, PageMeta: engine.PageMeta{Weight: 0}},
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
				PageIdentity: engine.PageIdentity{Title: "API Page", Slug: "api", RelPermalink: "/docs/api/", Kind: engine.KindPage},
				PageMeta:     engine.PageMeta{Weight: 1},
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
