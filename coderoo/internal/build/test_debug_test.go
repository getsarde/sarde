package build

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/coderoo-dev/coderoo/embedded"
	"github.com/coderoo-dev/coderoo/internal/config"
)

func TestBuild_Debug(t *testing.T) {
	projDir := createFixtureSite(t)
	cfg := config.Defaults()
	themeCfg := buildThemeConfig()

	builder := NewSiteBuilder(BuildOptions{
		ProjectDir:  projDir,
		Config:      cfg,
		ThemeConfig: themeCfg,
		EmbeddedFS:  embedded.ThemeFS(),
	})

	result, err := builder.Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	t.Logf("Build succeeded. PageCount: %d", result.PageCount)
	
	distDir := filepath.Join(projDir, "dist")
	
	// List all files in dist
	filepath.Walk(distDir, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			rel, _ := filepath.Rel(distDir, path)
			t.Logf("File: %s", rel)
		}
		return nil
	})
}
