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

func TestRunNewDirective(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	cmd := rootCmd
	cmd.SetArgs([]string{"new", "directive", "pullquote"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("new directive failed: %v", err)
	}

	for _, ext := range []string{".yaml", ".html", ".css"} {
		p := filepath.Join(dir, "directives", "pullquote"+ext)
		if _, err := os.Stat(p); err != nil {
			t.Errorf("%s not created: %v", p, err)
		}
	}

	data, _ := os.ReadFile(filepath.Join(dir, "directives", "pullquote.yaml"))
	if !strings.Contains(string(data), "name: pullquote") {
		t.Errorf("yaml missing name, got: %s", data)
	}
	css, _ := os.ReadFile(filepath.Join(dir, "directives", "pullquote.css"))
	if !strings.Contains(string(css), "--sd-accent") {
		t.Errorf("css missing --sd-* token usage, got: %s", css)
	}
}

func TestRunNewDirective_RejectsBuiltinName(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	cmd := rootCmd
	cmd.SetArgs([]string{"new", "directive", "card"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error for built-in name card")
	}
}

func TestRunNewDirective_RejectsInvalidName(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	cmd := rootCmd
	cmd.SetArgs([]string{"new", "directive", "My_Thing"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error for invalid name")
	}
}

func TestRunNewDirective_AlreadyExists(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	os.MkdirAll(filepath.Join(dir, "directives"), 0o755)
	os.WriteFile(filepath.Join(dir, "directives", "pullquote.yaml"), []byte("name: pullquote\n"), 0o644)

	cmd := rootCmd
	cmd.SetArgs([]string{"new", "directive", "pullquote"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error when directive already exists")
	}
}
