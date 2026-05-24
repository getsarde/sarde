package taxonomy

import (
	"testing"

	"github.com/frostybee/sarde/internal/engine"
)

func TestBuildTaxonomies_Tags(t *testing.T) {
	pages := []*engine.Page{
		{Title: "Post A", Tags: []string{"Go", "Testing"}},
		{Title: "Post B", Tags: []string{"Go", "Performance"}},
		{Title: "Post C", Tags: []string{"Testing"}},
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
		{Title: "Post A", Categories: []string{"Tutorials"}},
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
		{Title: "Post A", Tags: []string{"Go"}},
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
