package cli

import (
	"fmt"
	"os"

	"github.com/getsarde/sarde/internal/plugin"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate site config and content without building",
	Long:  "Validate site configuration and content files. Runs discovery, parsing, schema validation, and optional content linting without rendering or writing output.",
	RunE:  runValidate,
	// A validation failure is not a usage mistake — don't drown the actual
	// error in the flags listing.
	SilenceUsage: true,
}

func init() {
	validateCmd.Flags().Bool("lint", true, "Run content lint rules after validation")
	validateCmd.Flags().Bool("strict", false, "Exit with code 1 if any warnings exist")
	validateCmd.Flags().String("format", "pretty", "Error output format: pretty, json")
	rootCmd.AddCommand(validateCmd)
}

func runValidate(cmd *cobra.Command, args []string) error {
	format, _ := cmd.Flags().GetString("format")
	if format != "pretty" && format != "json" {
		return fmt.Errorf("unknown format %q (expected pretty or json)", format)
	}
	err := validateProject(cmd, args)
	if err != nil && format == "json" {
		return emitJSONError("validate_failed", err)
	}
	return err
}

func validateProject(cmd *cobra.Command, args []string) error {
	projectDir := projectDirFromArgs(args)

	cfg, themeCfg, err := resolveAll(cmd, projectDir)
	if err != nil {
		return err
	}

	builder := newSiteBuilder(projectDir, cfg, themeCfg)

	result, err := builder.Validate()
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	lint, _ := cmd.Flags().GetBool("lint")
	if lint && len(result.Pages) > 0 {
		lintWarnings := plugin.LintPages(result.Pages, cfg.ContentLint)
		result.Warnings = append(result.Warnings, lintWarnings...)
	}

	quiet, _ := cmd.Flags().GetBool("quiet")
	if !quiet {
		fmt.Printf("%d pages across %d collections validated in %s\n",
			result.PageCount, result.Collections, result.Duration.Round(1e6))
		if len(result.Warnings) > 0 {
			fmt.Printf("  %d warning(s):\n", len(result.Warnings))
			for _, w := range result.Warnings {
				fmt.Printf("    %s: [%s] %s\n", w.File, w.Field, w.Message)
			}
		}
	}

	strict, _ := cmd.Flags().GetBool("strict")
	if strict && len(result.Warnings) > 0 {
		os.Exit(1)
	}

	return nil
}
