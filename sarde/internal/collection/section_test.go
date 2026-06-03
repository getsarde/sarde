package collection

import (
	"testing"

	"github.com/frostybee/sarde/internal/engine"
)

func TestBuildSectionTree_Flat(t *testing.T) {
	pages := []*engine.Page{
		{PageIdentity: engine.PageIdentity{Title: "Docs", Slug: "docs", Kind: engine.KindSection, RelPermalink: "/docs/"}},
		{PageIdentity: engine.PageIdentity{Title: "Getting Started", Slug: "getting-started", Kind: engine.KindPage, RelPermalink: "/docs/getting-started/"}},
		{PageIdentity: engine.PageIdentity{Title: "API", Slug: "api", Kind: engine.KindPage, RelPermalink: "/docs/api/"}},
	}
	roots := BuildSectionTree(pages, "docs")
	if len(roots) != 1 {
		t.Fatalf("roots = %d, want 1", len(roots))
	}
	root := roots[0]
	if root.Title != "Docs" {
		t.Errorf("root.Title = %q, want %q", root.Title, "Docs")
	}
	if len(root.Pages) != 2 {
		t.Errorf("root.Pages = %d, want 2", len(root.Pages))
	}
}

func TestBuildSectionTree_Nested(t *testing.T) {
	pages := []*engine.Page{
		{PageIdentity: engine.PageIdentity{Title: "Docs", Slug: "docs", Kind: engine.KindSection, RelPermalink: "/docs/"}},
		{PageIdentity: engine.PageIdentity{Title: "Guides", Slug: "guides", Kind: engine.KindSection, RelPermalink: "/docs/guides/"}},
		{PageIdentity: engine.PageIdentity{Title: "Auth", Slug: "auth", Kind: engine.KindPage, RelPermalink: "/docs/guides/auth/"}},
	}
	roots := BuildSectionTree(pages, "docs")
	if len(roots) != 1 {
		t.Fatalf("roots = %d, want 1", len(roots))
	}
	root := roots[0]
	if len(root.Sections) != 1 {
		t.Fatalf("root.Sections = %d, want 1", len(root.Sections))
	}
	guides := root.Sections[0]
	if guides.Title != "Guides" {
		t.Errorf("subsection.Title = %q, want %q", guides.Title, "Guides")
	}
	if guides.Parent != root {
		t.Error("guides.Parent should be root")
	}
	if len(guides.Pages) != 1 {
		t.Errorf("guides.Pages = %d, want 1", len(guides.Pages))
	}
}

func TestBuildSectionTree_TransparentSection(t *testing.T) {
	pages := []*engine.Page{
		{PageIdentity: engine.PageIdentity{Title: "Docs", Slug: "docs", Kind: engine.KindSection, RelPermalink: "/docs/"}},
		{PageIdentity: engine.PageIdentity{Title: "Internal", Slug: "internal", Kind: engine.KindSection, RelPermalink: "/docs/internal/"},
			Params: map[string]any{"transparent": true}},
		{PageIdentity: engine.PageIdentity{Title: "Hidden Page", Slug: "hidden", Kind: engine.KindPage, RelPermalink: "/docs/internal/hidden/"}},
	}
	roots := BuildSectionTree(pages, "docs")
	root := roots[0]
	// Transparent section's pages should be hoisted to parent
	if len(root.Pages) != 1 {
		t.Errorf("root.Pages = %d, want 1 (hoisted from transparent)", len(root.Pages))
	}
}

func TestBuildSectionTree_RenderFalse(t *testing.T) {
	pages := []*engine.Page{
		{PageIdentity: engine.PageIdentity{Title: "Docs", Slug: "docs", Kind: engine.KindSection, RelPermalink: "/docs/"}},
		{PageIdentity: engine.PageIdentity{Title: "Group Only", Slug: "group", Kind: engine.KindSection, RelPermalink: "/docs/group/"},
			Params: map[string]any{"render": false}},
	}
	roots := BuildSectionTree(pages, "docs")
	root := roots[0]
	if len(root.Sections) != 1 {
		t.Fatalf("root.Sections = %d, want 1", len(root.Sections))
	}
	if root.Sections[0].Render {
		t.Error("section with render:false should have Render=false")
	}
}

