package navigation

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/coderoo-dev/coderoo/internal/engine"
)

func writeNavYAML(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, "nav.yaml")
	os.MkdirAll(dir, 0o755)
	os.WriteFile(path, []byte(content), 0o644)
	return path
}

func testCollection() *engine.Collection {
	return &engine.Collection{
		Name:   "docs",
		Title:  "Documentation",
		Config: docsConfig(),
		Pages: []*engine.Page{
			{Title: "Getting Started", Slug: "getting-started", RelPermalink: "/docs/getting-started/", Weight: 1, Kind: engine.KindPage},
			{Title: "Installation", Slug: "installation", RelPermalink: "/docs/installation/", Weight: 2, Kind: engine.KindPage},
			{Title: "Authentication", Slug: "authentication", RelPermalink: "/docs/guides/authentication/", Weight: 1, Kind: engine.KindPage},
		},
	}
}

func TestBuildNavTreeFromYAML_BasicStructure(t *testing.T) {
	dir := t.TempDir()
	path := writeNavYAML(t, dir, `
- label: "Getting Started"
  items:
    - page: getting-started
    - page: installation
- label: "Guides"
  items:
    - page: guides/authentication
`)

	col := testCollection()
	tree, err := BuildNavTreeFromYAML(path, col)
	if err != nil {
		t.Fatal(err)
	}

	if len(tree.Root.Children) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(tree.Root.Children))
	}

	gs := tree.Root.Children[0]
	if gs.Label != "Getting Started" {
		t.Errorf("group 1 label: got %q", gs.Label)
	}
	if len(gs.Children) != 2 {
		t.Fatalf("expected 2 children in group 1, got %d", len(gs.Children))
	}
	if gs.Children[0].Page.Slug != "getting-started" {
		t.Errorf("child 1 slug: got %q", gs.Children[0].Page.Slug)
	}

	guides := tree.Root.Children[1]
	if len(guides.Children) != 1 {
		t.Fatalf("expected 1 child in Guides, got %d", len(guides.Children))
	}
}

func TestBuildNavTreeFromYAML_ExternalLink(t *testing.T) {
	dir := t.TempDir()
	path := writeNavYAML(t, dir, `
- label: "GitHub"
  url: "https://github.com/example"
  external: true
`)

	col := testCollection()
	tree, err := BuildNavTreeFromYAML(path, col)
	if err != nil {
		t.Fatal(err)
	}

	node := tree.Root.Children[0]
	if node.Label != "GitHub" {
		t.Errorf("label: got %q", node.Label)
	}
	if node.URL != "https://github.com/example" {
		t.Errorf("URL: got %q", node.URL)
	}
	if node.Attrs["target"] != "_blank" {
		t.Errorf("expected target=_blank, got %q", node.Attrs["target"])
	}
}

func TestBuildNavTreeFromYAML_PageLabelFromTitle(t *testing.T) {
	dir := t.TempDir()
	path := writeNavYAML(t, dir, `
- page: getting-started
`)

	col := testCollection()
	tree, err := BuildNavTreeFromYAML(path, col)
	if err != nil {
		t.Fatal(err)
	}

	node := tree.Root.Children[0]
	if node.Label != "Getting Started" {
		t.Errorf("expected page title as label, got %q", node.Label)
	}
}

func TestBuildNavTreeFromYAML_MissingPage(t *testing.T) {
	dir := t.TempDir()
	path := writeNavYAML(t, dir, `
- page: nonexistent
- page: getting-started
`)

	col := testCollection()
	tree, err := BuildNavTreeFromYAML(path, col)
	if err != nil {
		t.Fatal(err)
	}

	// Missing page should be skipped.
	if len(tree.Root.Children) != 1 {
		t.Fatalf("expected 1 child (missing page skipped), got %d", len(tree.Root.Children))
	}
}

func TestBuildNavTreeFromYAML_Badge_ScalarWithLegacyColor(t *testing.T) {
	dir := t.TempDir()
	path := writeNavYAML(t, dir, `
- page: getting-started
  badge: "New"
  badge_color: "green"
`)

	col := testCollection()
	tree, err := BuildNavTreeFromYAML(path, col)
	if err != nil {
		t.Fatal(err)
	}

	page := tree.Root.Children[0].Page
	if page.Badge.Text != "New" {
		t.Errorf("badge text: got %q", page.Badge.Text)
	}
	if page.Badge.Variant != engine.BadgeVariantTip {
		t.Errorf("badge variant: got %q, want %q (green→tip alias)", page.Badge.Variant, engine.BadgeVariantTip)
	}
}

func TestBuildNavTreeFromYAML_Badge_ScalarOnly(t *testing.T) {
	dir := t.TempDir()
	path := writeNavYAML(t, dir, `
- page: getting-started
  badge: "Beta"
`)

	col := testCollection()
	tree, err := BuildNavTreeFromYAML(path, col)
	if err != nil {
		t.Fatal(err)
	}

	page := tree.Root.Children[0].Page
	if page.Badge.Text != "Beta" {
		t.Errorf("badge text: got %q", page.Badge.Text)
	}
	if page.Badge.Variant != engine.BadgeVariantDefault {
		t.Errorf("badge variant: got %q, want %q", page.Badge.Variant, engine.BadgeVariantDefault)
	}
}

func TestBuildNavTreeFromYAML_Badge_Mapping(t *testing.T) {
	dir := t.TempDir()
	path := writeNavYAML(t, dir, `
- page: getting-started
  badge:
    text: "WIP"
    variant: "caution"
`)

	col := testCollection()
	tree, err := BuildNavTreeFromYAML(path, col)
	if err != nil {
		t.Fatal(err)
	}

	page := tree.Root.Children[0].Page
	if page.Badge.Text != "WIP" {
		t.Errorf("badge text: got %q", page.Badge.Text)
	}
	if page.Badge.Variant != engine.BadgeVariantCaution {
		t.Errorf("badge variant: got %q, want %q", page.Badge.Variant, engine.BadgeVariantCaution)
	}
}

func TestBuildNavTreeFromYAML_FlatList(t *testing.T) {
	dir := t.TempDir()
	path := writeNavYAML(t, dir, `
- label: "Group"
  items:
    - page: getting-started
    - page: installation
`)

	col := testCollection()
	tree, err := BuildNavTreeFromYAML(path, col)
	if err != nil {
		t.Fatal(err)
	}

	if tree.TotalPages != 2 {
		t.Errorf("expected 2 flat pages, got %d", tree.TotalPages)
	}
}
