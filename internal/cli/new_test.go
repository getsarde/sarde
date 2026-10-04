package cli

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/getsarde/sarde/embedded"
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

// runNewSiteIn runs "sarde new site <name>" with the given --template value in
// a fresh temp dir and returns the site directory. --template is always passed
// explicitly because rootCmd keeps flag values between Execute calls.
func runNewSiteIn(t *testing.T, template string) (string, error) {
	t.Helper()
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	rootCmd.SetArgs([]string{"new", "site", "site", "--template=" + template, "--quiet"})
	err := rootCmd.Execute()
	return filepath.Join(dir, "site"), err
}

func assertExists(t *testing.T, dir string, rels ...string) {
	t.Helper()
	for _, rel := range rels {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(rel))); err != nil {
			t.Errorf("%s: %v", rel, err)
		}
	}
}

func assertMissing(t *testing.T, dir string, rels ...string) {
	t.Helper()
	for _, rel := range rels {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(rel))); err == nil {
			t.Errorf("%s should not exist", rel)
		}
	}
}

func TestRunNewSite_Default(t *testing.T) {
	site, err := runNewSiteIn(t, "")
	if err != nil {
		t.Fatalf("new site failed: %v", err)
	}
	assertExists(t, site, "sarde.yaml", "kazari.config.yaml", ".gitignore",
		"content/_index.md", "content/blog/hello-world.md", "content/docs/getting-started.md",
		"public/images/hero-light.svg", "public/images/hero-dark.svg")
	assertMissing(t, site, "content/courses")
}

// The next-step hint tells the user to cd into a site created in a subfolder,
// since running "sarde dev" from the parent fails.
func TestRunNewSite_NextStepHint(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"vanier", "Run 'cd vanier' then 'sarde dev' to start the dev server."},
		{"my site", `Run 'cd "my site"' then 'sarde dev' to start the dev server.`},
		{".", "  Run 'sarde dev' to start the dev server."},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			dir := t.TempDir()
			origWd, _ := os.Getwd()
			os.Chdir(dir)
			defer os.Chdir(origWd)

			// rootCmd keeps flag values between Execute calls; reset both.
			rootCmd.SetArgs([]string{"new", "site", tt.path, "--template=", "--quiet=false"})
			out, err := captureStdout(t, rootCmd.Execute)
			if err != nil {
				t.Fatalf("new site failed: %v", err)
			}
			if !strings.Contains(out, tt.want) {
				t.Errorf("output missing %q:\n%s", tt.want, out)
			}
		})
	}
}

func TestRunNewSite_CourseTemplate(t *testing.T) {
	site, err := runNewSiteIn(t, "course")
	if err != nil {
		t.Fatalf("new site --template course failed: %v", err)
	}

	tmpl, ok := embedded.SiteTemplate("course")
	if !ok {
		t.Fatal("course template not embedded")
	}
	count := 0
	err = fs.WalkDir(tmpl, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		count++
		want, _ := fs.ReadFile(tmpl, path)
		got, readErr := os.ReadFile(filepath.Join(site, filepath.FromSlash(path)))
		if readErr != nil {
			t.Errorf("%s: %v", path, readErr)
		} else if string(got) != string(want) {
			t.Errorf("%s differs from the embedded template", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if count < 20 {
		t.Errorf("course template has %d files, want the full site", count)
	}

	assertExists(t, site, "kazari.config.yaml", ".gitignore",
		"public/images/hero-light.svg", "public/images/hero-dark.svg",
		"content/courses/python-essentials/_index.md", "content/labs/web-fundamentals/hello-world/_index.md")
	assertMissing(t, site, "content/blog")
}

func TestRunNewSite_InvalidTemplate(t *testing.T) {
	site, err := runNewSiteIn(t, "bogus")
	if err == nil {
		t.Fatal("expected an error for an unknown template")
	}
	if !strings.Contains(err.Error(), `unknown template "bogus"`) || !strings.Contains(err.Error(), "course") {
		t.Errorf("error = %q, want the unknown name and the available templates", err)
	}
	assertMissing(t, site, "sarde.yaml")
}

// The scaffolded kazari.config.yaml must not set darkMode: Sarde applies its
// own dark-mode selector so code blocks follow the theme toggle, and a darkMode
// entry would only trigger a build warning.
func TestRunNewSite_KazariConfigLeavesDarkModeToSarde(t *testing.T) {
	site, err := runNewSiteIn(t, "")
	if err != nil {
		t.Fatalf("new site failed: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(site, "kazari.config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "darkMode:") {
			t.Errorf("kazari.config.yaml sets darkMode:\n%s", data)
		}
	}
}

// The course template ships its own kazari.config.yaml so it can turn on the
// per-block theme toggle. Everything else must match the shared scaffold
// config, so fixes to one (like the dark-mode change) reach the other.
func TestCourseTemplateKazariConfigTracksScaffold(t *testing.T) {
	tmpl, ok := embedded.SiteTemplate("course")
	if !ok {
		t.Fatal("course template not embedded")
	}
	data, err := fs.ReadFile(tmpl, "kazari.config.yaml")
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Replace(kazariConfigContent, "themeToggleButton: false", "themeToggleButton: true", 1)
	if want == kazariConfigContent {
		t.Fatal("scaffold kazari.config.yaml no longer has themeToggleButton: false; update this test")
	}
	if string(data) != want {
		t.Error("course template kazari.config.yaml differs from the scaffold config beyond themeToggleButton")
	}
}
