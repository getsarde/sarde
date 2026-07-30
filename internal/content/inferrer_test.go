package content

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/getsarde/sarde/internal/engine"
)

func TestInfer_TitleFromH1(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	os.WriteFile(path, []byte("# Hello World\nContent here.\n"), 0644)

	page := &engine.Page{PageContent: engine.PageContent{RawContent: "# Hello World\nContent here.\n"}}
	inf := &Inferrer{}
	if err := inf.Infer(page, path); err != nil {
		t.Fatal(err)
	}
	if page.Title != "Hello World" {
		t.Errorf("Title = %q, want %q", page.Title, "Hello World")
	}
}

func TestInfer_TitleFromFilename(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "getting-started.md")
	os.WriteFile(path, []byte("No heading here.\n"), 0644)

	page := &engine.Page{PageContent: engine.PageContent{RawContent: "No heading here.\n"}}
	inf := &Inferrer{}
	if err := inf.Infer(page, path); err != nil {
		t.Fatal(err)
	}
	if page.Title != "Getting Started" {
		t.Errorf("Title = %q, want %q", page.Title, "Getting Started")
	}
}

func TestInfer_TitlePreservesExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	os.WriteFile(path, []byte("# Markdown Title\n"), 0644)

	page := &engine.Page{PageIdentity: engine.PageIdentity{Title: "Explicit Title"}, PageContent: engine.PageContent{RawContent: "# Markdown Title\n"}}
	inf := &Inferrer{}
	inf.Infer(page, path)
	if page.Title != "Explicit Title" {
		t.Errorf("Title = %q, want %q", page.Title, "Explicit Title")
	}
}

func TestInfer_DateFromMtime(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	os.WriteFile(path, []byte("content"), 0644)

	page := &engine.Page{PageContent: engine.PageContent{RawContent: "content"}}
	inf := &Inferrer{}
	inf.Infer(page, path)

	if page.Date.IsZero() {
		t.Error("Date should be set from file mtime")
	}
	if page.Updated.IsZero() {
		t.Error("Updated should be set from file mtime")
	}
}

func TestInfer_DatePreservesExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	os.WriteFile(path, []byte("content"), 0644)

	explicit := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	page := &engine.Page{PageIdentity: engine.PageIdentity{Date: explicit, Updated: explicit}, PageContent: engine.PageContent{RawContent: "content"}}
	inf := &Inferrer{}
	inf.Infer(page, path)

	if !page.Date.Equal(explicit) {
		t.Errorf("Date = %v, want %v", page.Date, explicit)
	}
}

func TestInfer_WeightFromPrefix(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "03-advanced.md")
	os.WriteFile(path, []byte("# Advanced\n"), 0644)

	page := &engine.Page{PageContent: engine.PageContent{RawContent: "# Advanced\n"}}
	inf := &Inferrer{}
	inf.Infer(page, path)

	if page.Sidebar.Order != 3 {
		t.Errorf("Weight = %d, want 3", page.Sidebar.Order)
	}
	if page.Slug != "advanced" {
		t.Errorf("Slug = %q, want %q", page.Slug, "advanced")
	}
}

func TestInfer_SlugFromFilename(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "my-awesome-post.md")
	os.WriteFile(path, []byte("content"), 0644)

	page := &engine.Page{PageContent: engine.PageContent{RawContent: "content"}}
	inf := &Inferrer{}
	inf.Infer(page, path)

	if page.Slug != "my-awesome-post" {
		t.Errorf("Slug = %q, want %q", page.Slug, "my-awesome-post")
	}
}

func TestInfer_SlugPreservesExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "01-basics.md")
	os.WriteFile(path, []byte("content"), 0644)

	page := &engine.Page{PageIdentity: engine.PageIdentity{Slug: "custom-slug"}, PageContent: engine.PageContent{RawContent: "content"}}
	inf := &Inferrer{}
	inf.Infer(page, path)

	if page.Slug != "custom-slug" {
		t.Errorf("Slug = %q, want %q", page.Slug, "custom-slug")
	}
}

