package links

import (
	"testing"

	"github.com/getsarde/sarde/internal/engine"
)

func TestEnumerateLanes_SingleLang_NoVersioning(t *testing.T) {
	collections := map[string]*engine.Collection{
		"docs": {Name: "docs", Config: &engine.CollectionConfig{}},
		"blog": {Name: "blog", Config: &engine.CollectionConfig{}},
	}

	lanes := EnumerateLanes(collections, []string{""})
	if len(lanes) != 2 {
		t.Fatalf("expected 2 lanes, got %d", len(lanes))
	}
	// Sorted: blog first, docs second.
	if lanes[0].Collection != "blog" || lanes[1].Collection != "docs" {
		t.Errorf("expected [blog, docs], got [%s, %s]", lanes[0].Collection, lanes[1].Collection)
	}
}

func TestEnumerateLanes_MultiLang(t *testing.T) {
	collections := map[string]*engine.Collection{
		"docs": {Name: "docs", Config: &engine.CollectionConfig{}},
	}

	lanes := EnumerateLanes(collections, []string{"en", "fr", "ar"})
	if len(lanes) != 3 {
		t.Fatalf("expected 3 lanes, got %d", len(lanes))
	}
	for i, expectedLang := range []string{"ar", "en", "fr"} {
		if lanes[i].Lang != expectedLang {
			t.Errorf("lane %d: expected lang %q, got %q", i, expectedLang, lanes[i].Lang)
		}
	}
}

func TestEnumerateLanes_Versioned(t *testing.T) {
	collections := map[string]*engine.Collection{
		"docs": {
			Name: "docs",
			Config: &engine.CollectionConfig{
				Versioning: &engine.VersionConfig{
					Enabled: true,
					Versions: []engine.VersionDef{
						{ID: "v1"},
						{ID: "v2"},
					},
				},
			},
		},
	}

	lanes := EnumerateLanes(collections, []string{"en", "fr"})
	// 1 collection × 2 langs × 2 versions = 4 lanes
	if len(lanes) != 4 {
		t.Fatalf("expected 4 lanes, got %d", len(lanes))
	}
	// Sorted by collection, lang, version.
	expected := []DimKey{
		{Collection: "docs", Lang: "en", Version: "v1"},
		{Collection: "docs", Lang: "en", Version: "v2"},
		{Collection: "docs", Lang: "fr", Version: "v1"},
		{Collection: "docs", Lang: "fr", Version: "v2"},
	}
	for i, e := range expected {
		if lanes[i] != e {
			t.Errorf("lane %d: expected %+v, got %+v", i, e, lanes[i])
		}
	}
}

func TestEnumerateLanes_MixedVersionedAndUnversioned(t *testing.T) {
	collections := map[string]*engine.Collection{
		"docs": {
			Name: "docs",
			Config: &engine.CollectionConfig{
				Versioning: &engine.VersionConfig{
					Enabled:  true,
					Versions: []engine.VersionDef{{ID: "v1"}, {ID: "v2"}},
				},
			},
		},
		"blog": {Name: "blog", Config: &engine.CollectionConfig{}},
	}

	lanes := EnumerateLanes(collections, []string{"en"})
	// blog: 1 lane, docs: 2 lanes = 3
	if len(lanes) != 3 {
		t.Fatalf("expected 3 lanes, got %d", len(lanes))
	}
}

func TestEnumerateLanes_EmptyLanguages(t *testing.T) {
	collections := map[string]*engine.Collection{
		"docs": {Name: "docs", Config: &engine.CollectionConfig{}},
	}

	lanes := EnumerateLanes(collections, nil)
	if len(lanes) != 1 {
		t.Fatalf("expected 1 lane, got %d", len(lanes))
	}
	if lanes[0].Lang != "" {
		t.Errorf("expected empty lang, got %q", lanes[0].Lang)
	}
}

func TestComputeCoverage_AllLanesCovered(t *testing.T) {
	docsColl := &engine.Collection{Name: "docs", Config: &engine.CollectionConfig{}}

	pages := []*engine.Page{
		{
			PageIdentity:      engine.PageIdentity{FilePath: "content/docs/guide.md"},
			PageContent:       engine.PageContent{RawContent: "# Guide\n[auth](./auth.md)"},
			PageI18n:          engine.PageI18n{Lang: "en"},
			PageRelationships: engine.PageRelationships{Collection: docsColl},
		},
	}

	graph := NewLinkGraph()
	graph.Record(LinkRef{
		FromFile: "content/docs/guide.md",
		RawDest:  "./auth.md",
		Dim:      DimKey{Collection: "docs", Lang: "en"},
		Status:   StatusOK,
	})

	expected := EnumerateLanes(map[string]*engine.Collection{"docs": docsColl}, []string{"en"})
	summary := ComputeCoverage(graph, pages, expected)

	if summary.TotalLanes != 1 {
		t.Errorf("expected 1 lane, got %d", summary.TotalLanes)
	}
	if summary.TotalPages != 1 {
		t.Errorf("expected 1 page, got %d", summary.TotalPages)
	}
	if summary.TotalLinks != 1 {
		t.Errorf("expected 1 link, got %d", summary.TotalLinks)
	}
	if len(summary.MissedLanes) != 0 {
		t.Errorf("expected 0 missed lanes, got %d", len(summary.MissedLanes))
	}
}

