package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/alecthomas/chroma/v2/styles"
	"github.com/frostybee/sarde/internal/config"
	"github.com/frostybee/sarde/internal/theme/syntax"
	"github.com/spf13/cobra"
)

var themeChromaStylesCmd = &cobra.Command{
	Use:   "chromastyles",
	Short: "Preview or export Chroma syntax highlighting CSS",
	Long: `Generate CSS for a Chroma syntax highlighting style.

Use --list to see all available style names.
Use --style to specify a style (defaults to the configured light theme).
Use --dark to wrap output in .dark { } scoping for dark mode.
Use --output to write CSS to a file instead of stdout.`,
	RunE: runThemeChromaStyles,
}

func init() {
	themeChromaStylesCmd.Flags().String("style", "", "Chroma style name (default: configured light theme)")
	themeChromaStylesCmd.Flags().Bool("dark", false, "Wrap output in .dark { } scoping")
	themeChromaStylesCmd.Flags().Bool("list", false, "List all available Chroma style names")
	themeChromaStylesCmd.Flags().StringP("output", "o", "", "Write CSS to file instead of stdout")
	themeCmd.AddCommand(themeChromaStylesCmd)
}

func runThemeChromaStyles(cmd *cobra.Command, args []string) error {
	listFlag, _ := cmd.Flags().GetBool("list")
	if listFlag {
		fmt.Println(strings.Join(styles.Names(), "\n"))
		return nil
	}

	styleName, _ := cmd.Flags().GetString("style")
	darkFlag, _ := cmd.Flags().GetBool("dark")

	if styleName == "" {
		configPath, _ := cmd.Flags().GetString("config")
		cfg, err := config.Resolve(config.ResolveOptions{ConfigPath: configPath, Strict: true})
		if err != nil {
			cfg = config.Defaults()
		}
		if darkFlag && cfg.Markdown.Codeblocks.DarkTheme != "" {
			styleName = cfg.Markdown.Codeblocks.DarkTheme
		} else if cfg.Markdown.Codeblocks.LightTheme != "" {
			styleName = cfg.Markdown.Codeblocks.LightTheme
		} else {
			styleName = "github"
		}
	}

	css, err := syntax.GenerateStyleCSS(styleName)
	if err != nil {
		return err
	}

	if darkFlag {
		css = ".dark {\n" + css + "}\n"
	}

	outputPath, _ := cmd.Flags().GetString("output")
	if outputPath != "" {
		if err := os.WriteFile(outputPath, []byte(css), 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", outputPath, err)
		}
		if quiet, _ := cmd.Flags().GetBool("quiet"); !quiet {
			fmt.Printf("Wrote %s CSS to %s\n", styleName, outputPath)
		}
		return nil
	}

	fmt.Print(css)
	return nil
}
