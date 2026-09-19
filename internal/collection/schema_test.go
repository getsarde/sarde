package collection

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/getsarde/sarde/internal/engine"
)

func TestResolveSchema_DocsGetsNestedSidebarAndTOC(t *testing.T) {
	dir := createEffectiveFixtureSite(t, minimalSardeYAML, "guides")

	rs, err := ResolveSchema(dir, "guides")
	if err != nil {
		t.Fatalf("ResolveSchema: %v", err)
	}
	if rs.Layout != "docs" || rs.Type != "docs" {
		t.Fatalf("layout/type = %s/%s, want docs/docs", rs.Layout, rs.Type)
	}
	for _, key := range []string{"sidebar", "toc"} {
		f, ok := rs.Fields[key]
		if !ok {
			t.Fatalf("docs collection missing nested %q field", key)
		}
		if f.Type != "object" || len(f.Fields) == 0 || f.Source != SchemaSourceCatalog {
			t.Errorf("%s = %+v, want object with children from catalog", key, f)
		}
	}
	if _, ok := rs.Fields["title"]; !ok {
		t.Error("docs collection missing identity field title")
	}
}

func TestResolveSchema_BlogExcludesDocsCategoriesButGetsTaxonomy(t *testing.T) {
	dir := createEffectiveFixtureSite(t, minimalSardeYAML, "posts")

	rs, err := ResolveSchema(dir, "posts")
	if err != nil {
		t.Fatalf("ResolveSchema: %v", err)
	}
	if rs.Type != "blog" {
		t.Fatalf("type = %s, want blog", rs.Type)
	}
	if _, ok := rs.Fields["sidebar"]; ok {
		t.Error("blog collection should not include the docs-only sidebar field")
	}
	if _, ok := rs.Fields["tags"]; !ok {
		t.Error("blog collection should get taxonomy fields via extra_categories")
	}
}

func TestResolveSchema_CustomFieldOverridesCatalog(t *testing.T) {
	dir := createEffectiveFixtureSite(t, minimalSardeYAML, "posts")
	configYAML := "frontmatter_schema:\n  fields:\n    author:\n      type: enum\n      options: [alice, bob]\n    rating:\n      type: int\n      min: 0\n      max: 5\n"
	os.WriteFile(filepath.Join(dir, "content", "posts", "config.yaml"), []byte(configYAML), 0o644)

	rs, err := ResolveSchema(dir, "posts")
	if err != nil {
		t.Fatalf("ResolveSchema: %v", err)
	}
	author := rs.Fields["author"]
	if author.Source != SchemaSourceCustom || author.Type != "enum" || len(author.Options) != 2 {
		t.Errorf("author = %+v, want custom enum with 2 options", author)
	}
	rating := rs.Fields["rating"]
	if rating.Source != SchemaSourceCustom || rating.Type != "int" || rating.Min == nil || *rating.Max != 5 {
		t.Errorf("rating = %+v, want custom int 0..5", rating)
	}
}

func TestResolveSchema_NoConfigYAMLIsCatalogOnly(t *testing.T) {
	dir := createEffectiveFixtureSite(t, minimalSardeYAML, "notes")

	rs, err := ResolveSchema(dir, "notes")
	if err != nil {
		t.Fatalf("ResolveSchema: %v", err)
	}
	if rs.Type != "default" {
		t.Fatalf("type = %s, want default", rs.Type)
	}
	for name, f := range rs.Fields {
		if f.Source != SchemaSourceCatalog {
			t.Errorf("field %s source = %s, want catalog only", name, f.Source)
		}
	}
}

func TestResolveSchema_UnknownCollection(t *testing.T) {
	dir := createEffectiveFixtureSite(t, minimalSardeYAML, "posts")

	_, err := ResolveSchema(dir, "nope")
	var uce *UnknownCollectionError
	if err == nil {
		t.Fatal("expected error for unknown collection")
	}
	if !errors.As(err, &uce) {
		t.Fatalf("error = %v, want UnknownCollectionError", err)
	}
}

func TestResolveSchema_LayoutOverrideChangesCategories(t *testing.T) {
	yaml := minimalSardeYAML + "collections:\n  notes:\n    layout: docs\n"
	dir := createEffectiveFixtureSite(t, yaml, "notes")

	rs, err := ResolveSchema(dir, "notes")
	if err != nil {
		t.Fatalf("ResolveSchema: %v", err)
	}
	if rs.Layout != "docs" {
		t.Fatalf("layout = %s, want docs (sarde.yaml override)", rs.Layout)
	}
	if _, ok := rs.Fields["sidebar"]; !ok {
		t.Error("layout override to docs should grant the sidebar field")
	}
}

func TestResolveSchema_ConfiguredButMissingDirIsKnown(t *testing.T) {
	yaml := minimalSardeYAML + "collections:\n  future:\n    layout: docs\n"
	dir := createEffectiveFixtureSite(t, yaml)

	rs, err := ResolveSchema(dir, "future")
	if err != nil {
		t.Fatalf("ResolveSchema on configured-but-missing dir: %v", err)
	}
	if rs.Layout != "docs" {
		t.Errorf("layout = %s, want docs", rs.Layout)
	}
}

// Every catalog-sourced field the resolver emits must trace back to the
// catalog — parity guard in the spirit of infer_catalog_test.go.
func TestResolveSchema_CatalogFieldsExistInCatalog(t *testing.T) {
	dir := createEffectiveFixtureSite(t, minimalSardeYAML, "docs", "posts", "labs", "notes")
	cat, err := engine.LoadFrontmatterCatalog()
	if err != nil {
		t.Fatal(err)
	}
	known := make(map[string]bool)
	for _, c := range cat.Categories {
		if c.Nested && c.ParentKey != "" {
			known[c.ParentKey] = true
			continue
		}
		for _, f := range c.Fields {
			known[f.Key] = true
		}
	}
	for _, name := range []string{"docs", "posts", "labs", "notes"} {
		rs, err := ResolveSchema(dir, name)
		if err != nil {
			t.Fatalf("ResolveSchema(%s): %v", name, err)
		}
		for key, f := range rs.Fields {
			if f.Source == SchemaSourceCatalog && !known[key] {
				t.Errorf("%s: resolved field %q not present in catalog", name, key)
			}
		}
	}
}
