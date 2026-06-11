package build

import (
	"reflect"
	"testing"

	"github.com/frostybee/sarde/embedded"
	"github.com/frostybee/sarde/internal/config"
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