func TestComputeCoverage_MissedLane(t *testing.T) {
	docsColl := &engine.Collection{Name: "docs", Config: &engine.CollectionConfig{}}

	// Only English pages, no French.
	pages := []*engine.Page{
		{
			PageIdentity:      engine.PageIdentity{FilePath: "content/docs/guide.md"},
			PageContent:       engine.PageContent{RawContent: "# Guide"},
			PageI18n:          engine.PageI18n{Lang: "en"},
			PageRelationships: engine.PageRelationships{Collection: docsColl},
		},
	}

	graph := NewLinkGraph()
	graph.Record(LinkRef{
		FromFile: "content/docs/guide.md",
		RawDest:  "./auth.md",
		Dim:      DimKey{Collection: "docs", Lang: "en"},
		Status:   StatusOK,
	})

	expected := EnumerateLanes(map[string]*engine.Collection{"docs": docsColl}, []string{"en", "fr"})
	summary := ComputeCoverage(graph, pages, expected)

	if len(summary.MissedLanes) != 1 {
		t.Fatalf("expected 1 missed lane, got %d", len(summary.MissedLanes))
	}
	if summary.MissedLanes[0].Lang != "fr" {
		t.Errorf("expected missed lane lang 'fr', got %q", summary.MissedLanes[0].Lang)
	}
}

func TestComputeCoverage_BrokenInOneLane(t *testing.T) {
	docsColl := &engine.Collection{Name: "docs", Config: &engine.CollectionConfig{}}

	pages := []*engine.Page{
		{
			PageIdentity:      engine.PageIdentity{FilePath: "content/docs/guide.md"},
			PageContent:       engine.PageContent{RawContent: "# Guide"},
			PageI18n:          engine.PageI18n{Lang: "en"},
			PageRelationships: engine.PageRelationships{Collection: docsColl},
		},
		{
			PageIdentity:      engine.PageIdentity{FilePath: "content/fr/docs/guide.md"},
			PageContent:       engine.PageContent{RawContent: "# Guide"},
			PageI18n:          engine.PageI18n{Lang: "fr", IsFallback: true},
			PageRelationships: engine.PageRelationships{Collection: docsColl},
		},
	}

	graph := NewLinkGraph()
	// OK in English.
	graph.Record(LinkRef{
		FromFile: "content/docs/guide.md",
		RawDest:  "./auth.md",
		Dim:      DimKey{Collection: "docs", Lang: "en"},
		Status:   StatusOK,
	})
	// Broken in French.
	graph.Record(LinkRef{
		FromFile: "content/fr/docs/guide.md",
		RawDest:  "./auth.md",
		Dim:      DimKey{Collection: "docs", Lang: "fr"},
		Status:   StatusBrokenTarget,
	})

	expected := EnumerateLanes(map[string]*engine.Collection{"docs": docsColl}, []string{"en", "fr"})
	summary := ComputeCoverage(graph, pages, expected)

	if summary.TotalBroken != 1 {
		t.Errorf("expected 1 total broken, got %d", summary.TotalBroken)
	}
	if len(summary.MissedLanes) != 0 {
		t.Errorf("expected 0 missed lanes, got %d", len(summary.MissedLanes))
	}

	// Verify per-lane stats.
	for _, lane := range summary.Lanes {
		if lane.Dim.Lang == "en" && lane.Broken != 0 {
			t.Error("English lane should have 0 broken")
		}
		if lane.Dim.Lang == "fr" && lane.Broken != 1 {
			t.Error("French lane should have 1 broken")
		}
		if lane.Dim.Lang == "fr" && !lane.IsFallback {
			t.Error("French lane should be marked as fallback")
		}
	}
}

