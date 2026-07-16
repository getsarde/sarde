package collection

import (
	"os"
	"path/filepath"
	"testing"
)

// createEffectiveFixtureSite creates a minimal project dir with the given
// sarde.yaml body and one content subdirectory per name in collections.
func createEffectiveFixtureSite(t *testing.T, sardeYAML string, collections ...string) string {
	t.Helper()
	dir := t.TempDir()

	os.MkdirAll(filepath.Join(dir, "content"), 0o755)
	for _, name := range collections {
		os.MkdirAll(filepath.Join(dir, "content", name), 0o755)
	}
	os.WriteFile(filepath.Join(dir, "sarde.yaml"), []byte(sardeYAML), 0o644)

	return dir
}

const minimalSardeYAML = "site:\n  title: \"T\"\n  url: \"http://localhost:3000\"\n"

func findCollection(t *testing.T, cols []EffectiveCollection, name string) EffectiveCollection {
	t.Helper()
	for _, c := range cols {
		if c.Name == name {
			return c
		}
	}
	t.Fatalf("collection %q not found in result (%d collections)", name, len(cols))
	return EffectiveCollection{}
}

func TestBuildEffectiveConfig_PureInference(t *testing.T) {
	dir := createEffectiveFixtureSite(t, minimalSardeYAML, "guides")

	cols, err := BuildEffectiveConfig(dir)
	if err != nil {
		t.Fatalf("BuildEffectiveConfig: %v", err)
	}

	c := findCollection(t, cols, "guides")
	if c.InferredType != "docs" {
		t.Errorf("InferredType = %q, want %q", c.InferredType, "docs")
	}
	if c.Layout.Value != "docs" || c.Layout.Source != SourceInferred {
		t.Errorf("Layout = %+v, want {docs inferred}", c.Layout)
	}
	if c.SortBy.Value != "order" || c.SortBy.Source != SourceInferred {
		t.Errorf("SortBy = %+v, want {order inferred}", c.SortBy)
	}
	if c.Sidebar == nil {
		t.Fatal("Sidebar = nil, want populated for docs bucket")
	}
	for label, fv := range map[string]FieldValue{
		"sidebar.collapsible":          c.Sidebar.Collapsible,
		"sidebar.collapsed_by_default": c.Sidebar.CollapsedByDefault,
		"sidebar.max_depth":            c.Sidebar.MaxDepth,
		"sidebar.search":               c.Sidebar.Search,
	} {
		if fv.Source != SourceInferred {
			t.Errorf("%s source = %q, want inferred", label, fv.Source)
		}
	}
	if c.TOC == nil {
		t.Fatal("TOC = nil, want populated for docs bucket")
	}
	if c.PrevNext == nil {
		t.Fatal("PrevNext = nil, want populated for docs bucket")
	}
	labels, ok := c.PrevNext.Labels.Value.([]string)
	if !ok || len(labels) != 2 || labels[0] != "Previous" || labels[1] != "Next" {
		t.Errorf("PrevNext.Labels = %+v, want [Previous Next]", c.PrevNext.Labels.Value)
	}
	if c.PrevNext.Labels.Source != SourceInferred {
		t.Errorf("PrevNext.Labels source = %q, want inferred", c.PrevNext.Labels.Source)
	}
}

func TestBuildEffectiveConfig_SardeYAMLOverride(t *testing.T) {
	yaml := minimalSardeYAML + `collections:
  blog:
    sort: "title asc"
`
	dir := createEffectiveFixtureSite(t, yaml, "blog")

	cols, err := BuildEffectiveConfig(dir)
	if err != nil {
		t.Fatalf("BuildEffectiveConfig: %v", err)
	}

	c := findCollection(t, cols, "blog")
	if c.InferredType != "blog" {
		t.Errorf("InferredType = %q, want %q", c.InferredType, "blog")
	}
	if c.SortBy.Value != "title" || c.SortBy.Source != SourceSardeYAML {
		t.Errorf("SortBy = %+v, want {title sarde_yaml}", c.SortBy)
	}
	if c.SortOrder.Value != "asc" || c.SortOrder.Source != SourceSardeYAML {
		t.Errorf("SortOrder = %+v, want {asc sarde_yaml}", c.SortOrder)
	}
	if c.Feed.Value != true || c.Feed.Source != SourceInferred {
		t.Errorf("Feed = %+v, want {true inferred}", c.Feed)
	}
	if c.Sidebar != nil {
		t.Errorf("Sidebar = %+v, want nil for blog bucket without override", c.Sidebar)
	}
}

