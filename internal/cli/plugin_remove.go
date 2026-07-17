package cli

import (
	"fmt"
	"os"

	"github.com/getsarde/sarde/internal/plugin/external"
	"github.com/spf13/cobra"
)

var pluginRemoveCmd = &cobra.Command{
	Use:   "remove <slug>",
	Short: "Remove an installed external plugin",
	Long: `Delete plugins/<slug>/ from the project. License files are kept:
they live outside the plugin directory and survive reinstalls.`,
	Args: cobra.ExactArgs(1),
	RunE: runPluginRemove,
}

func runPluginRemove(cmd *cobra.Command, args []string) error {
	slug := args[0]

	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	if err := external.Remove(projectDir, slug); err != nil {
		return err
	}

	if quiet, _ := cmd.Flags().GetBool("quiet"); !quiet {
		fmt.Printf("Removed plugin %q\n", slug)
	}
	return nil
}
