package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/getsarde/sarde/embedded"
	"github.com/getsarde/sarde/internal/build"
	"github.com/getsarde/sarde/internal/buildlock"
	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/devlog"
	"github.com/getsarde/sarde/internal/engine"
	"github.com/getsarde/sarde/internal/outputpath"
	"github.com/getsarde/sarde/internal/theme"
	"github.com/spf13/cobra"
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build the static site",
	Long:  "Build the static site from content/ to the output directory.",
	RunE:  runBuild,
	// A build failure (bad config, broken theme) is not a usage mistake —
	// don't drown the actual error in the flags listing. Cobra still prints
	// "Error: <msg>" to stderr (main.go itself never prints).
	SilenceUsage: true,
}

func init() {
	buildCmd.Flags().StringP("output", "o", "", "Override output directory (default: dist)")
	buildCmd.Flags().String("base-path", "", "Override URL base path (e.g. /docs/)")
	buildCmd.Flags().String("content", "", "Override content directory path")
	buildCmd.Flags().Bool("strict-i18n", false, "Warn on missing translation keys per language")
	buildCmd.Flags().String("format", "pretty", "Output format: pretty, json")
	rootCmd.AddCommand(buildCmd)
}

func runBuild(cmd *cobra.Command, args []string) error {
	format, _ := cmd.Flags().GetString("format")
	if format != "pretty" && format != "json" {
		return fmt.Errorf("unknown format %q (expected pretty or json)", format)
	}
	// One boundary for machine-readable failure: every error path inside the
	// build (config validation, theme load, lock, build proper) surfaces here
	// and becomes an {"error": ...} envelope on stdout in json mode.
	err := runBuildWithFormat(cmd, args, format)
	if err != nil && format == "json" {
		return emitJSONError("build_failed", err)
	}
	return err
}

func runBuildWithFormat(cmd *cobra.Command, args []string, format string) error {
	// Start the passive update lookup now so it overlaps with the build and
	// its result can be shown after the summary. JSON mode stays silent and
	// makes no network calls.
	var updateCheck *pendingUpdateCheck
	if format != "json" {
		updateCheck = beginUpdateCheck(cmd)
	}

	projectDir := projectDirFromArgs(args)
	if err := requireSite(cmd, projectDir); err != nil {
		return err
	}

	// Human output: the header comes first so config warnings and build log
	// lines land below it. warnBase brackets the devlog warnings this run
	// prints, which the final line counts.
	quiet, _ := cmd.Flags().GetBool("quiet")
	verbose, _ := cmd.Flags().GetBool("verbose")
	warnBase := devlog.WarnCount()
	var report *buildReporter
	if format == "json" {
		devlog.DisableProgress()
	} else {
		prevQuiet := devlog.Quiet()
		devlog.SetQuiet(quiet)
		defer devlog.SetQuiet(prevQuiet)
		report = newBuildReporter(os.Stdout, projectDir, verbose, quiet, build.FullBuildPhases())
		report.header()
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

	// Override base path and content directory from CLI flags.
	applyCommonOverrides(cmd, cfg)

	// Override strict i18n from CLI flag.
	if strictI18n, _ := cmd.Flags().GetBool("strict-i18n"); strictI18n {
		cfg.I18n.Strict = true
	}

	// Guard the output dir against a concurrent sarde process (dev or build)
	// writing into the same dist/. Resolved from the same cfg the builder
	// resolves later, so the locked dir always equals the written dir.
	outputDir, err := outputpath.ResolveOutputDir(projectDir, cfg.Build.Output)
	if err != nil {
		return err
	}
	lock, err := buildlock.Acquire(outputDir, "build")
	if err != nil {
		return err
	}
	defer lock.Release()

	// Build.
	builder := newSiteBuilder(projectDir, cfg, themeCfg)
	if report != nil {
		report.contentDir = builder.ContentDir()
		builder.SetPhaseObserver(report.onPhase)
		devlog.SetProgress("build", "%s...", build.FullBuildPhases()[0])
	}

	result, err := builder.Build()
	devlog.ClearProgress()
	if err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	// JSON output: emit the raw BuildResult and skip the human-readable
	// summary entirely. Consumed by the benchmark harness (cmd/sarde-bench)
	// and any tooling that wants structured build stats.
	if format == "json" {
		return json.NewEncoder(os.Stdout).Encode(result)
	}

	report.summary(result, cfg, int(devlog.WarnCount()-warnBase))

	// Passive update notice; waits briefly for the lookup started at the top
	// of the run. Nil-safe when the check was skipped or in JSON mode.
	updateCheck.finishAndNotify()

	return nil
}

// resolveAll resolves site config and theme config from the project directory.
func resolveAll(cmd *cobra.Command, projectDir string) (*config.SiteConfig, *engine.ThemeConfig, error) {
	cfg, err := config.Resolve(config.ResolveOptions{
		ConfigPath:   configPathFor(cmd, projectDir),
		CLIFlags:     CollectCLIFlags(cmd),
		EnvPrefix:    "SARDE",
		Strict:       true,
		KnownPlugins: build.KnownPluginNames(projectDir),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("resolving config: %w", err)
	}

	// Load theme. A configured custom theme that fails to load is an error:
	// silently falling back to the embedded default would build the site with
	// none of the user's theme customizations and no indication why.
	var thm *theme.Theme
	if cfg.Theme.Name != "" && cfg.Theme.Name != "default" {
		themeDir := filepath.Join(projectDir, "themes", cfg.Theme.Name)
		loaded, loadErr := theme.LoadFromDir(themeDir)
		if loadErr != nil {
			return nil, nil, fmt.Errorf("loading theme %q from %s: %w", cfg.Theme.Name, themeDir, loadErr)
		}
		thm = loaded
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

// foldThemeShortcuts folds the theme shortcut fields into Overrides (see
// config.FoldThemeShortcuts) and applies code_light / code_dark as the
// codeblock themes when those are not set explicitly.
func foldThemeShortcuts(cfg *config.SiteConfig) {
	config.FoldThemeShortcuts(&cfg.Theme)
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
