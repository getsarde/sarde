package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/getsarde/sarde/internal/collection"
	"github.com/spf13/cobra"
)

var effectiveConfigCmd = &cobra.Command{
	Use:   "effective-config [project-dir]",
	Short: "Show each collection's merged effective config with per-field provenance",
	Long: "Prints, for every content collection, the fully resolved config (after zero-config " +
		"inference and any sarde.yaml overrides), labeling each field as \"inferred\" (from the " +
		"collection directory's name) or \"sarde_yaml\" (explicitly set). Powers Sarde Studio's " +
		"Effective Config inspector; also useful directly for understanding a site's zero-config " +
		"defaults.",
	RunE: runEffectiveConfig,
}

func init() {
	effectiveConfigCmd.Flags().String("format", "pretty", "Output format: pretty, json")
	rootCmd.AddCommand(effectiveConfigCmd)
}

func runEffectiveConfig(cmd *cobra.Command, args []string) error {
	format, _ := cmd.Flags().GetString("format")
	if format != "json" && format != "pretty" {
		return fmt.Errorf("unknown format %q (expected pretty or json)", format)
	}

	projectDir := projectDirFromArgs(args)

	collections, err := collection.BuildEffectiveConfig(projectDir)
	if err != nil {
		if format == "json" {
			return printJSONError(err)
		}
		return err
	}

	if format == "json" {
		return json.NewEncoder(os.Stdout).Encode(map[string]any{"collections": collections})
	}
	printEffectiveConfig(collections)
	return nil
}

func printEffectiveConfig(collections []collection.EffectiveCollection) {
	printField := func(label string, fv collection.FieldValue) {
		val := "auto"
		if fv.Value != nil {
			val = fmt.Sprintf("%v", fv.Value)
		}
		fmt.Printf("  %-30s %-24s [%s]\n", label+":", val, fv.Source)
	}
	for _, c := range collections {
		fmt.Printf("%s  (inferred as %s collection from its name)\n", c.Name, c.InferredType)
		printField("sort_by", c.SortBy)
		printField("sort_order", c.SortOrder)
		printField("layout", c.Layout)
		printField("permalink", c.Permalink)
		printField("paginate", c.Paginate)
		printField("feed", c.Feed)
		printField("tabs", c.Tabs)
		if c.Sidebar != nil {
			printField("sidebar.collapsible", c.Sidebar.Collapsible)
			printField("sidebar.collapsed_by_default", c.Sidebar.CollapsedByDefault)
			printField("sidebar.max_depth", c.Sidebar.MaxDepth)
			printField("sidebar.search", c.Sidebar.Search)
		}
		if c.TOC != nil {
			printField("toc.enabled", c.TOC.Enabled)
			printField("toc.scroll_highlight", c.TOC.ScrollHighlight)
			printField("toc.depth", c.TOC.Depth)
		}
		if c.PrevNext != nil {
			printField("prev_next.enabled", c.PrevNext.Enabled)
			printField("prev_next.labels", c.PrevNext.Labels)
		}
		fmt.Println()
	}
}
