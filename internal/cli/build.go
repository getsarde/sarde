package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/coderoo-dev/coderoo/embedded"
	"github.com/coderoo-dev/coderoo/internal/build"
	"github.com/coderoo-dev/coderoo/internal/config"
	"github.com/coderoo-dev/coderoo/internal/engine"
	"github.com/coderoo-dev/coderoo/internal/theme"
	"github.com/spf13/cobra"
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build the static site",
	Long:  "Build the static site from content/ to the output directory.",
	RunE:  runBuild,
}

func init() {
	buildCmd.Flags().StringP("output", "o", "", "Override output directory (default: dist)")
	rootCmd.AddCommand(buildCmd)
}

func runBuild(cmd *cobra.Command, args []string) error {
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	// Resolve config.
	cfg, themeCfg, err := resolveAll(cmd, projectDir)
	if err != nil {
		return err
	}

	// Override output dir from CLI flag.
	if output, _ := cmd.Flags().GetString("output"); output != "" {
		cfg.Build.Output = output
	}

	// Build.
	builder := build.NewSiteBuilder(build.BuildOptions{
		ProjectDir:  projectDir,
		Config:      cfg,
		ThemeConfig: themeCfg,
		EmbeddedFS:  embedded.ThemeFS(),
	})

	result, err := builder.Build()
	if err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	// Print summary.
	quiet, _ := cmd.Flags().GetBool("quiet")
	if !quiet {
		fmt.Printf("Built %d pages in %s\n", result.PageCount, result.Duration.Round(1e6))
		if len(result.Warnings) > 0 {
			fmt.Printf("  %d warning(s):\n", len(result.Warnings))
			for _, w := range result.Warnings {
				fmt.Printf("    %s: %s\n", w.File, w.Message)
			}
		}
		fmt.Printf("  Output: %s\n", result.OutputDir)
	}

	return nil
}

// resolveAll resolves site config and theme config from the project directory.
func resolveAll(cmd *cobra.Command, projectDir string) (*config.SiteConfig, *engine.ThemeConfig, error) {
	configPath, _ := cmd.Flags().GetString("config")
	if !filepath.IsAbs(configPath) {
		configPath = filepath.Join(projectDir, configPath)
	}

	cfg, err := config.Resolve(config.ResolveOptions{
		ConfigPath: configPath,
		CLIFlags:   CollectCLIFlags(cmd),
		EnvPrefix:  "CODEROO",
	})
	if err != nil {
		return nil, nil, fmt.Errorf("resolving config: %w", err)
	}

	// Load theme.
	var thm *theme.Theme
	if cfg.Theme.Name != "" && cfg.Theme.Name != "default" {
		thm, _ = theme.LoadFromDir(filepath.Join(projectDir, "themes", cfg.Theme.Name))
	}
	if thm == nil {
		thm, _ = theme.LoadFromFS(embedded.ThemeFS(), ".")
	}

	// Resolve tokens.
	lightTokens := theme.ResolveTokens(theme.DefaultTokens(), thm, cfg.Theme.Preset, cfg.Theme.Overrides)
	lightTokens = theme.DeriveTokens(lightTokens)
	darkTokens := theme.ResolveDarkTokens(theme.DefaultDarkTokens(), thm, cfg.Theme.Preset, nil)
	styleTag := theme.GenerateStyleTag(lightTokens, darkTokens)

	name := "Default"
	slug := "default"
	if thm != nil {
		if thm.Name != "" {
			name = thm.Name
		}
		if thm.Slug != "" {
			slug = thm.Slug
		}
	}

	themeCfg := &engine.ThemeConfig{
		Name:        name,
		Slug:        slug,
		Tokens:      lightTokens,
		DarkTokens:  darkTokens,
		DarkEnabled: config.BoolVal(cfg.Theme.Dark, true),
		StyleTag:    styleTag,
	}

	return cfg, themeCfg, nil
}