func TestBuildEffectiveConfig_PartialSortOverride(t *testing.T) {
	yaml := minimalSardeYAML + `collections:
  blog:
    sort: "order"
`
	dir := createEffectiveFixtureSite(t, yaml, "blog")

	cols, err := BuildEffectiveConfig(dir)
	if err != nil {
		t.Fatalf("BuildEffectiveConfig: %v", err)
	}

	c := findCollection(t, cols, "blog")
	if c.SortBy.Value != "order" || c.SortBy.Source != SourceSardeYAML {
		t.Errorf("SortBy = %+v, want {order sarde_yaml}", c.SortBy)
	}
	if c.SortOrder.Value != "desc" || c.SortOrder.Source != SourceInferred {
		t.Errorf("SortOrder = %+v, want {desc inferred}", c.SortOrder)
	}
}

func TestBuildEffectiveConfig_PermalinkFallbackFromTopLevelMap(t *testing.T) {
	yaml := minimalSardeYAML + `permalinks:
  guides: "/:slug/"
`
	dir := createEffectiveFixtureSite(t, yaml, "guides")

	cols, err := BuildEffectiveConfig(dir)
	if err != nil {
		t.Fatalf("BuildEffectiveConfig: %v", err)
	}

	c := findCollection(t, cols, "guides")
	if c.Permalink.Value != "/:slug/" || c.Permalink.Source != SourceSardeYAML {
		t.Errorf("Permalink = %+v, want {/:slug/ sarde_yaml}", c.Permalink)
	}
}

func TestBuildEffectiveConfig_CollectionOnlyInYAML(t *testing.T) {
	yaml := minimalSardeYAML + `collections:
  archive:
    paginate: 5
`
	dir := createEffectiveFixtureSite(t, yaml) // no content/archive dir

	cols, err := BuildEffectiveConfig(dir)
	if err != nil {
		t.Fatalf("BuildEffectiveConfig: %v", err)
	}

	c := findCollection(t, cols, "archive")
	if c.InferredType != "default" {
		t.Errorf("InferredType = %q, want %q", c.InferredType, "default")
	}
	if c.Paginate.Value != 5 || c.Paginate.Source != SourceSardeYAML {
		t.Errorf("Paginate = %+v, want {5 sarde_yaml}", c.Paginate)
	}
}

func TestBuildEffectiveConfig_NestedOverrides(t *testing.T) {
	yaml := minimalSardeYAML + `collections:
  docs:
    feed: false
    tabs: true
    sidebar:
      max_depth: 2
    toc:
      enabled: false
`
	dir := createEffectiveFixtureSite(t, yaml, "docs")

	cols, err := BuildEffectiveConfig(dir)
	if err != nil {
		t.Fatalf("BuildEffectiveConfig: %v", err)
	}

	c := findCollection(t, cols, "docs")
	if c.Feed.Value != false || c.Feed.Source != SourceSardeYAML {
		t.Errorf("Feed = %+v, want {false sarde_yaml} (explicit false must not read as inferred)", c.Feed)
	}
	if c.Tabs.Value != true || c.Tabs.Source != SourceSardeYAML {
		t.Errorf("Tabs = %+v, want {true sarde_yaml}", c.Tabs)
	}
	if c.Sidebar == nil || c.Sidebar.MaxDepth.Value != 2 || c.Sidebar.MaxDepth.Source != SourceSardeYAML {
		t.Errorf("Sidebar.MaxDepth = %+v, want {2 sarde_yaml}", c.Sidebar)
	}
	if c.Sidebar.Collapsible.Value != true || c.Sidebar.Collapsible.Source != SourceInferred {
		t.Errorf("Sidebar.Collapsible = %+v, want {true inferred}", c.Sidebar.Collapsible)
	}
	if c.TOC == nil || c.TOC.Enabled.Value != false || c.TOC.Enabled.Source != SourceSardeYAML {
		t.Errorf("TOC.Enabled = %+v, want {false sarde_yaml}", c.TOC)
	}
	if c.TOC.Depth.Value != 4 || c.TOC.Depth.Source != SourceInferred {
		t.Errorf("TOC.Depth = %+v, want {4 inferred}", c.TOC.Depth)
	}
}

func TestEnumerateCollectionNames_SkipsHiddenAndUnderscore(t *testing.T) {
	dir := createEffectiveFixtureSite(t, minimalSardeYAML, "docs", "_partials", ".hidden")

	cols, err := BuildEffectiveConfig(dir)
	if err != nil {
		t.Fatalf("BuildEffectiveConfig: %v", err)
	}
	if len(cols) != 1 || cols[0].Name != "docs" {
		names := make([]string, len(cols))
		for i, c := range cols {
			names[i] = c.Name
		}
		t.Errorf("collections = %v, want [docs]", names)
	}
}

func TestInferredBucket(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"blog", "blog"},
		{"posts", "blog"},
		{"docs", "docs"},
		{"guides", "docs"},
		{"tutorials", "docs"},
		{"projects", "default"},
	}
	for _, tt := range tests {
		if got := inferredBucket(tt.name); got != tt.want {
			t.Errorf("inferredBucket(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}