func TestBuildSectionTree_IndexPage(t *testing.T) {
	indexPage := &engine.Page{PageIdentity: engine.PageIdentity{Title: "Docs", Slug: "docs", Kind: engine.KindSection, RelPermalink: "/docs/"}}
	pages := []*engine.Page{indexPage}
	roots := BuildSectionTree(pages, "docs")
	if roots[0].IndexPage != indexPage {
		t.Error("IndexPage should be set")
	}
}

// findSection returns the child section with the given slug, or nil.
func findSection(sections []*engine.Section, slug string) *engine.Section {
	for _, s := range sections {
		if s.Slug == slug {
			return s
		}
	}
	return nil
}

func TestBuildSectionTree_InferredSection(t *testing.T) {
	// "reference" and "reference/api" have no _index.md; "guide" does.
	guideIndex := &engine.Page{PageIdentity: engine.PageIdentity{Title: "Guide", Slug: "guide", Kind: engine.KindSection, RelPermalink: "/docs/guide/"}}
	pages := []*engine.Page{
		{PageIdentity: engine.PageIdentity{Title: "Docs", Slug: "docs", Kind: engine.KindSection, RelPermalink: "/docs/"}},
		guideIndex,
		{PageIdentity: engine.PageIdentity{Title: "Intro", Slug: "intro", Kind: engine.KindPage, RelPermalink: "/docs/guide/intro/"}},
		{PageIdentity: engine.PageIdentity{Title: "Limits", Slug: "limits", Kind: engine.KindPage, RelPermalink: "/docs/reference/api/limits/"}},
	}
	roots := BuildSectionTree(pages, "docs")
	if len(roots) != 1 {
		t.Fatalf("roots = %d, want 1", len(roots))
	}
	root := roots[0]

	// Explicit _index.md sibling keeps its IndexPage and title.
	guide := findSection(root.Sections, "guide")
	if guide == nil {
		t.Fatal("guide section missing")
	}
	if guide.IndexPage != guideIndex {
		t.Error("guide should keep its explicit IndexPage")
	}
	if guide.Title != "Guide" {
		t.Errorf("guide.Title = %q, want %q", guide.Title, "Guide")
	}

	// Inferred top-level section.
	ref := findSection(root.Sections, "reference")
	if ref == nil {
		t.Fatal("inferred reference section missing")
	}
	if ref.IndexPage != nil {
		t.Error("inferred reference should have nil IndexPage")
	}
	if ref.Render {
		t.Error("inferred reference should have Render=false")
	}
	if ref.Title != "Reference" {
		t.Errorf("reference.Title = %q, want %q", ref.Title, "Reference")
	}

	// Nested inferred section, with the leaf page assigned to it.
	api := findSection(ref.Sections, "api")
	if api == nil {
		t.Fatal("inferred reference/api section missing")
	}
	if api.IndexPage != nil {
		t.Error("inferred api should have nil IndexPage")
	}
	if len(api.Pages) != 1 || api.Pages[0].Slug != "limits" {
		t.Errorf("api.Pages = %v, want [limits]", api.Pages)
	}
}

func TestSectionDir(t *testing.T) {
	tests := []struct {
		permalink, collection, want string
	}{
		{"/docs/", "docs", ""},
		{"/docs/guides/", "docs", "guides"},
		{"/docs/guides/advanced/", "docs", "guides/advanced"},
	}
	for _, tt := range tests {
		got := sectionDir(tt.permalink, tt.collection)
		if got != tt.want {
			t.Errorf("sectionDir(%q, %q) = %q, want %q", tt.permalink, tt.collection, got, tt.want)
		}
	}
}
