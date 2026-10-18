package cli

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/getsarde/sarde/internal/engine"
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
	validateCmd.Flags().String("format", "pretty", "Output format: pretty, json")
	rootCmd.AddCommand(validateCmd)
}

func runValidate(cmd *cobra.Command, args []string) error {
	format, _ := cmd.Flags().GetString("format")
	if format != "pretty" && format != "json" {
		return fmt.Errorf("unknown format %q (expected pretty or json)", format)
	}
	err := validateProject(cmd, args, format)
	// A --strict failure has already printed its result (with "ok": false
	// in JSON mode); only a validation error gets the envelope.
	if err != nil && format == "json" && !errors.Is(err, errStrictWarnings) {
		return emitJSONError("validate_failed", err)
	}
	return err
}

// errStrictWarnings fails a --strict run that found warnings.
var errStrictWarnings = errors.New("warnings found with --strict")

// validateJSON is the --format json result. Its keys are camelCase like the
// other machine outputs Studio reads; the build's JSON (engine.BuildResult)
// is a different, PascalCase shape and is not affected.
type validateJSON struct {
	OK          bool                  `json:"ok"`
	Pages       int                   `json:"pages"`
	Collections int                   `json:"collections"`
	DurationMs  int64                 `json:"durationMs"`
	Warnings    []validateJSONWarning `json:"warnings"`
}

type validateJSONWarning struct {
	File    string `json:"file"`
	Field   string `json:"field"`
	Message string `json:"message"`
	Level   string `json:"level"`
	Line    int    `json:"line,omitempty"`
	Col     int    `json:"col,omitempty"`
}

func validateProject(cmd *cobra.Command, args []string, format string) error {
	projectDir := projectDirFromArgs(args)
	if err := requireSite(cmd, projectDir); err != nil {
		return err
	}

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
	engine.SortWarnings(result.Warnings)

	strict, _ := cmd.Flags().GetBool("strict")
	failed := strict && len(result.Warnings) > 0
	out := cmd.OutOrStdout()

	if format == "json" {
		res := validateJSON{
			OK:          !failed,
			Pages:       result.PageCount,
			Collections: result.Collections,
			DurationMs:  result.Duration.Milliseconds(),
			Warnings:    make([]validateJSONWarning, 0, len(result.Warnings)),
		}
		for _, w := range result.Warnings {
			// Warnings carry "warning", "warn" or nothing depending on the
			// check; JSON consumers get one spelling.
			level := w.Level
			if level == "" || level == "warn" {
				level = "warning"
			}
			res.Warnings = append(res.Warnings, validateJSONWarning{
				File: w.File, Field: w.Field, Message: w.Message, Level: level, Line: w.Line, Col: w.Col,
			})
		}
		if err := json.NewEncoder(out).Encode(res); err != nil {
			return err
		}
	} else if quiet, _ := cmd.Flags().GetBool("quiet"); !quiet {
		fmt.Fprintf(out, "%d pages across %d collections validated in %s\n",
			result.PageCount, result.Collections, result.Duration.Round(1e6))
		if len(result.Warnings) > 0 {
			fmt.Fprintf(out, "  %d warning(s):\n", len(result.Warnings))
			for _, w := range result.Warnings {
				fmt.Fprintf(out, "    %s: [%s] %s\n", w.File, w.Field, w.Message)
			}
		}
	}

	if failed {
		return errStrictWarnings
	}
	return nil
}
