package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/getsarde/sarde/internal/engine"
	"github.com/spf13/cobra"
)

var catalogCmd = &cobra.Command{
	Use:   "catalog",
	Short: "Print the frontmatter field catalog",
	Long:  "Print the catalog of frontmatter fields Sarde recognizes, grouped by category, with the layout-to-category mapping. Used by Sarde Studio to offer field pickers.",
	Args:  cobra.NoArgs,
	RunE:  runCatalog,
}

func init() {
	catalogCmd.Flags().String("format", "pretty", "Output format: pretty, json")
	rootCmd.AddCommand(catalogCmd)
}

func runCatalog(cmd *cobra.Command, args []string) error {
	cat, err := engine.LoadFrontmatterCatalog()
	if err != nil {
		return err
	}

	format, _ := cmd.Flags().GetString("format")
	switch format {
	case "json":
		return json.NewEncoder(os.Stdout).Encode(cat)
	case "pretty":
		printCatalog(cat)
		return nil
	default:
		return fmt.Errorf("unknown format %q (expected pretty or json)", format)
	}
}

func printCatalog(cat *engine.FrontmatterCatalog) {
	for _, c := range cat.Categories {
		fmt.Printf("%s\n", c.Label)
		for _, f := range c.Fields {
			key := f.Key
			if c.Nested {
				key = c.ParentKey + "." + f.Key
			}
			note := ""
			if f.Unrendered {
				note = " (not rendered by current theme)"
			}
			fmt.Printf("  %-28s %-9s %s%s\n", key, f.Type, f.Description, note)
		}
		fmt.Println()
	}

	fmt.Println("Layouts")
	for _, layout := range []string{"default", "docs", "splash", "wide", "full", "centered"} {
		if cats, ok := cat.Layouts[layout]; ok {
			fmt.Printf("  %-28s %s\n", layout, strings.Join(cats, ", "))
		}
	}
}
