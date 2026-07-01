package navigation

import (
	"testing"

	"github.com/getsarde/sarde/internal/engine"
)

func TestBuildBreadcrumbs_NestedPage(t *testing.T) {
	rootSec := &engine.Section{Title: "Documentation", Slug: "docs", Permalink: "/docs/"}
	guideSec := &engine.Section{Title: "Guides", Slug: "guides", Permalink: "/docs/guides/", Parent: rootSec}
	advSec := &engine.Section{Title: "Advanced", Slug: "advanced", Permalink: "/docs/guides/advanced/", Parent: guideSec}

	page := &engine.Page{
		PageIdentity:      engine.PageIdentity{Title: "Monitoring", RelPermalink: "/docs/guides/advanced/monitoring/", Kind: engine.KindPage},
		PageRelationships: engine.PageRelationships{Section: advSec},
	}

	col := &engine.Collection{Name: "docs", Title: "Documentation"}

	crumbs := BuildBreadcrumbs(page, col)

	// Documentation (collection root) → Guides → Advanced → Monitoring
	if len(crumbs) != 4 {
		t.Fatalf("expected 4 crumbs, got %d", len(crumbs))
	}

	if crumbs[0].Label != "Documentation" || crumbs[0].URL != "/docs/" {
		t.Errorf("crumb 0: %+v", crumbs[0])
	}
	if crumbs[1].Label != "Guides" {
		t.Errorf("crumb 1: %+v", crumbs[1])
	}
	if crumbs[2].Label != "Advanced" {
		t.Errorf("crumb 2: %+v", crumbs[2])
	}
	if crumbs[3].Label != "Monitoring" || !crumbs[3].Current {
		t.Errorf("crumb 3: %+v", crumbs[3])
	}
}

func TestBuildBreadcrumbs_RootPage(t *testing.T) {
	page := &engine.Page{
		PageIdentity: engine.PageIdentity{Title: "Getting Started", RelPermalink: "/docs/getting-started/", Kind: engine.KindPage},
	}

	col := &engine.Collection{Name: "docs", Title: "Documentation"}

	crumbs := BuildBreadcrumbs(page, col)

	if len(crumbs) != 2 {
		t.Fatalf("expected 2 crumbs, got %d", len(crumbs))
	}
	if crumbs[0].Label != "Documentation" {
		t.Errorf("crumb 0: %+v", crumbs[0])
	}
	if crumbs[1].Label != "Getting Started" || !crumbs[1].Current {
		t.Errorf("crumb 1: %+v", crumbs[1])
	}
}

func TestBuildBreadcrumbs_SectionIndex(t *testing.T) {
	rootSec := &engine.Section{Title: "Documentation", Slug: "docs", Permalink: "/docs/"}
	guideSec := &engine.Section{Title: "Guides", Slug: "guides", Permalink: "/docs/guides/", Parent: rootSec}
	page := &engine.Page{
		PageIdentity:      engine.PageIdentity{Title: "Guides", RelPermalink: "/docs/guides/", Kind: engine.KindSection},
		PageRelationships: engine.PageRelationships{Section: guideSec},
	}

	col := &engine.Collection{Name: "docs", Title: "Documentation"}

	crumbs := BuildBreadcrumbs(page, col)

	// Documentation (collection root) → Guides (current)
	if len(crumbs) != 2 {
		t.Fatalf("expected 2 crumbs, got %d", len(crumbs))
	}
	if !crumbs[1].Current {
		t.Error("expected last crumb to be current")
	}
}

func TestBuildBreadcrumbs_TransparentSectionSkipped(t *testing.T) {
	rootSec := &engine.Section{Title: "Docs", Slug: "docs", Permalink: "/docs/"}
	transparentSec := &engine.Section{Title: "Internal", Slug: "internal", Permalink: "/docs/internal/", Transparent: true, Parent: rootSec}

	page := &engine.Page{
		PageIdentity:      engine.PageIdentity{Title: "Config", RelPermalink: "/docs/config/", Kind: engine.KindPage},
		PageRelationships: engine.PageRelationships{Section: transparentSec},
	}

	col := &engine.Collection{Name: "docs", Title: "Docs"}

	crumbs := BuildBreadcrumbs(page, col)

	// Root section skipped (collection crumb), transparent skipped: Docs → Config
	if len(crumbs) != 2 {
		t.Fatalf("expected 2 crumbs (transparent skipped), got %d", len(crumbs))
	}
}

