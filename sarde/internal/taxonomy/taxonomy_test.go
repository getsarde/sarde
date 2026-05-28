package taxonomy

import (
	"testing"

	"github.com/frostybee/sarde/internal/config"
	"github.com/frostybee/sarde/internal/engine"
)

func TestBuildTaxonomies_Tags(t *testing.T) {
	pages := []*engine.Page{
		{PageIdentity: engine.PageIdentity{Title: "Post A"}, PageTaxonomy: engine.PageTaxonomy{Tags: []string{"Go", "Testing"}}},
		{PageIdentity: engine.PageIdentity{Title: "Post B"}, PageTaxonomy: engine.PageTaxonomy{Tags: []string{"Go", "Performance"}}},
		{PageIdentity: engine.PageIdentity{Title: "Post C"}, PageTaxonomy: engine.PageTaxonomy{Tags: []string{"Testing"}}},
	}

	taxonomies := BuildTaxonomies(pages, nil)

	tags, ok := taxonomies["tags"]
	if !ok {
		t.Fatal("expected tags taxonomy")
	}

	goTerm := tags.Terms["go"]
	if goTerm == nil {
		t.Fatal("expected 'go' term")
	}
	if len(goTerm.Pages) != 2 {
		t.Errorf("go: expected 2 pages, got %d", len(goTerm.Pages))
	}
	if goTerm.Permalink != "/tags/go/" {
		t.Errorf("permalink: got %q", goTerm.Permalink)
	}

	testTerm := tags.Terms["testing"]
	if testTerm == nil {
		t.Fatal("expected 'testing' term")
	}
	if len(testTerm.Pages) != 2 {
		t.Errorf("testing: expected 2 pages, got %d", len(testTerm.Pages))
	}
}

func TestBuildTaxonomies_Categories(t *testing.T) {
	pages := []*engine.Page{
		{PageIdentity: engine.PageIdentity{Title: "Post A"}, PageTaxonomy: engine.PageTaxonomy{Categories: []string{"Tutorials"}}},
	}

	taxonomies := BuildTaxonomies(pages, nil)

	cats := taxonomies["categories"]
	if cats == nil {
		t.Fatal("expected categories taxonomy")
	}
	if len(cats.Terms) != 1 {
		t.Errorf("expected 1 term, got %d", len(cats.Terms))
	}
}

func TestBuildTaxonomies_EmptyRemoved(t *testing.T) {
	pages := []*engine.Page{
		{PageIdentity: engine.PageIdentity{Title: "Post A"}, PageTaxonomy: engine.PageTaxonomy{Tags: []string{"Go"}}},
		// No categories.
	}

	taxonomies := BuildTaxonomies(pages, nil)

	if _, ok := taxonomies["categories"]; ok {
		t.Error("expected empty categories taxonomy to be removed")
	}
}

func TestBuildTaxonomies_NoPages(t *testing.T) {
	taxonomies := BuildTaxonomies(nil, nil)
	if len(taxonomies) != 0 {
		t.Errorf("expected 0 taxonomies, got %d", len(taxonomies))
	}
}

func TestBuildTaxonomies_CustomTaxonomy(t *testing.T) {
	pages := []*engine.Page{
		{
			PageIdentity: engine.PageIdentity{Title: "Post A"},
			PageTaxonomy: engine.PageTaxonomy{
				Extra: map[string][]string{"authors": {"Alice", "Bob"}},
			},
		},
		{
			PageIdentity: engine.PageIdentity{Title: "Post B"},
			PageTaxonomy: engine.PageTaxonomy{
				Extra: map[string][]string{"authors": {"Alice"}},
			},
		},
	}

	taxCfg := map[string]config.TaxonomyConfig{
		"authors": {Singular: "author"},
	}
	taxonomies := BuildTaxonomies(pages, taxCfg)

	authors, ok := taxonomies["authors"]
	if !ok {
		t.Fatal("expected authors taxonomy")
	}
	if len(authors.Terms) != 2 {
		t.Errorf("expected 2 terms, got %d", len(authors.Terms))
	}
	alice := authors.Terms["alice"]
	if alice == nil {
		t.Fatal("expected 'alice' term")
	}
	if len(alice.Pages) != 2 {
		t.Errorf("alice: expected 2 pages, got %d", len(alice.Pages))
	}
	if alice.Permalink != "/authors/alice/" {
		t.Errorf("alice permalink: got %q", alice.Permalink)
	}
	bob := authors.Terms["bob"]
	if bob == nil {
		t.Fatal("expected 'bob' term")
	}
	if len(bob.Pages) != 1 {
		t.Errorf("bob: expected 1 page, got %d", len(bob.Pages))
	}
}

func TestBuildTaxonomies_MixedTagsAndCustom(t *testing.T) {
	pages := []*engine.Page{
		{
			PageIdentity: engine.PageIdentity{Title: "Post A"},
			PageTaxonomy: engine.PageTaxonomy{
				Tags:  []string{"Go"},
				Extra: map[string][]string{"authors": {"Alice"}},
			},
		},
	}

	taxCfg := map[string]config.TaxonomyConfig{
		"tags":    {Singular: "tag"},
		"authors": {Singular: "author"},
	}
	taxonomies := BuildTaxonomies(pages, taxCfg)

	if _, ok := taxonomies["tags"]; !ok {
		t.Error("expected tags taxonomy")
	}
	if _, ok := taxonomies["authors"]; !ok {
		t.Error("expected authors taxonomy")
	}
}

func TestBuildTaxonomies_EmptyCustomRemoved(t *testing.T) {
	pages := []*engine.Page{
		{
			PageIdentity: engine.PageIdentity{Title: "Post A"},
			PageTaxonomy: engine.PageTaxonomy{Tags: []string{"Go"}},
		},
	}

	taxCfg := map[string]config.TaxonomyConfig{
		"tags":    {Singular: "tag"},
		"authors": {Singular: "author"},
	}
	taxonomies := BuildTaxonomies(pages, taxCfg)

	if _, ok := taxonomies["authors"]; ok {
		t.Error("expected empty authors taxonomy to be removed")
	}
	if _, ok := taxonomies["tags"]; !ok {
		t.Error("expected tags taxonomy to remain")
	}
}
