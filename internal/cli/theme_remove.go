package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/consts"
	"github.com/spf13/cobra"
)

var themeRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove an installed theme",
	Args:  cobra.ExactArgs(1),
	RunE:  runThemeRemove,
}

func init() {
	themeRemoveCmd.Flags().Bool("force", false, "Remove even if the theme is currently active")
}

func runThemeRemove(cmd *cobra.Command, args []string) error {
	name := args[0]

	if name == "default" {
		return fmt.Errorf("the 'default' theme is embedded and cannot be removed")
	}

	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	themeDir := filepath.Join(projectDir, consts.DirThemes, name)
	if _, err := os.Stat(themeDir); err != nil {
		return fmt.Errorf("theme %q is not installed", name)
	}

	configPath, _ := cmd.Flags().GetString("config")
	cfg, err := config.Resolve(config.ResolveOptions{ConfigPath: configPath, Strict: true})
	if err != nil {
		cfg = config.Defaults()
	}

	if cfg.Theme.Name == name {
		force, _ := cmd.Flags().GetBool("force")
		if !force {
			return fmt.Errorf("theme %q is the active theme; use --force to remove", name)
		}
		fmt.Fprintf(os.Stderr, "warning: removing active theme %q\n", name)
	}

	if err := os.RemoveAll(themeDir); err != nil {
		return fmt.Errorf("removing theme %q: %w", name, err)
	}

	if quiet, _ := cmd.Flags().GetBool("quiet"); !quiet {
		fmt.Printf("Removed theme %q\n", name)
	}
	return nil
}
