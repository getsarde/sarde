package plugin

import (
	"strings"
	"testing"

	"github.com/frostybee/sarde/internal/config"
	"github.com/frostybee/sarde/internal/engine"
)

func TestRobots_GeneratesFile(t *testing.T) {
	outDir := t.TempDir()
	var warnings []engine.ValidationWarning

	ctx := &BuildDoneContext{
		Config:    config.Defaults(),
		OutputDir: outDir,
		Site:      &engine.SiteContext{BaseURL: "https://example.com"},
	}
	ctx.SetWarnings(&warnings)

	err := robotsBuildDone(ctx, nil)
	if err != nil {
		t.Fatalf("robotsBuildDone failed: %v", err)
	}

	data, err := readTestFile(outDir, "robots.txt")
	if err != nil {
		t.Fatalf("reading robots.txt: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "User-agent: *") {
		t.Error("expected User-agent directive")
	}
	if !strings.Contains(content, "Allow: /") {
		t.Error("expected Allow directive")
	}
	if !strings.Contains(content, "Sitemap: https://example.com/sitemap.xml") {
		t.Error("expected Sitemap directive")
	}
}

func TestRobots_NoSitemap(t *testing.T) {
	outDir := t.TempDir()
	var warnings []engine.ValidationWarning

	ctx := &BuildDoneContext{
		Config:    config.Defaults(),
		OutputDir: outDir,
		Site:      &engine.SiteContext{BaseURL: "https://example.com"},
	}
	ctx.SetWarnings(&warnings)

	cfg := map[string]any{"sitemap": false}
	err := robotsBuildDone(ctx, cfg)
	if err != nil {
		t.Fatalf("robotsBuildDone failed: %v", err)
	}

	data, _ := readTestFile(outDir, "robots.txt")
	if strings.Contains(string(data), "Sitemap:") {
		t.Error("should not include Sitemap when disabled")
	}
}
