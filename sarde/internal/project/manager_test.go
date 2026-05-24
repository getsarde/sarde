package project

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/frostybee/sarde/embedded"
)

func createTestProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	os.MkdirAll(filepath.Join(dir, "content", "blog"), 0o755)
	os.MkdirAll(filepath.Join(dir, "static"), 0o755)

	// sarde.yaml
	os.WriteFile(filepath.Join(dir, "sarde.yaml"), []byte("site:\n  title: \"Test Site\"\n  url: \"http://localhost:3000\"\n"), 0o644)

	// Content.
	os.WriteFile(filepath.Join(dir, "content", "_index.md"), []byte("---\ntitle: Home\n---\n# Welcome\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "content", "blog", "_index.md"), []byte("---\ntitle: Blog\n---\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "content", "blog", "hello.md"), []byte("---\ntitle: Hello\ndate: \"2025-01-01T00:00:00Z\"\n---\n# Hello\nFirst post.\n"), 0o644)

	return dir
}

func TestProjectManager_OpenClose(t *testing.T) {
	dir := createTestProject(t)
	hub := NewEventHub()
	pm := NewProjectManager(hub, embedded.ThemeFS(), nil)

	info, err := pm.OpenProject(dir)
	if err != nil {
		t.Fatalf("OpenProject failed: %v", err)
	}

	if info.Title != "Test Site" {
		t.Errorf("Title = %q, want Test Site", info.Title)
	}
	if pm.State() != StateOpen {
		t.Errorf("State = %v, want Open", pm.State())
	}

	if err := pm.CloseProject(); err != nil {
		t.Fatalf("CloseProject failed: %v", err)
	}
	if pm.State() != StateClosed {
		t.Errorf("State = %v, want Closed", pm.State())
	}
}

func TestProjectManager_CreateProject(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "new-site")
	hub := NewEventHub()
	pm := NewProjectManager(hub, embedded.ThemeFS(), nil)

	info, err := pm.CreateProject(dir, CreateOpts{Title: "My New Site"})
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	if info.Title != "My New Site" {
		t.Errorf("Title = %q, want My New Site", info.Title)
	}

	// sarde.yaml should exist.
	if _, err := os.Stat(filepath.Join(dir, "sarde.yaml")); err != nil {
		t.Error("sarde.yaml not created")
	}
	// content/_index.md should exist.
	if _, err := os.Stat(filepath.Join(dir, "content", "_index.md")); err != nil {
		t.Error("content/_index.md not created")
	}
}

func TestProjectManager_ContentCRUD(t *testing.T) {
	dir := createTestProject(t)
	hub := NewEventHub()
	pm := NewProjectManager(hub, embedded.ThemeFS(), nil)

	pm.OpenProject(dir)

	// List content.
	summaries, err := pm.ListContent("blog")
	if err != nil {
		t.Fatalf("ListContent failed: %v", err)
	}
	if len(summaries) == 0 {
		t.Fatal("expected at least 1 content item in blog")
	}

	// Read content.
	file, err := pm.ReadContent("blog/hello.md")
	if err != nil {
		t.Fatalf("ReadContent failed: %v", err)
	}
	if file.Title != "Hello" {
		t.Errorf("Title = %q, want Hello", file.Title)
	}

	// Create content.
	created, err := pm.CreateContent("blog", "New Post")
	if err != nil {
		t.Fatalf("CreateContent failed: %v", err)
	}
	if created.Title != "New Post" {
		t.Errorf("Title = %q, want New Post", created.Title)
	}
	if !created.Draft {
		t.Error("new content should be draft")
	}

	// Save content.
	err = pm.SaveContent("blog/new-post.md", map[string]any{"title": "Updated Post", "draft": false}, "# Updated\n")
	if err != nil {
		t.Fatalf("SaveContent failed: %v", err)
	}

	// Read back saved content.
	updated, _ := pm.ReadContent("blog/new-post.md")
	if updated.Title != "Updated Post" {
		t.Errorf("Title after save = %q, want Updated Post", updated.Title)
	}

	// Delete content.
	err = pm.DeleteContent("blog/new-post.md")
	if err != nil {
		t.Fatalf("DeleteContent failed: %v", err)
	}

	// Verify deleted.
	_, err = pm.ReadContent("blog/new-post.md")
	if err == nil {
		t.Error("expected error reading deleted file")
	}
}

func TestProjectManager_Build(t *testing.T) {
	dir := createTestProject(t)
	hub := NewEventHub()
	pm := NewProjectManager(hub, embedded.ThemeFS(), nil)

	pm.OpenProject(dir)

	result, err := pm.Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if result.PageCount == 0 {
		t.Error("expected at least 1 page")
	}

	// Verify output exists.
	distDir := filepath.Join(dir, "dist")
	if _, err := os.Stat(filepath.Join(distDir, "index.html")); err != nil {
		t.Error("index.html not found in output")
	}
}

func TestProjectManager_ClosedErrors(t *testing.T) {
	hub := NewEventHub()
	pm := NewProjectManager(hub, embedded.ThemeFS(), nil)

	// All operations should fail when no project is open.
	if _, err := pm.Build(); err == nil {
		t.Error("Build should fail when closed")
	}
	if _, err := pm.ListContent(""); err == nil {
		t.Error("ListContent should fail when closed")
	}
	if _, err := pm.ReadContent("test.md"); err == nil {
		t.Error("ReadContent should fail when closed")
	}
	if _, err := pm.CreateContent("blog", "Test"); err == nil {
		t.Error("CreateContent should fail when closed")
	}
}

func TestProjectManager_UpdateSettings(t *testing.T) {
	dir := createTestProject(t)
	hub := NewEventHub()
	pm := NewProjectManager(hub, embedded.ThemeFS(), nil)

	pm.OpenProject(dir)

	newTitle := "Updated Title"
	err := pm.UpdateSettings(SettingsInput{Title: &newTitle})
	if err != nil {
		t.Fatalf("UpdateSettings failed: %v", err)
	}

	cfg := pm.GetConfig()
	if cfg.Site.Title != "Updated Title" {
		t.Errorf("Title = %q, want Updated Title", cfg.Site.Title)
	}
}

func TestProjectManager_GetCollections(t *testing.T) {
	dir := createTestProject(t)
	hub := NewEventHub()
	pm := NewProjectManager(hub, embedded.ThemeFS(), nil)

	pm.OpenProject(dir)

	cols, err := pm.GetCollections()
	if err != nil {
		t.Fatalf("GetCollections failed: %v", err)
	}

	found := false
	for _, c := range cols {
		if c.Name == "blog" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected blog collection")
	}
}

func TestProjectManager_RenderMarkdown(t *testing.T) {
	hub := NewEventHub()
	pm := NewProjectManager(hub, embedded.ThemeFS(), nil)

	result, err := pm.RenderMarkdown("# Hello\n\nSome text here.")
	if err != nil {
		t.Fatalf("RenderMarkdown failed: %v", err)
	}

	if result.HTML == "" {
		t.Error("expected non-empty HTML")
	}
	if result.WordCount == 0 {
		t.Error("expected non-zero word count")
	}
}
