package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunNewCourse(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	cmd := rootCmd
	cmd.SetArgs([]string{"new", "course", "intro-to-go"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("new course failed: %v", err)
	}

	courseDir := filepath.Join(dir, "content", "courses", "intro-to-go")

	indexPath := filepath.Join(courseDir, "_index.md")
	if _, err := os.Stat(indexPath); err != nil {
		t.Fatalf("_index.md not created: %v", err)
	}

	configPath := filepath.Join(courseDir, "config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("config.yaml not created: %v", err)
	}
	configStr := string(data)
	if !strings.Contains(configStr, "title: Intro To Go") {
		t.Errorf("config.yaml missing title, got: %s", configStr)
	}
	if !strings.Contains(configStr, "icon: book") {
		t.Errorf("config.yaml missing icon, got: %s", configStr)
	}
}

func TestRunNewCourse_AlreadyExists(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	os.MkdirAll(filepath.Join(dir, "content", "courses", "my-course"), 0o755)

	cmd := rootCmd
	cmd.SetArgs([]string{"new", "course", "my-course"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error for existing course, got nil")
	}
}

func TestRunNewLesson(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	// Create course first.
	cmd := rootCmd
	cmd.SetArgs([]string{"new", "course", "my-course"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("new course failed: %v", err)
	}

	// Create first lesson.
	cmd.SetArgs([]string{"new", "lesson", "my-course", "hello-world"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("new lesson 1 failed: %v", err)
	}

	// Create second lesson.
	cmd.SetArgs([]string{"new", "lesson", "my-course", "variables"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("new lesson 2 failed: %v", err)
	}

	courseDir := filepath.Join(dir, "content", "courses", "my-course")

	if _, err := os.Stat(filepath.Join(courseDir, "01-hello-world.md")); err != nil {
		t.Error("01-hello-world.md not created")
	}
	if _, err := os.Stat(filepath.Join(courseDir, "02-variables.md")); err != nil {
		t.Error("02-variables.md not created")
	}
}

func TestRunNewLesson_CourseNotFound(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	cmd := rootCmd
	cmd.SetArgs([]string{"new", "lesson", "nonexistent", "intro"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error for nonexistent course, got nil")
	}
}

func TestRunNewLesson_GapInNumbering(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	cmd := rootCmd
	cmd.SetArgs([]string{"new", "course", "my-course"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("new course failed: %v", err)
	}

	courseDir := filepath.Join(dir, "content", "courses", "my-course")
	os.WriteFile(filepath.Join(courseDir, "05-foo.md"), []byte("---\ntitle: Foo\n---\n"), 0o644)

	cmd.SetArgs([]string{"new", "lesson", "my-course", "bar"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("new lesson failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(courseDir, "06-bar.md")); err != nil {
		t.Error("expected 06-bar.md (max+1), not created")
	}
}

func TestRunNew_FlatCommand(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	cmd := rootCmd
	cmd.SetArgs([]string{"new", "blog", "My First Post"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("new blog failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "content", "blog", "my-first-post.md")); err != nil {
		t.Error("my-first-post.md not created")
	}
}
