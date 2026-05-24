package content

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/frostybee/sarde/internal/engine"
)

// createFixture builds a test content directory structure.
func createFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	content := filepath.Join(dir, "content")

	// Home page
	writeFile(t, content, "_index.md", "---\ntitle: Home\n---\n# Welcome\n")

	// Standalone page
	writeFile(t, content, "about.md", "---\ntitle: About\n---\nAbout page.\n")

	// Blog collection
	writeFile(t, content, filepath.Join("blog", "_index.md"), "---\ntitle: Blog\n---\n")
	writeFile(t, content, filepath.Join("blog", "hello-world.md"), "---\ntitle: Hello World\n---\nFirst post.\n")
	writeFile(t, content, filepath.Join("blog", "second-post.md"), "---\ntitle: Second Post\n---\nAnother post.\n")

	// Docs collection with nested section
	writeFile(t, content, filepath.Join("docs", "_index.md"), "---\ntitle: Docs\n---\n")
	writeFile(t, content, filepath.Join("docs", "getting-started.md"), "---\ntitle: Getting Started\n---\nIntro.\n")
	writeFile(t, content, filepath.Join("docs", "guides", "_index.md"), "---\ntitle: Guides\n---\n")
	writeFile(t, content, filepath.Join("docs", "guides", "authentication.md"), "---\ntitle: Auth\n---\nAuth guide.\n")

	// Courses with numeric prefixes
	writeFile(t, content, filepath.Join("courses", "_index.md"), "---\ntitle: Courses\n---\n")
	writeFile(t, content, filepath.Join("courses", "01-basics.md"), "---\ntitle: Basics\n---\n")
	writeFile(t, content, filepath.Join("courses", "02-advanced.md"), "---\ntitle: Advanced\n---\n")

	// Bundle (index.md with sibling asset)
	writeFile(t, content, filepath.Join("blog", "my-bundle", "index.md"), "---\ntitle: Bundle Post\n---\n")
	writeFileRaw(t, filepath.Join(content, "blog", "my-bundle", "hero.jpg"), []byte("fake-image"))

	return content
}

func writeFile(t *testing.T, base, rel, body string) {
	t.Helper()
	path := filepath.Join(base, rel)
	os.MkdirAll(filepath.Dir(path), 0755)
	os.WriteFile(path, []byte(body), 0644)
}

func writeFileRaw(t *testing.T, path string, data []byte) {
	t.Helper()
	os.MkdirAll(filepath.Dir(path), 0755)
	os.WriteFile(path, data, 0644)
}

func TestDiscoverFiles_Fixture(t *testing.T) {
	content := createFixture(t)
	scanner := &Scanner{}
	files, err := scanner.DiscoverFiles(content)
	if err != nil {
		t.Fatalf("DiscoverFiles error: %v", err)
	}

	// Build a lookup by relative path
	byRel := make(map[string]ContentFile)
	for _, f := range files {
		byRel[f.RelPath] = f
	}

	tests := []struct {
		rel        string
		kind       engine.NodeKind
		collection string
	}{
		{"_index.md", engine.KindHome, ""},
		{"about.md", engine.KindStandalone, ""},
		{"blog/_index.md", engine.KindSection, "blog"},
		{"blog/hello-world.md", engine.KindPage, "blog"},
		{"blog/second-post.md", engine.KindPage, "blog"},
		{"docs/_index.md", engine.KindSection, "docs"},
		{"docs/getting-started.md", engine.KindPage, "docs"},
		{"docs/guides/_index.md", engine.KindSection, "docs"},
		{"docs/guides/authentication.md", engine.KindPage, "docs"},
		{"courses/_index.md", engine.KindSection, "courses"},
		{"courses/01-basics.md", engine.KindPage, "courses"},
		{"courses/02-advanced.md", engine.KindPage, "courses"},
		{"blog/my-bundle/index.md", engine.KindBundle, "blog"},
	}

	for _, tt := range tests {
		f, ok := byRel[tt.rel]
		if !ok {
			t.Errorf("file %q not found in results", tt.rel)
			continue
		}
		if f.Kind != tt.kind {
			t.Errorf("%s: Kind = %q, want %q", tt.rel, f.Kind, tt.kind)
		}
		if f.CollectionName != tt.collection {
			t.Errorf("%s: CollectionName = %q, want %q", tt.rel, f.CollectionName, tt.collection)
		}
	}
}

func TestDiscoverFiles_NumericPrefix(t *testing.T) {
	content := createFixture(t)
	scanner := &Scanner{}
	files, err := scanner.DiscoverFiles(content)
	if err != nil {
		t.Fatalf("DiscoverFiles error: %v", err)
	}

	byRel := make(map[string]ContentFile)
	for _, f := range files {
		byRel[f.RelPath] = f
	}

	f := byRel["courses/01-basics.md"]
	if f.Slug != "basics" {
		t.Errorf("01-basics.md slug = %q, want %q", f.Slug, "basics")
	}
	if f.Weight != 1 {
		t.Errorf("01-basics.md weight = %d, want 1", f.Weight)
	}

	f = byRel["courses/02-advanced.md"]
	if f.Slug != "advanced" {
		t.Errorf("02-advanced.md slug = %q, want %q", f.Slug, "advanced")
	}
	if f.Weight != 2 {
		t.Errorf("02-advanced.md weight = %d, want 2", f.Weight)
	}
}

func TestDiscoverFiles_BundleAssets(t *testing.T) {
	content := createFixture(t)
	scanner := &Scanner{}
	files, err := scanner.DiscoverFiles(content)
	if err != nil {
		t.Fatalf("DiscoverFiles error: %v", err)
	}

	byRel := make(map[string]ContentFile)
	for _, f := range files {
		byRel[f.RelPath] = f
	}

	f := byRel["blog/my-bundle/index.md"]
	if !f.IsBundle {
		t.Error("my-bundle should be a bundle")
	}
	if len(f.BundleAssets) != 1 {
		t.Errorf("BundleAssets len = %d, want 1", len(f.BundleAssets))
	}
}

func TestDiscover_GroupsByCollection(t *testing.T) {
	content := createFixture(t)
	scanner := &Scanner{}
	result, err := scanner.Discover(content)
	if err != nil {
		t.Fatalf("Discover error: %v", err)
	}

	// Root-level files (home + standalone)
	if len(result[""]) != 2 {
		t.Errorf("root files = %d, want 2 (_index.md + about.md)", len(result[""]))
	}
	// Blog collection (section + 2 posts + bundle)
	if len(result["blog"]) != 4 {
		t.Errorf("blog files = %d, want 4", len(result["blog"]))
	}
	// Docs collection (section + page + nested section + nested page)
	if len(result["docs"]) != 4 {
		t.Errorf("docs files = %d, want 4", len(result["docs"]))
	}
	// Courses collection (section + 2 pages)
	if len(result["courses"]) != 3 {
		t.Errorf("courses files = %d, want 3", len(result["courses"]))
	}
}
