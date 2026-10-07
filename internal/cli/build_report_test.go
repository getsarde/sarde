package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/devlog"
	"github.com/getsarde/sarde/internal/engine"
)

func TestVerdictLine(t *testing.T) {
	tests := []struct {
		pages, total, inLog int
		want                string
	}{
		{48, 0, 0, "Built 48 pages in 312 ms"},
		{1, 1, 0, "Built 1 page in 312 ms, 1 warning"},
		{48, 3, 1, "Built 48 pages in 312 ms, 3 warnings (1 in the log above)"},
	}
	for _, tt := range tests {
		if got := verdictLine(tt.pages, 312*time.Millisecond, tt.total, tt.inLog); got != tt.want {
			t.Errorf("verdictLine(%d, %d, %d) = %q, want %q", tt.pages, tt.total, tt.inLog, got, tt.want)
		}
	}
}

func TestPrepareWarnings(t *testing.T) {
	r := newBuildReporter(&bytes.Buffer{}, t.TempDir(), false, false, nil)
	lines, listed := r.prepareWarnings([]engine.ValidationWarning{
		{File: "z.md", Field: "title", Message: "is required", Level: "error"},
		{File: "a.md", Field: "lint", Message: "line 14: heading level skipped", Line: 14},
		{File: "a.md", Field: "alert", Message: "unclosed shortcode"},
		{File: "a.md", Field: "alert", Message: "unclosed shortcode"},
		{File: "b.md", Field: "link", Message: "broken anchor: #x", Line: 3, Col: 2},
		{File: "sarde.yaml: plugins.config.x", Field: "plugin", Message: "unknown plugin"},
	})

	if listed != 5 {
		t.Errorf("listed = %d, want 5 (link warning skipped, duplicates counted)", listed)
	}
	want := []string{
		"   WARN  a.md  [alert] unclosed shortcode (x2)",
		"   WARN  a.md:14  [lint] heading level skipped",
		"   WARN  sarde.yaml: plugins.config.x  [plugin] unknown plugin",
		"  ERROR  z.md  [title] is required",
	}
	if strings.Join(lines, "\n") != strings.Join(want, "\n") {
		t.Errorf("lines:\n%s\nwant:\n%s", strings.Join(lines, "\n"), strings.Join(want, "\n"))
	}
}

func TestResolveWarningFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "content", "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "content", "docs", "a.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := newBuildReporter(&bytes.Buffer{}, dir, false, false, nil)
	r.contentDir = filepath.Join(dir, "content")

	abs := filepath.Join(dir, "content", "docs", "a.md")
	want := devlog.DisplayPath(abs)
	if got := r.resolveWarningFile("docs/a.md"); got != want {
		t.Errorf("content-relative: got %q, want %q", got, want)
	}
	if got := r.resolveWarningFile("content/docs/a.md"); got != want {
		t.Errorf("project-relative: got %q, want %q", got, want)
	}
	if got := r.resolveWarningFile(abs); got != want {
		t.Errorf("absolute: got %q, want %q", got, want)
	}
	if got := r.resolveWarningFile("sarde.yaml: plugins.config.x"); got != "sarde.yaml: plugins.config.x" {
		t.Errorf("config reference changed: %q", got)
	}
	if got := r.resolveWarningFile("missing/file.md"); got != "missing/file.md" {
		t.Errorf("unresolvable path: got %q", got)
	}
}

func TestStripLinePrefix(t *testing.T) {
	if got := stripLinePrefix("line 7: image missing alt text", 7); got != "image missing alt text" {
		t.Errorf("matching prefix not stripped: %q", got)
	}
	if got := stripLinePrefix("line 7: x", 8); got != "line 7: x" {
		t.Errorf("mismatched prefix stripped: %q", got)
	}
	if got := stripLinePrefix("line 7: x", 0); got != "line 7: x" {
		t.Errorf("prefix stripped without a Line: %q", got)
	}
}

func reportFixture() *engine.BuildResult {
	return &engine.BuildResult{
		PageCount: 2,
		Duration:  42 * time.Millisecond,
		OutputDir: "dist",
		Warnings:  []engine.ValidationWarning{{File: "a.md", Field: "lint", Message: "m"}},
		LogMessages: []engine.BuildLogEntry{
			{Source: "sitemap", Message: "Generated sitemap.xml"},
			{Source: "robots", Message: "Generated robots.txt"},
		},
		SlowestPages: []engine.PageTiming{{Path: "/tags/", Duration: 1500 * time.Microsecond}},
	}
}

