package taxonomy

import (
	"testing"

	"github.com/getsarde/sarde/internal/engine"
)

// lookupFixture holds a hashed non-Latin term, a term re-keyed under a
// custom permalink ("C++" at cpp), and a plain term that owns the slug the
// custom one would have had ("C" at c).
func lookupFixture() *engine.Taxonomy {
	jp := TermSlug("日本語")
	return &engine.Taxonomy{
		Name: "tags",
		Terms: map[string]*engine.TaxonomyTerm{
			jp:    {Name: "日本語", Slug: jp, Permalink: "/tags/" + jp + "/"},
			"cpp": {Name: "C++", Slug: "cpp", CustomSlug: "cpp", Permalink: "/tags/cpp/"},
			"c":   {Name: "C", Slug: "c", Permalink: "/tags/c/"},
		},
	}
}

func TestLookupTerm(t *testing.T) {
	tax := lookupFixture()
	tests := []struct {
		name string
		want string // the term's permalink, "" for nil
	}{
		{"日本語", "/tags/" + TermSlug("日本語") + "/"},
		{"C++", "/tags/cpp/"},
		{"C", "/tags/c/"},
		// "C#" was never a term; its slug "c" belongs to C, which is what
		// a page tagged C# links to (they collide on the slug).
		{"C#", "/tags/c/"},
		{"Rust", ""},
	}
	for _, tt := range tests {
		got := LookupTerm(tax, tt.name)
		if tt.want == "" {
			if got != nil {
				t.Errorf("LookupTerm(%q) = %q, want nil", tt.name, got.Permalink)
			}
			continue
		}
		if got == nil || got.Permalink != tt.want {
			t.Errorf("LookupTerm(%q) = %v, want %s", tt.name, got, tt.want)
		}
	}
	if LookupTerm(nil, "C") != nil {
		t.Error("a nil taxonomy has no terms")
	}
}

// The term a non-Latin tag is stored under is the key LookupTerm tries.
func TestBuildTaxonomies_NonLatinTermFoundByName(t *testing.T) {
	page := &engine.Page{PageTaxonomy: engine.PageTaxonomy{Tags: []string{"日本語"}}}
	taxes, _ := BuildTaxonomies([]*engine.Page{page}, nil, "")
	term := LookupTerm(taxes["tags"], "日本語")
	if term == nil || len(term.Pages) != 1 {
		t.Fatalf("LookupTerm after BuildTaxonomies = %v", term)
	}
}
