package navigation

import (
	"testing"

	"github.com/getsarde/sarde/internal/engine"
)

func intPtr(v int) *int    { return &v }
func boolPtr(v bool) *bool { return &v }

// overridesCollection builds a docs collection with one section (guides)
// containing two pages, one phantom section (extra), and one root page.
func overridesCollection(sidebar *engine.SidebarConfig) *engine.Collection {
	guides := &engine.Section{
		Title:     "Guides",
		Slug:      "guides",
		Permalink: "/docs/guides/",
		Render:    true,
		IndexPage: &engine.Page{PageIdentity: engine.PageIdentity{Title: "Guides", Kind: engine.KindSection, RelPermalink: "/docs/guides/"}, Sidebar: engine.PageSidebar{Order: 1}},
		Pages: []*engine.Page{
			{PageIdentity: engine.PageIdentity{Title: "Auth", Slug: "auth", RelPermalink: "/docs/guides/auth/", Kind: engine.KindPage}, Sidebar: engine.PageSidebar{Order: 1}},
			{PageIdentity: engine.PageIdentity{Title: "Deploy", Slug: "deploy", RelPermalink: "/docs/guides/deploy/", Kind: engine.KindPage}, Sidebar: engine.PageSidebar{Order: 2}},
		},
	}
	for _, p := range guides.Pages {
		p.Section = guides
	}
	guides.IndexPage.Section = guides

	// Phantom section: no _index.md, Render false.
	extra := &engine.Section{
		Title:     "Extra",
		Slug:      "extra",
		Permalink: "/docs/extra/",
		Render:    false,
		Pages: []*engine.Page{
			{PageIdentity: engine.PageIdentity{Title: "Note", Slug: "note", RelPermalink: "/docs/extra/note/", Kind: engine.KindPage}},
		},
	}
	extra.Pages[0].Section = extra

	rootPage := &engine.Page{PageIdentity: engine.PageIdentity{Title: "Getting Started", Slug: "getting-started", RelPermalink: "/docs/getting-started/", Kind: engine.KindPage}}

	return &engine.Collection{
		Name:     "docs",
		Title:    "Docs",
		Config:   &engine.CollectionConfig{Layout: engine.LayoutDocs, Sidebar: sidebar},
		Sections: []*engine.Section{guides, extra},
		Pages:    []*engine.Page{rootPage, guides.IndexPage, guides.Pages[0], guides.Pages[1], extra.Pages[0]},
	}
}

func findChild(node *engine.NavNode, label string) *engine.NavNode {
	for _, c := range node.Children {
		if c.Label == label {
			return c
		}
	}
	return nil
}

func TestBuildNavTree_OverrideSectionProperties(t *testing.T) {
	col := overridesCollection(&engine.SidebarConfig{
		MaxDepth: 4,
		Overrides: map[string]*engine.SidebarOverride{
			"guides": {
				Label:       "Handbook",
				Description: "All the guides.",
				Order:       intPtr(50),
				Icon:        "book-open",
				Badge:       engine.Badge{Text: "New", Variant: engine.BadgeVariantTip},
				Attrs:       map[string]string{"data-x": "y"},
			},
		},
	})

	tree := BuildNavTree(col)
	group := findChild(tree.Root, "Handbook")
	if group == nil {
		t.Fatal("expected group with overridden label Handbook")
	}
	if group.Order != 50 {
		t.Errorf("Order = %d, want 50 (override wins over frontmatter order 1)", group.Order)
	}
	if group.Icon != "book-open" {
		t.Errorf("Icon = %q", group.Icon)
	}
	if group.Badge.Text != "New" || group.Badge.Variant != engine.BadgeVariantTip {
		t.Errorf("Badge = %+v", group.Badge)
	}
	if group.Description != "All the guides." {
		t.Errorf("Description = %q", group.Description)
	}
	if group.Attrs["data-x"] != "y" {
		t.Errorf("Attrs = %v", group.Attrs)
	}
}

func TestBuildNavTree_OverridePageProperties(t *testing.T) {
	col := overridesCollection(&engine.SidebarConfig{
		MaxDepth: 4,
		Overrides: map[string]*engine.SidebarOverride{
			"guides/auth": {Label: "Authentication", Order: intPtr(9), Badge: engine.Badge{Text: "Updated"}},
		},
	})

	tree := BuildNavTree(col)
	group := findChild(tree.Root, "Guides")
	if group == nil {
		t.Fatal("guides group missing")
	}
	page := findChild(group, "Authentication")
	if page == nil {
		t.Fatal("expected page with overridden label")
	}
	if page.Order != 9 {
		t.Errorf("Order = %d, want 9", page.Order)
	}
	if page.Badge.Text != "Updated" {
		t.Errorf("Badge = %+v", page.Badge)
	}
	// Order 9 > sibling order 2, so Authentication sorts after Deploy.
	if group.Children[len(group.Children)-1].Label != "Authentication" {
		t.Errorf("override order must affect sorting: children %v", labels(group.Children))
	}
}