func TestBuildReporterSummary_Sections(t *testing.T) {
	cfg := config.Defaults()

	var def bytes.Buffer
	newBuildReporter(&def, t.TempDir(), false, false, nil).summary(reportFixture(), cfg, 1)
	out := def.String()
	for _, want := range []string{"| Total", "1 warning:", "Built 2 pages in 42 ms, 2 warnings (1 in the log above)", "Output: dist"} {
		if !strings.Contains(out, want) {
			t.Errorf("default output missing %q:\n%s", want, out)
		}
	}
	for _, unwanted := range []string{"Plugins", "Slowest pages", "Theme:"} {
		if strings.Contains(out, unwanted) {
			t.Errorf("default output should not contain %q:\n%s", unwanted, out)
		}
	}
	if strings.Index(out, "1 warning:") > strings.Index(out, "Built 2 pages") {
		t.Errorf("warnings list must come before the final line:\n%s", out)
	}

	var verbose bytes.Buffer
	newBuildReporter(&verbose, t.TempDir(), true, false, nil).summary(reportFixture(), cfg, 0)
	out = verbose.String()
	for _, want := range []string{"Plugins", "Slowest pages", "1.5 ms  /tags/", "Theme:"} {
		if !strings.Contains(out, want) {
			t.Errorf("verbose output missing %q:\n%s", want, out)
		}
	}
	if strings.Index(out, "[robots]") > strings.Index(out, "[sitemap]") {
		t.Errorf("plugin lines must be sorted by source:\n%s", out)
	}

	var quiet bytes.Buffer
	newBuildReporter(&quiet, t.TempDir(), true, true, nil).summary(reportFixture(), cfg, 0)
	out = quiet.String()
	if !strings.Contains(out, "1 warning:") {
		t.Errorf("quiet output must keep warnings:\n%s", out)
	}
	for _, unwanted := range []string{"| Total", "Built", "Plugins"} {
		if strings.Contains(out, unwanted) {
			t.Errorf("quiet output should not contain %q:\n%s", unwanted, out)
		}
	}
}

func TestBuildReporterNextPhase(t *testing.T) {
	r := newBuildReporter(&bytes.Buffer{}, t.TempDir(), false, false, []string{"A", "B"})
	if got := r.nextPhase("A"); got != "B" {
		t.Errorf("nextPhase(A) = %q, want B", got)
	}
	if got := r.nextPhase("B"); got != "" {
		t.Errorf("nextPhase(B) = %q, want empty after the last phase", got)
	}
	if got := r.nextPhase("unknown"); got != "" {
		t.Errorf("nextPhase(unknown) = %q, want empty", got)
	}
}

// createCleanBuildSite is a site that builds without any warning: it has a
// description and does not ask for git dates outside a repository.
func createCleanBuildSite(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"sarde.yaml": "site:\n  title: \"Clean\"\n  description: \"A clean site\"\n  url: \"http://localhost:3000\"\n" +
			"build:\n  last_updated: mtime\n",
		"content/_index.md": "---\ntitle: Home\n---\n# Welcome\n",
	}
	for rel, body := range files {
		path := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// resetBuildFlags restores the shared rootCmd flags a build test changed.
func resetBuildFlags(t *testing.T) {
	t.Cleanup(func() {
		_ = rootCmd.PersistentFlags().Set("quiet", "false")
		_ = rootCmd.PersistentFlags().Set("verbose", "false")
		_ = buildCmd.Flags().Set("format", "pretty")
	})
}

func TestRunBuild_PrettyOutput(t *testing.T) {
	resetBuildFlags(t)
	dir := createCleanBuildSite(t)

	out, err := captureStdout(t, func() error {
		rootCmd.SetArgs([]string{"build", dir})
		return rootCmd.Execute()
	})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if !strings.HasPrefix(out, "Building site with sarde v") {
		t.Errorf("output must start with the header:\n%s", out)
	}
	if !strings.Contains(out, "\nBuilt 2 pages in ") || !strings.Contains(out, "  Output: ") {
		t.Errorf("missing final line or output path:\n%s", out)
	}
}

func TestRunBuild_QuietCleanBuildPrintsNothing(t *testing.T) {
	resetBuildFlags(t)
	dir := createCleanBuildSite(t)

	out, err := captureStdout(t, func() error {
		rootCmd.SetArgs([]string{"build", dir, "--quiet"})
		return rootCmd.Execute()
	})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if out != "" {
		t.Errorf("quiet clean build should print nothing on stdout, got:\n%s", out)
	}
}

func TestRunBuild_JSONKeepsContract(t *testing.T) {
	resetBuildFlags(t)
	dir := createCleanBuildSite(t)

	out, err := captureStdout(t, func() error {
		rootCmd.SetArgs([]string{"build", dir, "--format", "json"})
		return rootCmd.Execute()
	})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("stdout is not a single JSON object: %v\n%s", err, out)
	}
	for _, key := range []string{"PageCount", "Duration", "Warnings", "OutputDir", "PhaseTimings", "Links", "SlowestPages"} {
		if _, ok := m[key]; !ok {
			t.Errorf("JSON result missing %q", key)
		}
	}
}
