package collection

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/content"
	"github.com/getsarde/sarde/internal/engine"
	"github.com/getsarde/sarde/internal/navigation"
)

func intPtr(v int) *int { return &v }

// ---------------------------------------------------------------------------
// ApplySidebarFile
// ---------------------------------------------------------------------------

func TestApplySidebarFile_NilEntryNoOp(t *testing.T) {
	cfg := InferCollection("docs")
	out := ApplySidebarFile(cfg, nil)
	if out != cfg {
		t.Error("nil entry must return the input config unchanged")
	}
}

func TestApplySidebarFile_EmptyEntryNoOp(t *testing.T) {
	cfg := InferCollection("docs")
	out := ApplySidebarFile(cfg, &config.SidebarCollectionEntry{})
	if out != cfg {
		t.Error("empty entry must return the input config unchanged")
	}
}

func TestApplySidebarFile_ConvertsOverridesAndTabs(t *testing.T) {
	cfg := InferCollection("docs")
	entry := &config.SidebarCollectionEntry{
		Tabs: map[string]*config.SidebarTabOverride{
			"guide": {Label: "Getting Started", Icon: "book-open", Order: intPtr(10)},
		},
		Overrides: map[string]*config.SidebarNodeOverride{
			"guide/advanced": {
				Label:     "Advanced Topics",
				Order:     intPtr(20),
				Collapsed: boolPtr(true),
				Badge:     engine.Badge{Text: "New", Variant: engine.BadgeVariantTip},
				Hidden:    boolPtr(false),
				Attrs:     map[string]string{"data-x": "y"},
			},
		},
	}

	out := ApplySidebarFile(cfg, entry)

	// Input must not be mutated.
	if cfg.Sidebar.Overrides != nil || cfg.Sidebar.TabOverrides != nil {
		t.Error("input config was mutated")
	}

	ov := out.Sidebar.Overrides["guide/advanced"]
	if ov == nil {
		t.Fatal("expected guide/advanced override")
	}
	if ov.Label != "Advanced Topics" || *ov.Order != 20 || !*ov.Collapsed {
		t.Errorf("override fields: got %+v", ov)
	}
	if ov.Hidden == nil || *ov.Hidden {
		t.Errorf("hidden pointer must copy through: got %v", ov.Hidden)
	}
	if ov.Badge.Text != "New" || ov.Badge.Variant != engine.BadgeVariantTip {
		t.Errorf("badge: got %+v", ov.Badge)
	}
	if ov.Attrs["data-x"] != "y" {
		t.Errorf("attrs: got %v", ov.Attrs)
	}

	tab := out.Sidebar.TabOverrides["guide"]
	if tab == nil {
		t.Fatal("expected guide tab override")
	}
	if tab.Label != "Getting Started" || tab.Icon != "book-open" || *tab.Order != 10 {
		t.Errorf("tab override: got %+v", tab)
	}

	// Existing sidebar behavior settings must survive.
	if out.Sidebar.MaxDepth != cfg.Sidebar.MaxDepth || out.Sidebar.Collapsible != cfg.Sidebar.Collapsible {
		t.Error("existing sidebar settings lost during overlay")
	}
}

func TestApplySidebarFile_AllocatesSidebarWhenNil(t *testing.T) {
	cfg := &engine.CollectionConfig{Layout: engine.LayoutDocs} // no Sidebar block
	entry := &config.SidebarCollectionEntry{
		Overrides: map[string]*config.SidebarNodeOverride{
			"guide": {Label: "Guide"},
		},
	}
	out := ApplySidebarFile(cfg, entry)
	if out.Sidebar == nil {
		t.Fatal("Sidebar must be lazily allocated")
	}
	if out.Sidebar.Overrides["guide"].Label != "Guide" {
		t.Errorf("override: got %+v", out.Sidebar.Overrides["guide"])
	}
}

func TestApplySidebarFile_CollapseLevelWinsOverSardeYAML(t *testing.T) {
	inferred := InferCollection("docs")
	merged := MergeCollectionConfig(inferred, &config.CollectionSiteConfig{
		Sidebar: &config.CollectionSidebarConfig{CollapseLevel: intPtr(3)},
	})
	if merged.Sidebar.CollapseLevel != 3 {
		t.Fatalf("MergeCollectionConfig collapse_level: got %d", merged.Sidebar.CollapseLevel)
	}

	out := ApplySidebarFile(merged, &config.SidebarCollectionEntry{CollapseLevel: intPtr(1)})
	if out.Sidebar.CollapseLevel != 1 {
		t.Errorf("sidebar.yaml collapse_level must win: got %d", out.Sidebar.CollapseLevel)
	}
	if merged.Sidebar.CollapseLevel != 3 {
		t.Error("input config was mutated")
	}
}

// ---------------------------------------------------------------------------
// Tab overrides
// ---------------------------------------------------------------------------