func TestComputeCoverage_VersionedLanes(t *testing.T) {
	docsColl := &engine.Collection{
		Name: "docs",
		Config: &engine.CollectionConfig{
			Versioning: &engine.VersionConfig{
				Enabled:  true,
				Versions: []engine.VersionDef{{ID: "v1"}, {ID: "v2"}},
			},
		},
	}

	pages := []*engine.Page{
		{
			PageIdentity:      engine.PageIdentity{FilePath: "content/docs/v1/guide.md"},
			PageContent:       engine.PageContent{RawContent: "# Guide v1"},
			PageI18n:          engine.PageI18n{Lang: "en"},
			PageVersioning:    engine.PageVersioning{Version: "v1"},
			PageRelationships: engine.PageRelationships{Collection: docsColl},
		},
		{
			PageIdentity:      engine.PageIdentity{FilePath: "content/docs/v2/guide.md"},
			PageContent:       engine.PageContent{RawContent: "# Guide v2"},
			PageI18n:          engine.PageI18n{Lang: "en"},
			PageVersioning:    engine.PageVersioning{Version: "v2"},
			PageRelationships: engine.PageRelationships{Collection: docsColl},
		},
	}

	graph := NewLinkGraph()
	graph.Record(LinkRef{
		RawDest: "./auth.md",
		Dim:     DimKey{Collection: "docs", Lang: "en", Version: "v1"},
		Status:  StatusOK,
	})
	graph.Record(LinkRef{
		RawDest: "./auth.md",
		Dim:     DimKey{Collection: "docs", Lang: "en", Version: "v2"},
		Status:  StatusBrokenTarget,
	})

	expected := EnumerateLanes(map[string]*engine.Collection{"docs": docsColl}, []string{"en"})
	summary := ComputeCoverage(graph, pages, expected)

	if summary.TotalLanes != 2 {
		t.Errorf("expected 2 lanes, got %d", summary.TotalLanes)
	}
	if summary.TotalBroken != 1 {
		t.Errorf("expected 1 broken, got %d", summary.TotalBroken)
	}
}

func TestComputeCoverage_PagesWithNoLinks(t *testing.T) {
	docsColl := &engine.Collection{Name: "docs", Config: &engine.CollectionConfig{}}

	pages := []*engine.Page{
		{
			PageIdentity:      engine.PageIdentity{FilePath: "content/docs/no-links.md"},
			PageContent:       engine.PageContent{RawContent: "# No links here"},
			PageI18n:          engine.PageI18n{Lang: "en"},
			PageRelationships: engine.PageRelationships{Collection: docsColl},
		},
	}

	graph := NewLinkGraph()

	expected := EnumerateLanes(map[string]*engine.Collection{"docs": docsColl}, []string{"en"})
	summary := ComputeCoverage(graph, pages, expected)

	// Page exists but has no links — the lane is still "covered" (the page was rendered).
	if summary.TotalLanes != 1 {
		t.Errorf("expected 1 lane, got %d", summary.TotalLanes)
	}
	if summary.TotalLinks != 0 {
		t.Errorf("expected 0 links, got %d", summary.TotalLinks)
	}
	if len(summary.MissedLanes) != 0 {
		t.Errorf("expected 0 missed lanes, got %d", len(summary.MissedLanes))
	}
}

func TestComputeCoverage_SkipsPagesWithNoContent(t *testing.T) {
	docsColl := &engine.Collection{Name: "docs", Config: &engine.CollectionConfig{}}

	pages := []*engine.Page{
		{
			PageIdentity:      engine.PageIdentity{FilePath: "content/docs/empty.md"},
			PageContent:       engine.PageContent{RawContent: ""}, // no content
			PageI18n:          engine.PageI18n{Lang: "en"},
			PageRelationships: engine.PageRelationships{Collection: docsColl},
		},
	}

	graph := NewLinkGraph()
	expected := EnumerateLanes(map[string]*engine.Collection{"docs": docsColl}, []string{"en"})
	summary := ComputeCoverage(graph, pages, expected)

	if summary.TotalPages != 0 {
		t.Errorf("expected 0 pages (empty content skipped), got %d", summary.TotalPages)
	}
	if len(summary.MissedLanes) != 1 {
		t.Errorf("expected 1 missed lane (no pages with content), got %d", len(summary.MissedLanes))
	}
}

func TestComputeCoverage_ExternalLinks(t *testing.T) {
	docsColl := &engine.Collection{Name: "docs", Config: &engine.CollectionConfig{}}

	pages := []*engine.Page{
		{
			PageIdentity:      engine.PageIdentity{FilePath: "content/docs/guide.md"},
			PageContent:       engine.PageContent{RawContent: "# Guide"},
			PageI18n:          engine.PageI18n{Lang: "en"},
			PageRelationships: engine.PageRelationships{Collection: docsColl},
		},
	}

	graph := NewLinkGraph()
	graph.Record(LinkRef{
		RawDest: "https://example.com",
		Dim:     DimKey{Collection: "docs", Lang: "en"},
		Status:  StatusExternal,
	})

	expected := EnumerateLanes(map[string]*engine.Collection{"docs": docsColl}, []string{"en"})
	summary := ComputeCoverage(graph, pages, expected)

	if summary.Lanes[0].External != 1 {
		t.Errorf("expected 1 external link, got %d", summary.Lanes[0].External)
	}
}
