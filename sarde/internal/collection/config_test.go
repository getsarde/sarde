package collection

import (
	"testing"

	"github.com/frostybee/sarde/internal/config"
	"github.com/frostybee/sarde/internal/engine"
)

func boolPtr(v bool) *bool { return &v }

func TestMergeCollectionConfig_NilSiteCfg(t *testing.T) {
	inferred := InferCollection("blog")
	merged := MergeCollectionConfig(inferred, nil)
	if merged.SortBy != "date" {
		t.Errorf("SortBy = %q, want %q", merged.SortBy, "date")
	}
}

func TestMergeCollectionConfig_SortOverride(t *testing.T) {
	inferred := InferCollection("blog") // date desc
	siteCfg := &config.CollectionSiteConfig{Sort: "title asc"}
	merged := MergeCollectionConfig(inferred, siteCfg)
	if merged.SortBy != "title" {
		t.Errorf("SortBy = %q, want %q", merged.SortBy, "title")
	}
	if merged.SortOrder != "asc" {
		t.Errorf("SortOrder = %q, want %q", merged.SortOrder, "asc")
	}
}

func TestMergeCollectionConfig_SortByOnly(t *testing.T) {
	inferred := InferCollection("blog") // date desc
	siteCfg := &config.CollectionSiteConfig{Sort: "order"}
	merged := MergeCollectionConfig(inferred, siteCfg)
	if merged.SortBy != "order" {
		t.Errorf("SortBy = %q, want %q", merged.SortBy, "order")
	}
	// SortOrder should remain from inferred
	if merged.SortOrder != "desc" {
		t.Errorf("SortOrder = %q, want %q (preserved from inferred)", merged.SortOrder, "desc")
	}
}

func TestMergeCollectionConfig_LayoutOverride(t *testing.T) {
	inferred := InferCollection("blog") // default layout
	siteCfg := &config.CollectionSiteConfig{Layout: "docs"}
	merged := MergeCollectionConfig(inferred, siteCfg)
	if merged.Layout != engine.LayoutDocs {
		t.Errorf("Layout = %q, want %q", merged.Layout, engine.LayoutDocs)
	}
}

func TestMergeCollectionConfig_FeedBoolPtr(t *testing.T) {
	inferred := InferCollection("blog") // Feed=true
	siteCfg := &config.CollectionSiteConfig{Feed: boolPtr(false)}
	merged := MergeCollectionConfig(inferred, siteCfg)
	if merged.Feed {
		t.Error("Feed should be false after override")
	}
}

func TestMergeCollectionConfig_PaginateOverride(t *testing.T) {
	inferred := InferCollection("blog") // Paginate=10
	siteCfg := &config.CollectionSiteConfig{Paginate: 20}
	merged := MergeCollectionConfig(inferred, siteCfg)
	if merged.Paginate != 20 {
		t.Errorf("Paginate = %d, want 20", merged.Paginate)
	}
}

func TestMergeCollectionConfig_SidebarMerge(t *testing.T) {
	inferred := InferCollection("docs") // has sidebar defaults
	siteCfg := &config.CollectionSiteConfig{
		Sidebar: &config.CollectionSidebarConfig{
			CollapsedByDefault: boolPtr(true),
		},
	}
	merged := MergeCollectionConfig(inferred, siteCfg)
	if merged.Sidebar == nil {
		t.Fatal("Sidebar should not be nil")
	}
	if !merged.Sidebar.CollapsedByDefault {
		t.Error("CollapsedByDefault should be true after merge")
	}
	// Other sidebar values should be preserved from inferred
	if !merged.Sidebar.Collapsible {
		t.Error("Collapsible should still be true (from inferred)")
	}
}

func TestMergeCollectionConfig_PrevNextLabels(t *testing.T) {
	inferred := InferCollection("docs")
	siteCfg := &config.CollectionSiteConfig{
		PrevNext: &config.CollectionPrevNextConfig{
			Labels: []string{"← Back", "Forward →"},
		},
	}
	merged := MergeCollectionConfig(inferred, siteCfg)
	if merged.PrevNext.Labels[0] != "← Back" {
		t.Errorf("PrevNext.Labels[0] = %q, want %q", merged.PrevNext.Labels[0], "← Back")
	}
}

func TestParseSortString(t *testing.T) {
	tests := []struct {
		input     string
		wantBy    string
		wantOrder string
	}{
		{"date desc", "date", "desc"},
		{"order asc", "order", "asc"},
		{"title", "title", ""},
		{"", "", ""},
		{"slug desc", "slug", "desc"},
	}
	for _, tt := range tests {
		by, order := parseSortString(tt.input)
		if by != tt.wantBy || order != tt.wantOrder {
			t.Errorf("parseSortString(%q) = (%q, %q), want (%q, %q)",
				tt.input, by, order, tt.wantBy, tt.wantOrder)
		}
	}
}
