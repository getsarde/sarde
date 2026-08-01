package engine_test

import (
	"testing"

	"github.com/getsarde/sarde/internal/content/markdown/extensions/aside"
	"github.com/getsarde/sarde/internal/engine"
)

func loadDirectiveCatalog(t *testing.T) *engine.DirectiveCatalog {
	t.Helper()
	cat, err := engine.LoadDirectiveCatalog()
	if err != nil {
		t.Fatalf("loading directive catalog: %v", err)
	}
	return cat
}

var validFieldTypes = engine.ValidDirectiveFieldTypes

var validPlacements = engine.ValidDirectiveFieldPlacements

func TestDirectiveCatalog_Structure(t *testing.T) {
	cat := loadDirectiveCatalog(t)

	if len(cat.Categories) == 0 {
		t.Fatal("directive catalog has no categories")
	}

	seen := map[string]bool{}
	for _, c := range cat.Categories {
		if c.Name == "" || c.Label == "" {
			t.Errorf("category %q missing name or label", c.Name)
		}
		for _, d := range c.Directives {
			if d.Name == "" {
				t.Errorf("category %q has a directive with no name", c.Name)
				continue
			}
			if seen[d.Name] {
				t.Errorf("duplicate directive name %q", d.Name)
			}
			seen[d.Name] = true

			if d.Kind != "callout" && d.Kind != "block" {
				t.Errorf("directive %q has invalid kind %q", d.Name, d.Kind)
			}
			if d.Label == "" || d.Description == "" {
				t.Errorf("directive %q missing label or description", d.Name)
			}
			if d.ChildTemplate != "" && d.ChildDefaultCount < 1 {
				t.Errorf("directive %q has child_template but no child_default_count", d.Name)
			}

			fieldNames := map[string]bool{}
			for _, f := range d.Fields {
				if f.Name == "" || f.Label == "" {
					t.Errorf("directive %q has a field missing name or label", d.Name)
				}
				if fieldNames[f.Name] {
					t.Errorf("directive %q has duplicate field %q", d.Name, f.Name)
				}
				fieldNames[f.Name] = true
				if !validFieldTypes[f.Type] {
					t.Errorf("directive %q field %q has invalid type %q", d.Name, f.Name, f.Type)
				}
				if !validPlacements[f.Placement] {
					t.Errorf("directive %q field %q has invalid placement %q", d.Name, f.Name, f.Placement)
				}
				if f.Type == "enum" && len(f.Options) == 0 {
					t.Errorf("directive %q enum field %q has no options", d.Name, f.Name)
				}
				if f.Default != "" && f.Type == "enum" {
					found := false
					for _, o := range f.Options {
						if o == f.Default {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("directive %q field %q default %q not in options", d.Name, f.Name, f.Default)
					}
				}
			}
			if d.ChildCountField != "" && !fieldNames[d.ChildCountField] {
				t.Errorf("directive %q child_count_field %q does not reference a field", d.Name, d.ChildCountField)
			}
		}
	}
}

// The callout entries must exactly mirror the aside parser's valid type set —
// a drifted catalog would let Studio generate directives that silently render
// as plain paragraphs.
func TestDirectiveCatalog_CalloutNamesMatchAside(t *testing.T) {
	cat := loadDirectiveCatalog(t)

	catNames := map[string]bool{}
	for _, c := range cat.Categories {
		for _, d := range c.Directives {
			if d.Kind == "callout" {
				catNames[d.Name] = true
			}
		}
	}

	for name := range aside.ValidTypes {
		if !catNames[name] {
			t.Errorf("aside.ValidTypes has %q but the directive catalog is missing it", name)
		}
	}
	for name := range catNames {
		if !aside.ValidTypes[name] {
			t.Errorf("directive catalog has callout %q, which is not in aside.ValidTypes", name)
		}
	}
}
