package content

import (
	"testing"

	"github.com/getsarde/sarde/internal/engine"
)

// Layouts with a base-template mapping in internal/template/engine.go.
// split and presentation are declared in LayoutType but unimplemented,
// so the catalog must not offer them.
var implementedLayouts = map[string]bool{
	"default": true, "docs": true, "splash": true,
	"wide": true, "full": true, "centered": true,
}

func loadCatalog(t *testing.T) *engine.FrontmatterCatalog {
	t.Helper()
	cat, err := engine.LoadFrontmatterCatalog()
	if err != nil {
		t.Fatalf("loading frontmatter catalog: %v", err)
	}
	return cat
}

// catalogTopLevelKeys collects the top-level frontmatter keys the catalog
// describes: field keys of flat categories plus the parent_key of nested
// categories (sidebar, toc).
func catalogTopLevelKeys(t *testing.T, cat *engine.FrontmatterCatalog) map[string]bool {
	t.Helper()
	keys := map[string]bool{}
	for _, c := range cat.Categories {
		if c.Nested {
			if c.ParentKey == "" {
				t.Errorf("nested category %q has no parent_key", c.Name)
				continue
			}
			keys[c.ParentKey] = true
			continue
		}
		for _, f := range c.Fields {
			keys[f.Key] = true
		}
	}
	return keys
}

func TestCatalog_MatchesKnownFrontmatterKeys(t *testing.T) {
	cat := loadCatalog(t)
	catalogKeys := catalogTopLevelKeys(t, cat)

	for key := range knownFrontmatterKeys {
		if !catalogKeys[key] {
			t.Errorf("key %q is in knownFrontmatterKeys but missing from frontmatter_catalog.yaml", key)
		}
	}
	for key := range catalogKeys {
		if !knownFrontmatterKeys[key] {
			t.Errorf("key %q is in frontmatter_catalog.yaml but missing from knownFrontmatterKeys", key)
		}
	}
}

func TestCatalog_MatchesKnownNestedKeys(t *testing.T) {
	cat := loadCatalog(t)

	nestedKnown := map[string]map[string]bool{
		"sidebar": knownSidebarKeys,
		"toc":     knownTOCKeys,
	}

	seen := map[string]bool{}
	for _, c := range cat.Categories {
		if !c.Nested {
			continue
		}
		known, ok := nestedKnown[c.ParentKey]
		if !ok {
			t.Errorf("nested category %q has parent_key %q with no known-key set in the validator", c.Name, c.ParentKey)
			continue
		}
		seen[c.ParentKey] = true

		catKeys := map[string]bool{}
		for _, f := range c.Fields {
			catKeys[f.Key] = true
		}
		for key := range known {
			if !catKeys[key] {
				t.Errorf("key %q is in the validator's %s keys but missing from the catalog's %q category", key, c.ParentKey, c.Name)
			}
		}
		for key := range catKeys {
			if !known[key] {
				t.Errorf("key %q is in the catalog's %q category but missing from the validator's %s keys", key, c.Name, c.ParentKey)
			}
		}
	}
	for parentKey := range nestedKnown {
		if !seen[parentKey] {
			t.Errorf("validator defines nested keys for %q but the catalog has no matching nested category", parentKey)
		}
	}
}

func TestCatalog_LayoutsAreImplementedAndCategoriesExist(t *testing.T) {
	cat := loadCatalog(t)

	categories := map[string]bool{}
	for _, c := range cat.Categories {
		categories[c.Name] = true
	}

	for layout, cats := range cat.Layouts {
		if !implementedLayouts[layout] {
			t.Errorf("catalog layouts map contains %q, which is not an implemented layout", layout)
		}
		for _, name := range cats {
			if !categories[name] {
				t.Errorf("layout %q references unknown category %q", layout, name)
			}
		}
	}
	for layout := range implementedLayouts {
		if _, ok := cat.Layouts[layout]; !ok {
			t.Errorf("implemented layout %q is missing from the catalog layouts map", layout)
		}
	}

	for typeName, ct := range cat.CollectionTypes {
		for _, name := range ct.ExtraCategories {
			if !categories[name] {
				t.Errorf("collection type %q references unknown extra category %q", typeName, name)
			}
		}
		if _, ok := cat.Layouts[ct.Layout]; !ok {
			t.Errorf("collection type %q infers layout %q, which is not in the catalog layouts map", typeName, ct.Layout)
		}
	}
}
