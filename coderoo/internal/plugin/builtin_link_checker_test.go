package plugin

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/coderoo-dev/coderoo/internal/config"
	"github.com/coderoo-dev/coderoo/internal/engine"
)

func TestLinkChecker_DetectsBrokenLinks(t *testing.T) {
	outDir := t.TempDir()

	// Create output structure.
	os.MkdirAll(filepath.Join(outDir, "about"), 0o755)
	os.WriteFile(filepath.Join(outDir, "about", "index.html"), []byte("<p>About page</p>"), 0o644)
	os.WriteFile(filepath.Join(outDir, "index.html"), []byte(`
		<a href="/about/">About</a>
		<a href="/missing/">Missing</a>
		<a href="https://external.com">External</a>
	`), 0o644)

	var warnings []engine.ValidationWarning
	ctx := &BuildDoneContext{
		Config:    config.Defaults(),
		OutputDir: outDir,
	}
	ctx.SetWarnings(&warnings)

	err := linkCheckerBuildDone(ctx, nil)
	if err != nil {
		t.Fatalf("linkCheckerBuildDone failed: %v", err)
	}

	// Should detect 1 broken link (/missing/), skip external.
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(warnings))
	}
	if warnings[0].Message != "broken link: /missing/" {
		t.Errorf("warning message = %q", warnings[0].Message)
	}
}

func TestLinkChecker_ValidLinks(t *testing.T) {
	outDir := t.TempDir()

	os.MkdirAll(filepath.Join(outDir, "docs"), 0o755)
	os.WriteFile(filepath.Join(outDir, "docs", "index.html"), []byte("<p>Docs</p>"), 0o644)
	os.WriteFile(filepath.Join(outDir, "index.html"), []byte(`<a href="/docs/">Docs</a>`), 0o644)

	var warnings []engine.ValidationWarning
	ctx := &BuildDoneContext{
		Config:    config.Defaults(),
		OutputDir: outDir,
	}
	ctx.SetWarnings(&warnings)

	linkCheckerBuildDone(ctx, nil)

	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings for valid links, got %d", len(warnings))
	}
}

func TestLinkChecker_IgnorePatterns(t *testing.T) {
	outDir := t.TempDir()

	os.WriteFile(filepath.Join(outDir, "index.html"), []byte(`
		<a href="/api/v1/">API</a>
		<a href="/missing/">Missing</a>
	`), 0o644)

	var warnings []engine.ValidationWarning
	ctx := &BuildDoneContext{
		Config:    config.Defaults(),
		OutputDir: outDir,
	}
	ctx.SetWarnings(&warnings)

	cfg := map[string]any{"ignore": []any{"/api/*"}}
	linkCheckerBuildDone(ctx, cfg)

	// /api/v1/ is ignored, /missing/ is reported.
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(warnings))
	}
}

func TestLinkChecker_SkipsExternalAndSpecial(t *testing.T) {
	outDir := t.TempDir()

	os.WriteFile(filepath.Join(outDir, "index.html"), []byte(`
		<a href="https://google.com">Google</a>
		<a href="mailto:test@example.com">Email</a>
		<a href="#section">Anchor</a>
		<a href="data:text/plain,hi">Data</a>
		<a href="tel:+1234567890">Phone</a>
	`), 0o644)

	var warnings []engine.ValidationWarning
	ctx := &BuildDoneContext{
		Config:    config.Defaults(),
		OutputDir: outDir,
	}
	ctx.SetWarnings(&warnings)

	linkCheckerBuildDone(ctx, nil)

	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings for external/special links, got %d", len(warnings))
	}
}
