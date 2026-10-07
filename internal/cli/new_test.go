package cli

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/getsarde/sarde/internal/sitetemplate"
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

// fakeTemplate makes runNewSite scaffold from an in-memory template instead
// of downloading one. It returns the spec the command asked for.
func fakeTemplate(t *testing.T, files map[string]string) *sitetemplate.Spec {
	t.Helper()
	m := fstest.MapFS{}
	for name, body := range files {
		m[name] = &fstest.MapFile{Data: []byte(body)}
	}
	var got sitetemplate.Spec
	prev := fetchSiteTemplate
	fetchSiteTemplate = func(_ context.Context, spec sitetemplate.Spec) (*sitetemplate.Result, error) {
		got = spec
		return &sitetemplate.Result{
			FS:      sitetemplate.HideRootFiles(m, "README.md", "LICENSE"),
			Ref:     "main",
			Cleanup: func() {},
		}, nil
	}
	t.Cleanup(func() { fetchSiteTemplate = prev })
	return &got
}

// courseLike is the shape of the course template folder in the templates
// repository: its own kazari config, a Pages workflow, and repo-level files
// that must not end up in the site.
var courseLike = map[string]string{
	"README.md":          "# course template\n",
	"LICENSE":            "MIT\n",
	"sarde.yaml":         "site:\n  title: Course\n",
	"kazari.config.yaml": strings.Replace(kazariConfigContent, "themeToggleButton: false", "themeToggleButton: true", 1),
	"content/_index.md":  "---\ntitle: Home\n---\n",
	"content/courses/python-essentials/_index.md": "---\ntitle: Python\n---\n",
	".github/workflows/deploy.yml":                "on: push\n",
}

func TestRunNewSite_Template(t *testing.T) {
	spec := fakeTemplate(t, courseLike)
	site, err := runNewSiteIn(t, "course")
	if err != nil {
		t.Fatalf("new site --template course failed: %v", err)
	}
	if !spec.Official || spec.Owner != "getsarde" || spec.Repo != "sarde-templates" || spec.Subpath != "course" {
		t.Errorf("fetched spec = %+v, want the registry entry for course", *spec)
	}

	assertExists(t, site, "sarde.yaml", "content/_index.md", "content/courses/python-essentials/_index.md",
		".github/workflows/deploy.yml",
		// Added by the scaffold because the template has no copy.
		".gitignore", "public/images/hero-light.svg", "public/images/hero-dark.svg")
	assertMissing(t, site, "README.md", "LICENSE", "content/blog")

	// The template's own kazari config wins over the scaffold's.
	data, err := os.ReadFile(filepath.Join(site, "kazari.config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "themeToggleButton: true") {
		t.Errorf("template kazari.config.yaml was overwritten by the scaffold copy")
	}
}

// A template that ships a GitHub Pages workflow fails on every push until
// Pages builds from Actions, so the CLI says how to enable it or remove it.
func TestRunNewSite_TemplateWorkflowHint(t *testing.T) {
	newSite := func(template string) string {
		t.Helper()
		dir := t.TempDir()
		origWd, _ := os.Getwd()
		os.Chdir(dir)
		defer os.Chdir(origWd)
		// rootCmd is shared across tests, so reset the flags other tests set.
		rootCmd.SetArgs([]string{"new", "site", "site", "--template=" + template, "--quiet=false"})
		out, err := captureStdout(t, rootCmd.Execute)
		if err != nil {
			t.Fatalf("new site --template=%q failed: %v", template, err)
		}
		return out
	}

	fakeTemplate(t, courseLike)
	out := newSite("course")
	for _, want := range []string{".github/workflows/deploy.yml", "Settings > Pages", `from the "course" template`} {
		if !strings.Contains(out, want) {
			t.Errorf("course scaffold output missing %q:\n%s", want, out)
		}
	}

	without := map[string]string{}
	for k, v := range courseLike {
		if k != ".github/workflows/deploy.yml" {
			without[k] = v
		}
	}
	fakeTemplate(t, without)
	if out := newSite("course"); strings.Contains(out, "GitHub Pages") {
		t.Errorf("a template without a workflow should not mention Pages:\n%s", out)
	}

	if out := newSite(""); strings.Contains(out, "GitHub Pages") {
		t.Errorf("default scaffold should not mention the workflow:\n%s", out)
	}
}

func TestRunNewSite_TemplateExistingFile(t *testing.T) {
	fakeTemplate(t, courseLike)
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)
	os.MkdirAll(filepath.Join(dir, "site", "content"), 0o755)
	os.WriteFile(filepath.Join(dir, "site", "content", "_index.md"), []byte("mine"), 0o644)

	rootCmd.SetArgs([]string{"new", "site", "site", "--template=course", "--quiet"})
	err := rootCmd.Execute()
	if !errors.Is(err, fs.ErrExist) || !strings.Contains(err.Error(), "writing course template") {
		t.Errorf("expected an fs.ErrExist for the pre-existing file, got %v", err)
	}
}

func TestRunNewSite_TemplateFetchError(t *testing.T) {
	prev := fetchSiteTemplate
	fetchSiteTemplate = func(context.Context, sitetemplate.Spec) (*sitetemplate.Result, error) {
		return nil, errors.New("boom")
	}
	t.Cleanup(func() { fetchSiteTemplate = prev })

	site, err := runNewSiteIn(t, "course")
	if err == nil || !strings.HasPrefix(err.Error(), "fetching template course: boom") {
		t.Fatalf("error = %v", err)
	}
	assertMissing(t, site, "sarde.yaml", "content")
}

func TestRunNewSite_InvalidTemplate(t *testing.T) {
	// An unknown name is rejected offline: the fetcher must never run.
	prev := fetchSiteTemplate
	fetchSiteTemplate = func(context.Context, sitetemplate.Spec) (*sitetemplate.Result, error) {
		t.Fatal("fetcher called for an unknown template name")
		return nil, nil
	}
	t.Cleanup(func() { fetchSiteTemplate = prev })

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
