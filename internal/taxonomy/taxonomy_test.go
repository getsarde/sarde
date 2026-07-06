package taxonomy

import (
	"strings"
	"testing"

	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/engine"
)

func TestBuildTaxonomies_Tags(t *testing.T) {
	pages := []*engine.Page{
		{PageIdentity: engine.PageIdentity{Title: "Post A"}, PageTaxonomy: engine.PageTaxonomy{Tags: []string{"Go", "Testing"}}},
		{PageIdentity: engine.PageIdentity{Title: "Post B"}, PageTaxonomy: engine.PageTaxonomy{Tags: []string{"Go", "Performance"}}},
		{PageIdentity: engine.PageIdentity{Title: "Post C"}, PageTaxonomy: engine.PageTaxonomy{Tags: []string{"Testing"}}},
	}

	taxonomies, warnings := BuildTaxonomies(pages, nil, "")
	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %v", warnings)
	}

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

	taxonomies, _ := BuildTaxonomies(pages, nil, "")

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

	taxonomies, _ := BuildTaxonomies(pages, nil, "")

	if _, ok := taxonomies["categories"]; ok {
		t.Error("expected empty categories taxonomy to be removed")
	}
}

func TestBuildTaxonomies_NoPages(t *testing.T) {
	taxonomies, _ := BuildTaxonomies(nil, nil, "")
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
	taxonomies, _ := BuildTaxonomies(pages, taxCfg, "")

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
	taxonomies, _ := BuildTaxonomies(pages, taxCfg, "")

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
	taxonomies, _ := BuildTaxonomies(pages, taxCfg, "")

	if _, ok := taxonomies["authors"]; ok {
		t.Error("expected empty authors taxonomy to be removed")
	}
	if _, ok := taxonomies["tags"]; !ok {
		t.Error("expected tags taxonomy to remain")
	}
}

func TestBuildTaxonomies_MultiLangScoping(t *testing.T) {
	// The same logical post, translated into three languages, all tagged "Go".
	pages := []*engine.Page{
		{PageIdentity: engine.PageIdentity{Title: "Post (en)"}, PageI18n: engine.PageI18n{Lang: "en"}, PageTaxonomy: engine.PageTaxonomy{Tags: []string{"Go"}}},
		{PageIdentity: engine.PageIdentity{Title: "Post (fr)"}, PageI18n: engine.PageI18n{Lang: "fr"}, PageTaxonomy: engine.PageTaxonomy{Tags: []string{"Go"}}},
		{PageIdentity: engine.PageIdentity{Title: "Post (ar)"}, PageI18n: engine.PageI18n{Lang: "ar"}, PageTaxonomy: engine.PageTaxonomy{Tags: []string{"Go"}}},
	}

	// Scoped to the default language: the post is listed once, not three times.
	scoped, _ := BuildTaxonomies(pages, nil, "en")
	if got := len(scoped["tags"].Terms["go"].Pages); got != 1 {
		t.Errorf("scoped to en: expected 1 page on 'go', got %d", got)
	}

	// No scoping (lang ""): every language variant is included (legacy behavior,
	// single-language sites where Lang is empty).
	all, _ := BuildTaxonomies(pages, nil, "")
	if got := len(all["tags"].Terms["go"].Pages); got != 3 {
		t.Errorf("unscoped: expected 3 pages on 'go', got %d", got)
	}
}

func TestAddTerm_CollisionWarning(t *testing.T) {
	tax := &engine.Taxonomy{
		Name:      "tags",
		Terms:     make(map[string]*engine.TaxonomyTerm),
		Permalink: "/tags/",
	}
	pageA := &engine.Page{PageIdentity: engine.PageIdentity{Title: "Post A"}}
	pageB := &engine.Page{PageIdentity: engine.PageIdentity{Title: "Post B"}}

	// "C++" and "C#" both slugify to "c".
	if w := addTerm(tax, "C++", pageA); w != "" {
		t.Fatalf("expected no warning for first term, got %q", w)
	}
	w := addTerm(tax, "C#", pageB)
	if w == "" {
		t.Fatal("expected a collision warning, got none")
	}
	if !strings.Contains(w, `"tags"`) || !strings.Contains(w, `"C++"`) || !strings.Contains(w, `"C#"`) || !strings.Contains(w, `"c"`) {
		t.Errorf("warning missing expected details: %q", w)
	}

	// The first term wins the slug; both pages are recorded against it.
	term, ok := tax.Terms["c"]
	if !ok {
		t.Fatal("expected term at slug 'c'")
	}
	if term.Name != "C++" {
		t.Errorf("expected surviving term name 'C++', got %q", term.Name)
	}
	if len(term.Pages) != 2 {
		t.Errorf("expected 2 pages on colliding slug, got %d", len(term.Pages))
	}
}

func TestAddTerm_EmptySlugFallback(t *testing.T) {
	tax := &engine.Taxonomy{
		Name:      "tags",
		Terms:     make(map[string]*engine.TaxonomyTerm),
		Permalink: "/tags/",
	}
	page := &engine.Page{PageIdentity: engine.PageIdentity{Title: "Post A"}}

	for _, termName := range []string{"@#$", "++", "日本語", "Привет"} {
		if w := addTerm(tax, termName, page); w != "" {
			t.Errorf("%q: unexpected warning %q", termName, w)
		}
	}

	if len(tax.Terms) != 4 {
		t.Fatalf("expected 4 distinct terms, got %d", len(tax.Terms))
	}
	for slug, term := range tax.Terms {
		if slug == "" {
			t.Errorf("term %q keyed under empty slug", term.Name)
		}
		wantPermalink := "/tags/" + slug + "/"
		if term.Permalink != wantPermalink {
			t.Errorf("term %q: expected permalink %q, got %q", term.Name, wantPermalink, term.Permalink)
		}
		if term.Permalink == tax.Permalink {
			t.Errorf("term %q permalink collides with taxonomy index permalink %q", term.Name, tax.Permalink)
		}
	}
}
