package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alecthomas/chroma/v2/styles"
	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/content/markdown"
	"github.com/getsarde/sarde/internal/theme/syntax"
	"github.com/spf13/cobra"
)

var themeChromaStylesCmd = &cobra.Command{
	Use:   "chromastyles",
	Short: "Preview or export Chroma syntax highlighting CSS",
	Long: `Generate CSS for a Chroma syntax highlighting style.

Use --list to see all available style names.
Use --style to specify a style (defaults to the code block theme for the mode,
from themes in kazari.config.yaml).
Use --dark to wrap output in [data-theme="dark"] { } scoping for dark mode.
Use --output to write CSS to a file instead of stdout.`,
	RunE: runThemeChromaStyles,
}

func init() {
	themeChromaStylesCmd.Flags().String("style", "", "Chroma style name (default: the code block theme from kazari.config.yaml)")
	themeChromaStylesCmd.Flags().Bool("dark", false, "Wrap output in [data-theme=\"dark\"] { } scoping")
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
		light, dark := markdown.EffectiveCodeThemes(&cfg.Markdown.Codeblocks, filepath.Dir(configPath))
		if darkFlag && dark != "" {
			styleName = dark
		} else if light != "" {
			styleName = light
		} else {
			styleName = "github"
		}
	}

	css, err := syntax.GenerateStyleCSS(styleName)
	if err != nil {
		return err
	}

	if darkFlag {
		css = "[data-theme=\"dark\"] {\n" + css + "}\n"
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
