package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/frostybee/sarde/embedded"
	"github.com/frostybee/sarde/internal/build"
	"github.com/frostybee/sarde/internal/config"
	"github.com/spf13/cobra"
)

var checkLinksCmd = &cobra.Command{
	Use:     "check-links",
	Aliases: []string{"check"},
	Short:   "Check links without building",
	Long:    "Run link validation (internal and optionally external) without rendering templates or writing output.\n\nAliased as 'check' for backward compatibility.",
	RunE:    runCheck,
}

func init() {
	checkLinksCmd.Flags().Bool("strict", false, "Treat all link issues as errors (exit 1)")
	checkLinksCmd.Flags().Bool("external", false, "Also probe external URLs")
	checkLinksCmd.Flags().String("report", "", "Report format: pretty, json, github-actions")
	checkLinksCmd.Flags().String("base-path", "", "Override URL base path (e.g. /docs/)")
	checkLinksCmd.Flags().String("content", "", "Override content directory path")
	rootCmd.AddCommand(checkLinksCmd)
}

func runCheck(cmd *cobra.Command, args []string) error {
	projectDir := projectDirFromArgs(args)

	cfg, themeCfg, err := resolveAll(cmd, projectDir)
	if err != nil {
		return err
	}

	if basePath, _ := cmd.Flags().GetString("base-path"); basePath != "" {
		cfg.Build.BasePath = config.NormalizeBasePath(basePath)
	}
	if contentDir, _ := cmd.Flags().GetString("content"); contentDir != "" {
		if !filepath.IsAbs(contentDir) {
			contentDir, _ = filepath.Abs(contentDir)
		}
		cfg.Content.Dir = contentDir
	}

	builder := build.NewSiteBuilder(build.BuildOptions{
		ProjectDir:  projectDir,
		Config:      cfg,
		ThemeConfig: themeCfg,
		EmbeddedFS:  embedded.ThemeFS(),
	})

	strict, _ := cmd.Flags().GetBool("strict")
	external, _ := cmd.Flags().GetBool("external")
	report, _ := cmd.Flags().GetString("report")

	result, err := builder.Check(build.CheckOptions{
		External:     external,
		Strict:       strict,
		ReportFormat: report,
	})
	if err != nil {
		return fmt.Errorf("check failed: %w", err)
	}

	// Report output is already printed to stderr by GenerateReport.
	// Print a summary if no findings were reported.
	quiet, _ := cmd.Flags().GetBool("quiet")
	if !quiet && result.Output == "" {
		fmt.Fprintf(os.Stderr, "checked %d pages, %d links across %d lanes — no issues found\n",
			result.PageCount, result.LinkCount, result.Lanes)
	}

	if result.HasErrors {
		os.Exit(1)
	}

	return nil
}