func tabbedTestCollection(t *testing.T, sidebar *engine.SidebarConfig) *engine.Collection {
	t.Helper()
	pages := []*engine.Page{
		{PageIdentity: engine.PageIdentity{Title: "Docs", Slug: "docs", Kind: engine.KindSection, RelPermalink: "/docs/"}},
		{PageIdentity: engine.PageIdentity{Title: "Guide", Slug: "guide", Kind: engine.KindSection, RelPermalink: "/docs/guide/"}, Params: map[string]any{"icon": "compass"}},
		{PageIdentity: engine.PageIdentity{Title: "Intro", Slug: "intro", Kind: engine.KindPage, RelPermalink: "/docs/guide/intro/"}},
		// "extra" has no _index.md: phantom top-level section.
		{PageIdentity: engine.PageIdentity{Title: "Note", Slug: "note", Kind: engine.KindPage, RelPermalink: "/docs/extra/note/"}},
	}
	return &engine.Collection{
		Name:     "docs",
		Title:    "Docs",
		Config:   &engine.CollectionConfig{Layout: engine.LayoutDocs, Sidebar: sidebar},
		Pages:    pages,
		Sections: BuildSectionTree(pages, "docs"),
	}
}

func TestBuildTabs_TabOverrideAppliesAndWinsOverFrontmatter(t *testing.T) {
	col := tabbedTestCollection(t, &engine.SidebarConfig{
		TabOverrides: map[string]*engine.TabOverride{
			"guide": {Label: "Getting Started", Icon: "book-open", Order: intPtr(10), Description: "Basics."},
		},
	})

	tabs := BuildTabs(col, "")
	var guide *engine.DocsTab
	for _, tab := range tabs {
		if tab.Slug == "guide" {
			guide = tab
		}
	}
	if guide == nil {
		t.Fatal("guide tab missing")
	}
	if guide.Title != "Getting Started" {
		t.Errorf("Title = %q, want override to win over _index.md title", guide.Title)
	}
	if guide.Icon != "book-open" {
		t.Errorf("Icon = %q, want override to win over frontmatter icon", guide.Icon)
	}
	if guide.Order != 10 {
		t.Errorf("Order = %d, want 10", guide.Order)
	}
	if guide.Description != "Basics." {
		t.Errorf("Description = %q", guide.Description)
	}
}

func TestBuildTabs_TabOverrideOnPhantomSection(t *testing.T) {
	col := tabbedTestCollection(t, &engine.SidebarConfig{
		TabOverrides: map[string]*engine.TabOverride{
			"extra": {Label: "Extras", Icon: "star"},
		},
	})

	tabs := BuildTabs(col, "")
	var extra *engine.DocsTab
	for _, tab := range tabs {
		if tab.Slug == "extra" {
			extra = tab
		}
	}
	if extra == nil {
		t.Fatal("phantom extra tab missing")
	}
	if extra.Title != "Extras" || extra.Icon != "star" {
		t.Errorf("phantom tab override: Title=%q Icon=%q", extra.Title, extra.Icon)
	}
}

// ---------------------------------------------------------------------------
// Unmatched-key warnings
// ---------------------------------------------------------------------------

func TestCollectSidebarOverrideWarnings(t *testing.T) {
	sidebar := &engine.SidebarConfig{
		MaxDepth: 4,
		Overrides: map[string]*engine.SidebarOverride{
			"guide/intro":    {Label: "Intro"},
			"does/not/exist": {Hidden: boolPtr(true)},
		},
		TabOverrides: map[string]*engine.TabOverride{
			"guide":  {Label: "Guide"},
			"no-tab": {Label: "Nope"},
		},
	}
	col := tabbedTestCollection(t, sidebar)

	// Build tabs (marks tab keys) and the nav tree (marks override keys).
	col.Tabs = BuildTabs(col, "")
	navigation.BuildNavTree(col)

	warnings := CollectSidebarOverrideWarnings(map[string]*engine.Collection{"docs": col})

	var fields []string
	for _, w := range warnings {
		fields = append(fields, w.Field)
	}
	joined := strings.Join(fields, ";")
	if !strings.Contains(joined, "docs.does/not/exist") {
		t.Errorf("expected unmatched override warning, got %v", fields)
	}
	if !strings.Contains(joined, "docs.tabs.no-tab") {
		t.Errorf("expected unmatched tab warning, got %v", fields)
	}
	if strings.Contains(joined, "docs.guide/intro") || strings.Contains(joined, "docs.tabs.guide") {
		t.Errorf("matched keys must not warn, got %v", fields)
	}
}

