package navigation

import (
	"testing"

	"github.com/frostybee/sarde/internal/engine"
)

func TestBuildBreadcrumbs_NestedPage(t *testing.T) {
	parentSec := &engine.Section{Title: "Guides", Slug: "guides", Permalink: "/docs/guides/"}
	childSec := &engine.Section{Title: "Advanced", Slug: "advanced", Permalink: "/docs/guides/advanced/", Parent: parentSec}

	page := &engine.Page{
		Title:        "Monitoring",
		RelPermalink: "/docs/guides/advanced/monitoring/",
		Kind:         engine.KindPage,
		Section:      childSec,
	}

	col := &engine.Collection{Name: "docs", Title: "Documentation"}

	crumbs := BuildBreadcrumbs(page, col)

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
		Title:        "Getting Started",
		RelPermalink: "/docs/getting-started/",
		Kind:         engine.KindPage,
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
	sec := &engine.Section{Title: "Guides", Slug: "guides", Permalink: "/docs/guides/"}
	page := &engine.Page{
		Title:        "Guides",
		RelPermalink: "/docs/guides/",
		Kind:         engine.KindSection,
		Section:      sec,
	}

	col := &engine.Collection{Name: "docs", Title: "Documentation"}

	crumbs := BuildBreadcrumbs(page, col)

	if len(crumbs) != 2 {
		t.Fatalf("expected 2 crumbs, got %d", len(crumbs))
	}
	// Last crumb should be current.
	if !crumbs[1].Current {
		t.Error("expected last crumb to be current")
	}
}

func TestBuildBreadcrumbs_TransparentSectionSkipped(t *testing.T) {
	parentSec := &engine.Section{Title: "Internal", Slug: "internal", Permalink: "/docs/internal/", Transparent: true}

	page := &engine.Page{
		Title:        "Config",
		RelPermalink: "/docs/config/",
		Kind:         engine.KindPage,
		Section:      parentSec,
	}

	col := &engine.Collection{Name: "docs", Title: "Docs"}

	crumbs := BuildBreadcrumbs(page, col)

	// Transparent section should be skipped: Docs → Config
	if len(crumbs) != 2 {
		t.Fatalf("expected 2 crumbs (transparent skipped), got %d", len(crumbs))
	}
}

func TestBuildBreadcrumbs_Nil(t *testing.T) {
	if BuildBreadcrumbs(nil, nil) != nil {
		t.Error("expected nil")
	}
}
