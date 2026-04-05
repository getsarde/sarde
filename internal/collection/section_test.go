package collection

import (
	"testing"

	"github.com/coderoo-dev/coderoo/internal/engine"
)

func TestBuildSectionTree_Flat(t *testing.T) {
	pages := []*engine.Page{
		{Title: "Docs", Slug: "docs", Kind: engine.KindSection, RelPermalink: "/docs/"},
		{Title: "Getting Started", Slug: "getting-started", Kind: engine.KindPage, RelPermalink: "/docs/getting-started/"},
		{Title: "API", Slug: "api", Kind: engine.KindPage, RelPermalink: "/docs/api/"},
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
		{Title: "Docs", Slug: "docs", Kind: engine.KindSection, RelPermalink: "/docs/"},
		{Title: "Guides", Slug: "guides", Kind: engine.KindSection, RelPermalink: "/docs/guides/"},
		{Title: "Auth", Slug: "auth", Kind: engine.KindPage, RelPermalink: "/docs/guides/auth/"},
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
		{Title: "Docs", Slug: "docs", Kind: engine.KindSection, RelPermalink: "/docs/"},
		{Title: "Internal", Slug: "internal", Kind: engine.KindSection, RelPermalink: "/docs/internal/",
			Params: map[string]any{"transparent": true}},
		{Title: "Hidden Page", Slug: "hidden", Kind: engine.KindPage, RelPermalink: "/docs/internal/hidden/"},
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
		{Title: "Docs", Slug: "docs", Kind: engine.KindSection, RelPermalink: "/docs/"},
		{Title: "Group Only", Slug: "group", Kind: engine.KindSection, RelPermalink: "/docs/group/",
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
	indexPage := &engine.Page{Title: "Docs", Slug: "docs", Kind: engine.KindSection, RelPermalink: "/docs/"}
	pages := []*engine.Page{indexPage}
	roots := BuildSectionTree(pages, "docs")
	if roots[0].IndexPage != indexPage {
		t.Error("IndexPage should be set")
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