func TestBuildCollections_SidebarFileSkippedForNonSidebarLayout(t *testing.T) {
	contentDir := t.TempDir()
	writeTestFile(t, contentDir, filepath.Join("blog", "post.md"), "---\ntitle: Post\n---\nBody.\n")

	scanner := &content.Scanner{}
	files, err := scanner.DiscoverFiles(contentDir)
	if err != nil {
		t.Fatal(err)
	}

	siteCfg := config.Defaults()
	siteCfg.SidebarFile = config.SidebarFile{
		"blog": &config.SidebarCollectionEntry{
			Overrides: map[string]*config.SidebarNodeOverride{
				"post": {Label: "Renamed"},
			},
		},
	}

	collections, warnings, err := BuildCollections(files, siteCfg, contentDir, nil)
	if err != nil {
		t.Fatal(err)
	}

	blog := collections["blog"]
	if blog == nil {
		t.Fatal("blog collection missing")
	}
	if blog.Config.Sidebar != nil && blog.Config.Sidebar.Overrides != nil {
		t.Error("overrides must not attach to a non-sidebar layout")
	}

	ignored := 0
	for _, w := range warnings {
		if strings.Contains(w.Message, "does not use a sidebar layout") {
			ignored++
		}
	}
	if ignored != 1 {
		t.Errorf("expected exactly one entry-ignored warning, got %d in %v", ignored, warnings)
	}
	if extra := CollectSidebarOverrideWarnings(collections); len(extra) != 0 {
		t.Errorf("non-sidebar layout must produce no unmatched-key warnings, got %v", extra)
	}
}

func TestRebuildNavTreesWithFallbacks_HonorsNavYAML(t *testing.T) {
	contentDir := t.TempDir()
	writeTestFile(t, contentDir, filepath.Join("docs", "guide", "nav.yaml"),
		"- label: Custom Group\n  items:\n    - page: guide/intro\n")

	pages := []*engine.Page{
		{PageIdentity: engine.PageIdentity{Title: "Guide", Slug: "guide", Kind: engine.KindSection, RelPermalink: "/docs/guide/"}, PageI18n: engine.PageI18n{Lang: "en"}},
		{PageIdentity: engine.PageIdentity{Title: "Intro", Slug: "intro", Kind: engine.KindPage, RelPermalink: "/docs/guide/intro/"}, PageI18n: engine.PageI18n{Lang: "en"}},
		{PageIdentity: engine.PageIdentity{Title: "API", Slug: "api", Kind: engine.KindSection, RelPermalink: "/docs/api/"}, PageI18n: engine.PageI18n{Lang: "en"}},
		{PageIdentity: engine.PageIdentity{Title: "Ref", Slug: "ref", Kind: engine.KindPage, RelPermalink: "/docs/api/ref/"}, PageI18n: engine.PageI18n{Lang: "en"}},
	}
	col := &engine.Collection{
		Name:     "docs",
		Title:    "Docs",
		Config:   &engine.CollectionConfig{Layout: engine.LayoutDocs},
		Pages:    pages,
		Sections: BuildSectionTree(pages, "docs"),
		IsTabbed: true,
	}
	for _, p := range pages {
		p.Collection = col
	}
	col.Tabs = BuildTabs(col, contentDir)

	RebuildNavTreesWithFallbacks(map[string]*engine.Collection{"docs": col}, pages, []string{"en", "fr"}, contentDir)

	var guide *engine.DocsTab
	for _, tab := range col.Tabs {
		if tab.Slug == "guide" {
			guide = tab
		}
	}
	if guide == nil {
		t.Fatal("guide tab missing")
	}
	tree := guide.NavTrees["en"]
	if tree == nil || len(tree.Root.Children) == 0 {
		t.Fatal("guide tab en tree missing after rebuild")
	}
	if tree.Root.Children[0].Label != "Custom Group" {
		t.Errorf("rebuild must honor nav.yaml, got root child %q", tree.Root.Children[0].Label)
	}
}

func TestCollectSidebarOverrideWarnings_LaneUnion(t *testing.T) {
	// A key matched in only one of two lanes must not warn: two synthetic
	// lane collections share the same *SidebarConfig pointer.
	sidebar := &engine.SidebarConfig{
		MaxDepth: 4,
		Overrides: map[string]*engine.SidebarOverride{
			"v2-only": {Label: "V2 Feature"},
		},
	}
	cfg := &engine.CollectionConfig{Layout: engine.LayoutDocs, Sidebar: sidebar}

	lane1Pages := []*engine.Page{
		{PageIdentity: engine.PageIdentity{Title: "Old", Slug: "old", Kind: engine.KindPage, RelPermalink: "/docs/old/"}},
	}
	lane2Pages := []*engine.Page{
		{PageIdentity: engine.PageIdentity{Title: "V2 Only", Slug: "v2-only", Kind: engine.KindPage, RelPermalink: "/docs/v2-only/"}},
	}
	lane1 := &engine.Collection{Name: "docs", Title: "Docs", Config: cfg, Pages: lane1Pages, Sections: BuildSectionTree(lane1Pages, "docs")}
	lane2 := &engine.Collection{Name: "docs", Title: "Docs", Config: cfg, Pages: lane2Pages, Sections: BuildSectionTree(lane2Pages, "docs")}

	navigation.BuildNavTree(lane1) // does not match v2-only
	navigation.BuildNavTree(lane2) // matches v2-only

	warnings := CollectSidebarOverrideWarnings(map[string]*engine.Collection{"docs": lane1})
	if len(warnings) != 0 {
		t.Errorf("key matched in one lane must not warn, got %v", warnings)
	}
}
