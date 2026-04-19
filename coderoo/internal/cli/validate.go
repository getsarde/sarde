package cli

import (
	"fmt"

	"github.com/coderoo-dev/coderoo/embedded"
	"github.com/coderoo-dev/coderoo/internal/build"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate site config and content without building",
	Long:  "Validate site configuration and content files. Runs discovery, parsing, and schema validation without rendering or writing output.",
	RunE:  runValidate,
}

func init() {
	rootCmd.AddCommand(validateCmd)
}

func runValidate(cmd *cobra.Command, args []string) error {
	projectDir := projectDirFromArgs(args)

	cfg, themeCfg, err := resolveAll(cmd, projectDir)
	if err != nil {
		return err
	}

	builder := build.NewSiteBuilder(build.BuildOptions{
		ProjectDir:  projectDir,
		Config:      cfg,
		ThemeConfig: themeCfg,
		EmbeddedFS:  embedded.ThemeFS(),
	})

	result, err := builder.Validate()
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	quiet, _ := cmd.Flags().GetBool("quiet")
	if !quiet {
		fmt.Printf("%d pages across %d collections validated in %s\n",
			result.PageCount, result.Collections, result.Duration.Round(1e6))
		if len(result.Warnings) > 0 {
			fmt.Printf("  %d warning(s):\n", len(result.Warnings))
			for _, w := range result.Warnings {
				fmt.Printf("    %s: %s\n", w.File, w.Message)
			}
		}
	}

	return nil
}
