package build

import (
	"reflect"
	"testing"

	"github.com/getsarde/sarde/embedded"
	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/engine"
)

// Check applies per-call overrides (strict policies, report format, enabled)
// to the shared config; they must be fully restored afterwards so a reused
// builder/config is unaffected.
func TestCheck_RestoresLinkValidationConfig(t *testing.T) {
	projDir := createFixtureSite(t)
	cfg := config.Defaults()
	cfg.LinkValidation.OnBroken = "warn"
	cfg.LinkValidation.Exclude = []string{"x/*"}

	before := cfg.LinkValidation

	builder := NewSiteBuilder(BuildOptions{
		ProjectDir:  projDir,
		Config:      cfg,
		ThemeConfig: buildThemeConfig(),
		EmbeddedFS:  embedded.ThemeFS(),
	})

	// Run twice: the restore must be idempotent across reuse.
	for i := 1; i <= 2; i++ {
		if _, err := builder.Check(CheckOptions{Strict: true, ReportFormat: "json"}); err != nil {
			t.Fatalf("Check run %d: %v", i, err)
		}
		if !reflect.DeepEqual(cfg.LinkValidation, before) {
			t.Errorf("run %d: LinkValidation mutated by Check:\n got: %+v\nwant: %+v", i, cfg.LinkValidation, before)
		}
		if builder.checkOnly {
			t.Errorf("run %d: checkOnly not reset", i)
		}
	}
}

// The CLI wraps Build's error as "build failed: %w", so the error itself must
// not carry that prefix too ("build failed: build failed: ...").
func TestBuild_BrokenLinkErrorHasNoBuildFailedPrefix(t *testing.T) {
	projDir := createFixtureSite(t)
	writeFixture(t, projDir, "content/docs/broken.md", "---\ntitle: Broken\n---\nSee [the missing page](./missing.md).\n")
	cfg := config.Defaults()
	cfg.LinkValidation.OnBroken = "error"

	builder := NewSiteBuilder(BuildOptions{
		ProjectDir:  projDir,
		Config:      cfg,
		ThemeConfig: buildThemeConfig(),
		EmbeddedFS:  embedded.ThemeFS(),
	})
	_, err := builder.Build()
	if err == nil {
		t.Fatal("Build succeeded with a broken internal link and on_broken: error")
	}
	if got, want := err.Error(), "link validation errors found"; got != want {
		t.Errorf("Build error = %q, want %q", got, want)
	}
}

// An extension-less site path that no page answers is a broken target once
// the whole site is known, while links to pages that exist only in the
// output, a tag page, a numbered list page or a redirect from the redirects:
// config, and to a public/ folder with an index.html pass without a warning.
func TestBuild_UnresolvedSitePathIsBrokenTarget(t *testing.T) {
	newSite := func(t *testing.T, body string) string {
		dir := t.TempDir()
		writeFixture(t, dir, "content/_index.md", "---\ntitle: Home\n---\nWelcome.\n")
		writeFixture(t, dir, "content/blog/_index.md", "---\ntitle: Blog\n---\n")
		writeFixture(t, dir, "content/blog/one.md", "---\ntitle: One\ndate: 2025-01-02\ntags: [botany]\n---\nBody.\n")
		writeFixture(t, dir, "content/blog/two.md", "---\ntitle: Two\ndate: 2025-01-01\ntags: [botany]\n---\nBody.\n")
		writeFixture(t, dir, "public/demo/index.html", "<p>demo</p>")
		writeFixture(t, dir, "content/docs/guide.md", "---\ntitle: Guide\n---\n"+body)
		return dir
	}
	build := func(dir string) (*engine.BuildResult, error) {
		cfg := config.Defaults()
		cfg.Build.Minify = config.BoolPtr(false)
		cfg.Collections = map[string]*config.CollectionSiteConfig{"blog": {Paginate: config.IntPtr(1)}}
		cfg.Redirects = map[string]string{"/syllabus": "/docs/guide/"}
		return NewSiteBuilder(BuildOptions{ProjectDir: dir, Config: cfg, ThemeConfig: buildThemeConfig(), EmbeddedFS: embedded.ThemeFS()}).Build()
	}

	result, err := build(newSite(t, "See [botany](/tags/botany), [page two](/blog/page/2), [the syllabus](/syllabus) and [the demo](/demo/).\n"))
	if err != nil {
		t.Fatalf("Build failed on links to output-only pages: %v", err)
	}
	for _, w := range result.Warnings {
		if w.Field == "link" {
			t.Errorf("unexpected link warning: %v", w)
		}
	}

	if _, err := build(newSite(t, "See [the lost page](/nope/missing).\n")); err == nil || err.Error() != "link validation errors found" {
		t.Errorf("Build with a link to nothing: err = %v, want link validation errors", err)
	}
}