func TestBuildBreadcrumbs_Nil(t *testing.T) {
	if BuildBreadcrumbs(nil, nil) != nil {
		t.Error("expected nil")
	}
}

func TestBuildBreadcrumbsTabbed_DisjointSectionTrees(t *testing.T) {
	// Reproduce the pointer-identity bug: tab.Section points into tree A,
	// page.Section points into tree B (rebuilt by BuildSectionTree in buildTab).
	// Same permalinks, different pointers.

	// Tree A (original, referenced by tab.Section)
	treeARoot := &engine.Section{Title: "Docs", Slug: "docs", Permalink: "/docs/"}
	treeAGuide := &engine.Section{Title: "Guide", Slug: "guide", Permalink: "/docs/guide/", Parent: treeARoot}

	// Tree B (rebuilt, referenced by page.Section)
	treeBRoot := &engine.Section{Title: "Docs", Slug: "docs", Permalink: "/docs/"}
	treeBGuide := &engine.Section{Title: "Guide", Slug: "guide", Permalink: "/docs/guide/", Parent: treeBRoot}

	page := &engine.Page{
		PageIdentity:      engine.PageIdentity{Title: "Accordion", RelPermalink: "/docs/guide/accordion/", Kind: engine.KindPage},
		PageRelationships: engine.PageRelationships{Section: treeBGuide},
	}

	col := &engine.Collection{Name: "docs", Title: "Velox Docs"}
	tab := &engine.DocsTab{
		Title:     "Guide",
		Slug:      "guide",
		Permalink: "/docs/guide/",
		Section:   treeAGuide,
	}

	crumbs := BuildBreadcrumbsTabbed(page, col, tab)

	// Velox Docs → Guide → Accordion (no duplicates)
	if len(crumbs) != 3 {
		labels := make([]string, len(crumbs))
		for i, c := range crumbs {
			labels[i] = c.Label
		}
		t.Fatalf("expected 3 crumbs, got %d: %v", len(crumbs), labels)
	}
	if crumbs[0].Label != "Velox Docs" || crumbs[0].URL != "/docs/" {
		t.Errorf("crumb 0: %+v", crumbs[0])
	}
	if crumbs[1].Label != "Guide" || crumbs[1].URL != "/docs/guide/" {
		t.Errorf("crumb 1: %+v", crumbs[1])
	}
	if crumbs[2].Label != "Accordion" || !crumbs[2].Current {
		t.Errorf("crumb 2: %+v", crumbs[2])
	}
}

func TestBuildBreadcrumbsTabbed_NestedSection(t *testing.T) {
	tabSec := &engine.Section{Title: "Guide", Slug: "guide", Permalink: "/docs/guide/"}
	advSec := &engine.Section{
		Title: "Advanced", Slug: "advanced", Permalink: "/docs/guide/advanced/",
		Parent:    tabSec,
		IndexPage: &engine.Page{PageIdentity: engine.PageIdentity{Title: "Advanced", RelPermalink: "/docs/guide/advanced/"}},
	}

	page := &engine.Page{
		PageIdentity:      engine.PageIdentity{Title: "Hooks", RelPermalink: "/docs/guide/advanced/hooks/", Kind: engine.KindPage},
		PageRelationships: engine.PageRelationships{Section: advSec},
	}

	col := &engine.Collection{Name: "docs", Title: "Docs"}
	tab := &engine.DocsTab{
		Title:     "Guide",
		Slug:      "guide",
		Permalink: "/docs/guide/",
		Section:   tabSec,
	}

	crumbs := BuildBreadcrumbsTabbed(page, col, tab)

	// Docs → Guide → Advanced → Hooks
	if len(crumbs) != 4 {
		labels := make([]string, len(crumbs))
		for i, c := range crumbs {
			labels[i] = c.Label
		}
		t.Fatalf("expected 4 crumbs, got %d: %v", len(crumbs), labels)
	}
	if crumbs[0].Label != "Docs" {
		t.Errorf("crumb 0: %+v", crumbs[0])
	}
	if crumbs[1].Label != "Guide" {
		t.Errorf("crumb 1: %+v", crumbs[1])
	}
	if crumbs[2].Label != "Advanced" {
		t.Errorf("crumb 2: %+v", crumbs[2])
	}
	if crumbs[3].Label != "Hooks" || !crumbs[3].Current {
		t.Errorf("crumb 3: %+v", crumbs[3])
	}
}
