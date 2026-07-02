package cli

import (
	"path/filepath"

	"github.com/getsarde/sarde/embedded"
	"github.com/getsarde/sarde/internal/build"
	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/engine"
	"github.com/spf13/cobra"
)

// applyCommonOverrides applies the --base-path and --content flag overrides
// shared by the build, check-links, and dev commands to the resolved config.
func applyCommonOverrides(cmd *cobra.Command, cfg *config.SiteConfig) {
	if basePath, _ := cmd.Flags().GetString("base-path"); basePath != "" {
		cfg.Build.BasePath = config.NormalizeBasePath(basePath)
	}
	if contentDir, _ := cmd.Flags().GetString("content"); contentDir != "" {
		if !filepath.IsAbs(contentDir) {
			contentDir, _ = filepath.Abs(contentDir)
		}
		cfg.Content.Dir = contentDir
	}
}

// newSiteBuilder constructs a SiteBuilder with the standard embedded theme,
// the shape shared by the build, check-links, and validate commands.
func newSiteBuilder(projectDir string, cfg *config.SiteConfig, themeCfg *engine.ThemeConfig) *build.SiteBuilder {
	return build.NewSiteBuilder(build.BuildOptions{
		ProjectDir:  projectDir,
		Config:      cfg,
		ThemeConfig: themeCfg,
		EmbeddedFS:  embedded.ThemeFS(),
	})
}
