package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/frostybee/sarde/embedded"
	"github.com/frostybee/sarde/internal/build"
	"github.com/frostybee/sarde/internal/config"
	"github.com/frostybee/sarde/internal/engine"
	"github.com/frostybee/sarde/internal/theme"
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
	buildCmd.Flags().String("base-path", "", "Override URL base path (e.g. /docs/)")
	buildCmd.Flags().String("content", "", "Override content directory path")
	buildCmd.Flags().Bool("strict-i18n", false, "Warn on missing translation keys per language")
	rootCmd.AddCommand(buildCmd)
}

func runBuild(cmd *cobra.Command, args []string) error {
	projectDir := projectDirFromArgs(args)

	// Resolve config.
	cfg, themeCfg, err := resolveAll(cmd, projectDir)
	if err != nil {
		return err
	}

	// Override output dir from CLI flag.
	if output, _ := cmd.Flags().GetString("output"); output != "" {
		cfg.Build.Output = output
	}

	// Override base path from CLI flag.
	if basePath, _ := cmd.Flags().GetString("base-path"); basePath != "" {
		cfg.Build.BasePath = config.NormalizeBasePath(basePath)
	}

	// Override content directory from CLI flag.
	if contentDir, _ := cmd.Flags().GetString("content"); contentDir != "" {
		if !filepath.IsAbs(contentDir) {
			contentDir, _ = filepath.Abs(contentDir)
		}
		cfg.Content.Dir = contentDir
	}

	// Override strict i18n from CLI flag.
	if strictI18n, _ := cmd.Flags().GetBool("strict-i18n"); strictI18n {
		cfg.I18n.Strict = true
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
	verbose, _ := cmd.Flags().GetBool("verbose")
	if !quiet {
		printBuildSummary(result, verbose, cfg)
		if len(result.Warnings) > 0 {
			fmt.Printf("\n  %d warning(s):\n", len(result.Warnings))
			for _, w := range result.Warnings {
				fmt.Printf("    %s: %s\n", w.File, w.Message)
			}
		}
	}

	return nil
}

func printBuildSummary(result *engine.BuildResult, verbose bool, cfg *config.SiteConfig) {
	fmt.Printf("\nStart building sites ...\n")
	fmt.Printf("sarde v%s %s/%s\n", Version, runtime.GOOS, runtime.GOARCH)

	if verbose {
		fmt.Println()
		for _, pt := range result.PhaseTimings {
			fmt.Printf("[build] %s... done (%s)\n", pt.Phase, pt.Duration.Round(time.Millisecond))
		}
	}

	fmt.Println()
	printStatsTable(result)

	if len(result.LogMessages) > 0 {
		fmt.Println()
		for _, msg := range result.LogMessages {
			fmt.Printf("[%s] %s\n", msg.Source, msg.Message)
		}
	}

	fmt.Printf("\nBuilt in %d ms\n", result.Duration.Milliseconds())
	fmt.Printf("  Output: %s\n", result.OutputDir)

	if verbose {
		fmt.Printf("  Theme: %s\n", cfg.Theme.Name)
		fmt.Printf("  Base path: %q\n", cfg.Build.BasePath)
		fmt.Printf("  Content dir: %s\n", cfg.Content.Dir)
	}
}

func printStatsTable(result *engine.BuildResult) {
	type row struct {
		label string
		value int
	}
	rows := []row{
		{"Pages", result.PageCount},
		{"Paginator pages", result.PaginatorPages},
		{"Collections", result.Collections},
		{"Bundle assets", result.BundleAssets},
		{"Static files", result.StaticFiles},
		{"Processed images", result.ProcessedImages},
		{"Aliases", result.AliasCount},
		{"Sitemaps", result.SitemapCount},
	}

	fmt.Printf("%19s | Total\n", "")
	fmt.Printf("-------------------+-------\n")
	for _, r := range rows {
		fmt.Printf("  %-17s|%5d\n", r.label, r.value)
	}
}

// resolveAll resolves site config and theme config from the project directory.
func resolveAll(cmd *cobra.Command, projectDir string) (*config.SiteConfig, *engine.ThemeConfig, error) {
	configPath, _ := cmd.Flags().GetString("config")
	if !filepath.IsAbs(configPath) {
		configPath = filepath.Join(projectDir, configPath)
	}

	cfg, err := config.Resolve(config.ResolveOptions{
		ConfigPath:   configPath,
		CLIFlags:     CollectCLIFlags(cmd),
		EnvPrefix:    "SARDE",
		Strict:       true,
		KnownPlugins: build.KnownPluginNames(),
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

	// Fold theme shortcut fields into overrides before token resolution.
	foldThemeShortcuts(cfg)

	// Validate token names in overrides.
	known := theme.KnownTokens()
	if err := theme.ValidateOverrides("theme.overrides", cfg.Theme.Overrides, known); err != nil {
		return nil, nil, err
	}
	if err := theme.ValidateOverrides("theme.dark_overrides", cfg.Theme.DarkOverrides, known); err != nil {
		return nil, nil, err
	}

	// Resolve tokens.
	lightTokens := theme.ResolveTokens(theme.DefaultTokens(), thm, cfg.Theme.Preset, cfg.Theme.Overrides)
	lightTokens = theme.DeriveTokens(lightTokens)
	darkTokens := theme.ResolveDarkTokens(theme.DefaultDarkTokens(), thm, cfg.Theme.Preset, darkOverrides(cfg))
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

// projectDirFromArgs returns the project directory from the first positional arg,
// or falls back to the current working directory. Used by all CLI commands so the
// desktop app (Tauri) can pass the project path explicitly.
func projectDirFromArgs(args []string) string {
	if len(args) > 0 && args[0] != "" {
		if filepath.IsAbs(args[0]) {
			return args[0]
		}
		if abs, err := filepath.Abs(args[0]); err == nil {
			return abs
		}
	}
	dir, _ := os.Getwd()
	return dir
}

// foldThemeShortcuts maps convenience theme config fields (accent_color, etc.)
// into the Overrides map so they flow through the token resolution cascade.
// Explicit Overrides entries take precedence over shortcuts.
func foldThemeShortcuts(cfg *config.SiteConfig) {
	accentVal := cfg.Theme.AccentColor
	if accentVal == "" {
		accentVal = cfg.Theme.PrimaryColor
	}
	shortcuts := map[string]string{
		"accent":    accentVal,
		"font-sans": cfg.Theme.FontFamily,
		"font-mono": cfg.Theme.FontMono,
	}
	if cfg.Theme.Overrides == nil {
		cfg.Theme.Overrides = make(map[string]string)
	}
	for token, val := range shortcuts {
		if val != "" {
			if _, exists := cfg.Theme.Overrides[token]; !exists {
				cfg.Theme.Overrides[token] = val
			}
		}
	}
	if cfg.Theme.CodeLight != "" && cfg.Markdown.Codeblocks.LightTheme == "" {
		cfg.Markdown.Codeblocks.LightTheme = cfg.Theme.CodeLight
	}
	if cfg.Theme.CodeDark != "" && cfg.Markdown.Codeblocks.DarkTheme == "" {
		cfg.Markdown.Codeblocks.DarkTheme = cfg.Theme.CodeDark
	}
}

func darkOverrides(cfg *config.SiteConfig) map[string]string {
	if len(cfg.Theme.DarkOverrides) > 0 {
		return cfg.Theme.DarkOverrides
	}
	return cfg.Theme.Overrides
}
