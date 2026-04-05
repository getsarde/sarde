package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadWriteContentFile(t *testing.T) {
	dir := t.TempDir()
	contentDir := filepath.Join(dir, "content")
	os.MkdirAll(contentDir, 0o755)

	fm := map[string]any{"title": "Hello", "draft": true}
	body := "# Hello\n\nThis is content.\n"

	err := writeContentFile(contentDir, "blog/hello.md", fm, body)
	if err != nil {
		t.Fatalf("writeContentFile failed: %v", err)
	}

	// Verify file exists.
	absPath := filepath.Join(contentDir, "blog", "hello.md")
	if _, err := os.Stat(absPath); err != nil {
		t.Fatalf("file not created: %v", err)
	}

	// Read it back.
	gotFM, gotBody, err := readContentFile(contentDir, "blog/hello.md")
	if err != nil {
		t.Fatalf("readContentFile failed: %v", err)
	}

	if gotFM["title"] != "Hello" {
		t.Errorf("title = %v, want Hello", gotFM["title"])
	}
	if !strings.Contains(gotBody, "# Hello") {
		t.Errorf("body missing # Hello")
	}
}

func TestValidateContentPath(t *testing.T) {
	dir := t.TempDir()
	contentDir := filepath.Join(dir, "content")
	os.MkdirAll(contentDir, 0o755)

	tests := []struct {
		path  string
		valid bool
	}{
		{"blog/hello.md", true},
		{"docs/getting-started.md", true},
		{"../etc/passwd", false},
		{"../../secret.md", false},
		{"", false},
	}

	for _, tt := range tests {
		err := validateContentPath(contentDir, tt.path)
		if tt.valid && err != nil {
			t.Errorf("validateContentPath(%q) = %v, want nil", tt.path, err)
		}
		if !tt.valid && err == nil {
			t.Errorf("validateContentPath(%q) = nil, want error", tt.path)
		}
	}
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		title string
		want  string
	}{
		{"Hello World", "hello-world"},
		{"My First Post!", "my-first-post"},
		{"  Spaces  ", "spaces"},
		{"Already-slugified", "already-slugified"},
		{"UPPER CASE", "upper-case"},
	}

	for _, tt := range tests {
		got := slugify(tt.title)
		if got != tt.want {
			t.Errorf("slugify(%q) = %q, want %q", tt.title, got, tt.want)
		}
	}
}

func TestScaffoldFrontmatter(t *testing.T) {
	fm := scaffoldFrontmatter("My Post")

	if fm["title"] != "My Post" {
		t.Errorf("title = %v, want My Post", fm["title"])
	}
	if fm["draft"] != true {
		t.Errorf("draft = %v, want true", fm["draft"])
	}
	if fm["date"] == nil {
		t.Error("date should be set")
	}
}

func TestExtractSummary(t *testing.T) {
	fm := map[string]any{
		"title": "Test Post",
		"draft": true,
		"date":  "2025-06-01T00:00:00Z",
	}
	body := "This is a test body with some words in it."

	title, draft, date, _, wc, rt := extractSummary(fm, body)

	if title != "Test Post" {
		t.Errorf("title = %q", title)
	}
	if !draft {
		t.Error("expected draft = true")
	}
	if date.IsZero() {
		t.Error("expected non-zero date")
	}
	if wc == 0 {
		t.Error("expected non-zero word count")
	}
	if rt == 0 {
		t.Error("expected non-zero reading time")
	}
}
