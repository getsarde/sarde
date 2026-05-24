package content

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/frostybee/sarde/internal/engine"
)

func TestInfer_TitleFromH1(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	os.WriteFile(path, []byte("# Hello World\nContent here.\n"), 0644)

	page := &engine.Page{RawContent: "# Hello World\nContent here.\n"}
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

	page := &engine.Page{RawContent: "No heading here.\n"}
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

	page := &engine.Page{Title: "Explicit Title", RawContent: "# Markdown Title\n"}
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

	page := &engine.Page{RawContent: "content"}
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
	page := &engine.Page{Date: explicit, Updated: explicit, RawContent: "content"}
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

	page := &engine.Page{RawContent: "# Advanced\n"}
	inf := &Inferrer{}
	inf.Infer(page, path)

	if page.Weight != 3 {
		t.Errorf("Weight = %d, want 3", page.Weight)
	}
	if page.Slug != "advanced" {
		t.Errorf("Slug = %q, want %q", page.Slug, "advanced")
	}
}

func TestInfer_SlugFromFilename(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "my-awesome-post.md")
	os.WriteFile(path, []byte("content"), 0644)

	page := &engine.Page{RawContent: "content"}
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

	page := &engine.Page{Slug: "custom-slug", RawContent: "content"}
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

	page := &engine.Page{RawContent: "content"}
	inf := &Inferrer{}
	inf.Infer(page, path)

	want := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)
	if !page.Date.Equal(want) {
		t.Errorf("Date = %v, want %v", page.Date, want)
	}
	if page.Slug != "hello-world" {
		t.Errorf("Slug = %q, want %q", page.Slug, "hello-world")
	}
}

func TestInfer_DatePrefixFrontmatterWins(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "2024-03-15-hello.md")
	os.WriteFile(path, []byte("content"), 0644)

	explicit := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	page := &engine.Page{Date: explicit, RawContent: "content"}
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

	page := &engine.Page{RawContent: "content"}
	inf := &Inferrer{}
	inf.Infer(page, path)

	want := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	if !page.Date.Equal(want) {
		t.Errorf("Date = %v, want %v", page.Date, want)
	}
	if page.Slug != "intro" {
		t.Errorf("Slug = %q, want %q", page.Slug, "intro")
	}
	if page.Weight != 1 {
		t.Errorf("Weight = %d, want 1", page.Weight)
	}
}

func TestInfer_IndexMdSlugFromParentDir(t *testing.T) {
	dir := t.TempDir()
	docsDir := filepath.Join(dir, "docs")
	os.MkdirAll(docsDir, 0755)
	path := filepath.Join(docsDir, "_index.md")
	os.WriteFile(path, []byte("# Docs\n"), 0644)

	page := &engine.Page{RawContent: "# Docs\n"}
	inf := &Inferrer{}
	inf.Infer(page, path)

	if page.Slug != "docs" {
		t.Errorf("Slug = %q, want %q", page.Slug, "docs")
	}
}
