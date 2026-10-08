package build

import (
	"reflect"
	"testing"

	"github.com/getsarde/sarde/embedded"
	"github.com/getsarde/sarde/internal/config"
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
