package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/getsarde/sarde/internal/plugin/catalog"
	"github.com/spf13/cobra"
)

var pluginsCmd = &cobra.Command{
	Use:   "plugins [path]",
	Short: "Print the plugin catalog",
	Long: "Print every plugin Sarde accepts in plugins.enabled — built-in server and client plugins plus external plugins found under the project's plugins/ directory — with descriptions, default state, and configurable fields. Used by Sarde Studio to render plugin settings.\n\n" +
		"Metadata only: plugins.disabled and premium license checks are not applied; builds enforce both.",
	Args:         cobra.MaximumNArgs(1),
	SilenceUsage: true,
	RunE:         runPlugins,
}

func init() {
	pluginsCmd.Flags().String("format", "pretty", "Output format: pretty, json")
	rootCmd.AddCommand(pluginsCmd)
}

func runPlugins(cmd *cobra.Command, args []string) error {
	cat := catalog.Build(projectDirFromArgs(args))

	format, _ := cmd.Flags().GetString("format")
	switch format {
	case "json":
		return json.NewEncoder(os.Stdout).Encode(cat)
	case "pretty":
		printPluginCatalog(cat)
		return nil
	default:
		return fmt.Errorf("unknown format %q (expected pretty or json)", format)
	}
}

func printPluginCatalog(cat catalog.Catalog) {
	for _, p := range cat.Plugins {
		state := "off"
		if p.DefaultEnabled {
			state = "on"
		}
		fmt.Printf("%-24s %-9s %-10s %-4s %s\n", p.ID, p.Kind, p.Group, state, p.Description)
	}
}
