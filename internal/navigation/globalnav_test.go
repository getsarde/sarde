package navigation

import (
	"testing"

	"github.com/getsarde/sarde/internal/engine"
)

func TestBuildGlobalNav_Basic(t *testing.T) {
	blog := &engine.Collection{Name: "blog", Title: "Blog"}
	docs := &engine.Collection{Name: "docs", Title: "Documentation"}

	site := &engine.SiteContext{
		Collections: map[string]*engine.Collection{
			"blog": blog,
			"docs": docs,
		},
	}

	nav := BuildGlobalNav(site, docs, nil, "")

	if nav == nil {
		t.Fatal("expected non-nil")
	}
	if len(nav.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(nav.Items))
	}

	// Alphabetical order: blog, docs.
	if nav.Items[0].Label != "Blog" {
		t.Errorf("item 0: got %q", nav.Items[0].Label)
	}
	if nav.Items[0].IsActive {
		t.Error("blog should not be is-active")
	}
	if nav.Items[1].Label != "Documentation" {
		t.Errorf("item 1: got %q", nav.Items[1].Label)
	}
	if !nav.Items[1].IsActive {
		t.Error("docs should be is-active")
	}
}

func TestBuildGlobalNav_NilCurrentCollection(t *testing.T) {
	site := &engine.SiteContext{
		Collections: map[string]*engine.Collection{
			"blog": {Name: "blog", Title: "Blog"},
		},
	}

	nav := BuildGlobalNav(site, nil, nil, "")
	if nav == nil {
		t.Fatal("expected non-nil")
	}
	if nav.Items[0].IsActive {
		t.Error("no collection should be is-active")
	}
}

func TestBuildGlobalNav_NilSite(t *testing.T) {
	if BuildGlobalNav(nil, nil, nil, "") != nil {
		t.Error("expected nil for nil site")
	}
}

// On a translated page the header names each collection in the page's
// language.
func TestBuildGlobalNav_TranslatedLabels(t *testing.T) {
	docs := &engine.Collection{Name: "docs", Title: "Documentation", LangIndex: map[string]*engine.Page{
		"ar": {PageIdentity: engine.PageIdentity{Title: "التوثيق"}},
	}}
	site := &engine.SiteContext{Collections: map[string]*engine.Collection{"docs": docs}}
	if got := BuildGlobalNav(site, docs, nil, "ar").Items[0].Label; got != "التوثيق" {
		t.Errorf("label on an Arabic page = %q, want the Arabic title", got)
	}
	if got := BuildGlobalNav(site, docs, nil, "en").Items[0].Label; got != "Documentation" {
		t.Errorf("label on an English page = %q, want the shared title", got)
	}
}