func TestInfer_DatePrefixFilename(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "2024-03-15-hello-world.md")
	os.WriteFile(path, []byte("content"), 0644)

	page := &engine.Page{PageContent: engine.PageContent{RawContent: "content"}}
	inf := &Inferrer{}
	inf.Infer(page, path)

	want := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)
	if !page.Date.Equal(want) {
		t.Errorf("Date = %v, want %v", page.Date, want)
	}
	if page.Slug != "hello-world" {
		t.Errorf("Slug = %q, want %q", page.Slug, "hello-world")
	}
	if !page.DateExplicit {
		t.Error("a filename date prefix is a deliberate choice: DateExplicit should be true")
	}
}

func TestInfer_MtimeDateIsNotExplicit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "plain-page.md")
	os.WriteFile(path, []byte("content"), 0644)

	page := &engine.Page{PageContent: engine.PageContent{RawContent: "content"}}
	inf := &Inferrer{}
	inf.Infer(page, path)

	if page.Date.IsZero() {
		t.Fatal("expected mtime fallback to populate Date")
	}
	if page.DateExplicit {
		t.Error("an mtime-inferred date must not be marked explicit")
	}
}

func TestInfer_DatePrefixFrontmatterWins(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "2024-03-15-hello.md")
	os.WriteFile(path, []byte("content"), 0644)

	explicit := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	page := &engine.Page{PageIdentity: engine.PageIdentity{Date: explicit}, PageContent: engine.PageContent{RawContent: "content"}}
	inf := &Inferrer{}
	inf.Infer(page, path)

	if !page.Date.Equal(explicit) {
		t.Errorf("Date = %v, want %v (frontmatter must win)", page.Date, explicit)
	}
}

func TestInfer_DatePrefixWithNumericRemainder(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "2024-01-15-01-intro.md")
	os.WriteFile(path, []byte("content"), 0644)

	page := &engine.Page{PageContent: engine.PageContent{RawContent: "content"}}
	inf := &Inferrer{}
	inf.Infer(page, path)

	want := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	if !page.Date.Equal(want) {
		t.Errorf("Date = %v, want %v", page.Date, want)
	}
	if page.Slug != "intro" {
		t.Errorf("Slug = %q, want %q", page.Slug, "intro")
	}
	if page.Sidebar.Order != 1 {
		t.Errorf("Weight = %d, want 1", page.Sidebar.Order)
	}
}

func TestInfer_IndexMdSlugFromParentDir(t *testing.T) {
	dir := t.TempDir()
	docsDir := filepath.Join(dir, "docs")
	os.MkdirAll(docsDir, 0755)
	path := filepath.Join(docsDir, "_index.md")
	os.WriteFile(path, []byte("# Docs\n"), 0644)

	page := &engine.Page{PageContent: engine.PageContent{RawContent: "# Docs\n"}}
	inf := &Inferrer{}
	inf.Infer(page, path)

	if page.Slug != "docs" {
		t.Errorf("Slug = %q, want %q", page.Slug, "docs")
	}
}

// show_updated controls whether the timestamp is shown, not whether it is
// resolved. Zeroing Updated would also strip sitemap lastmod, SEO
// dateModified, and feed timestamps for that page.
func TestInfer_ShowUpdatedFalseStillResolvesUpdated(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	os.WriteFile(path, []byte("content"), 0644)

	page := &engine.Page{
		PageContent: engine.PageContent{RawContent: "content"},
		Params:      map[string]any{"show_updated": false},
	}
	inf := &Inferrer{}
	inf.Infer(page, path)

	if page.Updated.IsZero() {
		t.Error("Updated should still be resolved as data when show_updated is false")
	}
	if page.ShowUpdated() {
		t.Error("ShowUpdated should report false so the badge stays hidden")
	}
}
