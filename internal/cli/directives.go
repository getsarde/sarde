package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/getsarde/sarde/internal/engine"
	"github.com/spf13/cobra"
)

var directivesCmd = &cobra.Command{
	Use:   "directives",
	Short: "Print the block directive catalog",
	Long:  "Print the catalog of ::: block directives Sarde recognizes, grouped by category, with syntax templates and key fields. Used by Sarde Studio to offer a directive picker.",
	Args:  cobra.NoArgs,
	RunE:  runDirectives,
}

func init() {
	directivesCmd.Flags().String("format", "pretty", "Output format: pretty, json")
	rootCmd.AddCommand(directivesCmd)
}

func runDirectives(cmd *cobra.Command, args []string) error {
	cat, err := engine.LoadDirectiveCatalog()
	if err != nil {
		return err
	}

	format, _ := cmd.Flags().GetString("format")
	switch format {
	case "json":
		return json.NewEncoder(os.Stdout).Encode(cat)
	case "pretty":
		printDirectiveCatalog(cat)
		return nil
	default:
		return fmt.Errorf("unknown format %q (expected pretty or json)", format)
	}
}

func printDirectiveCatalog(cat *engine.DirectiveCatalog) {
	for _, c := range cat.Categories {
		fmt.Printf("%s\n", c.Label)
		for _, d := range c.Directives {
			fmt.Printf("  %-18s %s\n", d.Name, d.Description)
		}
		fmt.Println()
	}
}
