package collection

import (
	"testing"

	"github.com/getsarde/sarde/internal/engine"
)

// The catalog's collection_types name lists must mirror the inference
// maps in infer.go — Studio uses the catalog for name-based layout
// inference and must agree with the engine.
func TestCatalog_CollectionTypeNamesMatchInference(t *testing.T) {
	cat, err := engine.LoadFrontmatterCatalog()
	if err != nil {
		t.Fatalf("loading frontmatter catalog: %v", err)
	}

	cases := []struct {
		typeName string
		inferred map[string]bool
	}{
		{"blog", blogNames},
		{"docs", docsNames},
		{"slides", slidesNames},
		{"labs", labsNames},
	}

	for _, c := range cases {
		ct, ok := cat.CollectionTypes[c.typeName]
		if !ok {
			t.Errorf("catalog collection_types is missing %q", c.typeName)
			continue
		}
		catNames := map[string]bool{}
		for _, name := range ct.Names {
			catNames[name] = true
		}
		for name := range c.inferred {
			if !catNames[name] {
				t.Errorf("collection name %q is in infer.go's %s names but missing from the catalog", name, c.typeName)
			}
		}
		for name := range catNames {
			if !c.inferred[name] {
				t.Errorf("collection name %q is in the catalog's %s names but missing from infer.go", name, c.typeName)
			}
		}
	}
}
