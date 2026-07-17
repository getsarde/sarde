package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/consts"
	"github.com/getsarde/sarde/internal/license"
	"github.com/getsarde/sarde/internal/plugin/external"
	"github.com/spf13/cobra"
)

var pluginCmd = &cobra.Command{
	Use:   "plugin",
	Short: "Manage external plugins",
	Long:  "Install, list, inspect, and remove external plugins in the plugins/ directory.",
}

var pluginListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed external plugins",
	RunE:  runPluginList,
}

func init() {
	pluginCmd.AddCommand(pluginListCmd)
	pluginCmd.AddCommand(pluginInstallCmd)
	pluginCmd.AddCommand(pluginRemoveCmd)
	pluginCmd.AddCommand(pluginInfoCmd)
	rootCmd.AddCommand(pluginCmd)
}

func runPluginList(cmd *cobra.Command, args []string) error {
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	dirs := external.DiscoverDirs(projectDir)
	if len(dirs) == 0 {
		fmt.Println("No external plugins installed. Install one with 'sarde plugin install <source>'.")
		return nil
	}

	disabled := disabledPluginSet(projectDir)

	fmt.Printf("%-20s %-10s %-20s %s\n", "SLUG", "VERSION", "LICENSE", "STATUS")
	for _, dir := range dirs {
		slug := filepath.Base(dir)
		m, err := external.LoadManifest(dir)
		if err != nil {
			fmt.Printf("%-20s %-10s %-20s %s\n", slug, "-", "-", "invalid manifest")
			continue
		}
		status := "enabled"
		if disabled[slug] {
			status = "disabled"
		}
		fmt.Printf("%-20s %-10s %-20s %s\n", slug, m.Version, pluginLicenseStatus(projectDir, m), status)
	}
	return nil
}

// disabledPluginSet resolves plugins.disabled from site config, tolerating a
// missing or invalid sarde.yaml (zero-config projects have none).
func disabledPluginSet(projectDir string) map[string]bool {
	set := make(map[string]bool)
	cfg, err := config.Resolve(config.ResolveOptions{
		ConfigPath: filepath.Join(projectDir, consts.FileSiteConfig),
		EnvPrefix:  "SARDE",
	})
	if err != nil {
		return set
	}
	for _, name := range cfg.Plugins.Disabled {
		set[name] = true
	}
	return set
}

// pluginLicenseStatus summarizes a plugin's license state for display.
func pluginLicenseStatus(projectDir string, m *external.Manifest) string {
	if !m.Premium {
		return "free"
	}
	if err := license.VerifyFor(projectDir, m.Slug, m.Version); err != nil {
		if _, found := license.Locate(projectDir, m.Slug); !found {
			return "premium (no license)"
		}
		return "premium (invalid)"
	}
	return "premium (licensed)"
}
