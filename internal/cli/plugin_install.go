package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/getsarde/sarde/internal/build"
	"github.com/getsarde/sarde/internal/license"
	"github.com/getsarde/sarde/internal/plugin/external"
	"github.com/spf13/cobra"
)

var pluginInstallCmd = &cobra.Command{
	Use:   "install <source>",
	Short: "Install an external plugin from a zip, URL, or local path",
	Long: `Install a plugin into plugins/<slug>/ from:
  - A local zip file: sarde plugin install slideviewer.zip
  - A GitHub repository: github.com/user/plugin-name
  - A direct zip/tar.gz URL
  - A local directory path

The destination directory name always matches the slug declared in the
plugin's plugin.yaml manifest.`,
	Args: cobra.ExactArgs(1),
	RunE: runPluginInstall,
}

func runPluginInstall(cmd *cobra.Command, args []string) error {
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	m, err := external.Install(projectDir, args[0], build.KnownPluginNames(""))
	if err != nil {
		return err
	}

	if quiet, _ := cmd.Flags().GetBool("quiet"); !quiet {
		fmt.Printf("Installed plugin %q %s to plugins/%s/\n", m.Name, m.Version, m.Slug)
		if m.Premium {
			fmt.Println()
			fmt.Printf("%q is a premium plugin and needs a license to activate.\n", m.Name)
			if m.PurchaseURL != "" {
				fmt.Printf("Purchase a license at %s\n", m.PurchaseURL)
			}
			fmt.Println("Then install it with: sarde license install <license-file>")
			fmt.Printf("License locations checked:\n  %s\n",
				strings.Join(license.CandidatePaths(projectDir, m.Slug), "\n  "))
		}
	}
	return nil
}