func TestBuildNavTree_HiddenSectionDropsSubtree(t *testing.T) {
	col := overridesCollection(&engine.SidebarConfig{
		MaxDepth: 4,
		Overrides: map[string]*engine.SidebarOverride{
			"guides": {Hidden: true},
		},
	})

	tree := BuildNavTree(col)
	if findChild(tree.Root, "Guides") != nil {
		t.Error("hidden section must be dropped")
	}
	for _, flat := range tree.Flat {
		if flat.Slug == "auth" || flat.Slug == "deploy" {
			t.Errorf("hidden section's children leaked into flat list: %q", flat.Slug)
		}
	}
}

func TestBuildNavTree_HiddenPageDropsLeafOnly(t *testing.T) {
	col := overridesCollection(&engine.SidebarConfig{
		MaxDepth: 4,
		Overrides: map[string]*engine.SidebarOverride{
			"guides/auth": {Hidden: true},
		},
	})

	tree := BuildNavTree(col)
	group := findChild(tree.Root, "Guides")
	if group == nil {
		t.Fatal("guides group missing")
	}
	if findChild(group, "Auth") != nil {
		t.Error("hidden page must be dropped")
	}
	if findChild(group, "Deploy") == nil {
		t.Error("sibling page must survive")
	}
}

func TestBuildNavTree_OverridePhantomSection(t *testing.T) {
	col := overridesCollection(&engine.SidebarConfig{
		MaxDepth: 4,
		Overrides: map[string]*engine.SidebarOverride{
			"extra": {Label: "Extras", Icon: "star", Order: intPtr(99)},
		},
	})

	tree := BuildNavTree(col)
	group := findChild(tree.Root, "Extras")
	if group == nil {
		t.Fatal("phantom section override must apply without _index.md")
	}
	if group.Icon != "star" || group.Order != 99 {
		t.Errorf("phantom override: Icon=%q Order=%d", group.Icon, group.Order)
	}
}

func TestBuildNavTree_CollapsedOverrideControlsDefaultOpen(t *testing.T) {
	col := overridesCollection(&engine.SidebarConfig{
		MaxDepth: 4,
		Overrides: map[string]*engine.SidebarOverride{
			"guides": {Collapsed: boolPtr(false)},
			"extra":  {Collapsed: boolPtr(true)},
		},
	})

	tree := BuildNavTree(col)
	if g := findChild(tree.Root, "Guides"); g == nil || !g.DefaultOpen {
		t.Error("collapsed false must force DefaultOpen")
	}
	if g := findChild(tree.Root, "Extra"); g == nil || g.DefaultOpen {
		t.Error("collapsed true must clear DefaultOpen")
	}
}

func TestBuildNavTree_CollapseLevel(t *testing.T) {
	col := overridesCollection(&engine.SidebarConfig{
		MaxDepth:      4,
		CollapseLevel: 1,
	})

	tree := BuildNavTree(col)
	if g := findChild(tree.Root, "Guides"); g == nil || !g.DefaultOpen {
		t.Error("collapse_level 1 must open depth-1 groups")
	}

	// Unset collapse_level leaves DefaultOpen untouched.
	col2 := overridesCollection(&engine.SidebarConfig{MaxDepth: 4})
	tree2 := BuildNavTree(col2)
	if g := findChild(tree2.Root, "Guides"); g == nil || g.DefaultOpen {
		t.Error("unset collapse_level must not set DefaultOpen")
	}
}

func TestBuildNavTree_LaneIndependentMatching(t *testing.T) {
	// Two lane collections sharing one *SidebarConfig: a key present in only
	// the second lane must end up matched (no unmatched warning material).
	sidebar := &engine.SidebarConfig{
		MaxDepth: 4,
		Overrides: map[string]*engine.SidebarOverride{
			"guides/auth": {Label: "Authentication"},
		},
	}

	lane1Pages := []*engine.Page{
		{PageIdentity: engine.PageIdentity{Title: "Other", Slug: "other", RelPermalink: "/docs/other/", Kind: engine.KindPage}},
	}
	lane1 := &engine.Collection{
		Name: "docs", Title: "Docs",
		Config: &engine.CollectionConfig{Layout: engine.LayoutDocs, Sidebar: sidebar},
		Pages:  lane1Pages,
	}
	BuildNavTree(lane1)
	if got := sidebar.UnmatchedOverrideKeys(); len(got) != 1 {
		t.Fatalf("after lane 1 the key must still be unmatched, got %v", got)
	}

	lane2 := overridesCollection(sidebar)
	BuildNavTree(lane2)
	if got := sidebar.UnmatchedOverrideKeys(); len(got) != 0 {
		t.Errorf("key matched in lane 2 must not be unmatched, got %v", got)
	}
}

func labels(nodes []*engine.NavNode) []string {
	out := make([]string, len(nodes))
	for i, n := range nodes {
		out[i] = n.Label
	}
	return out
}
