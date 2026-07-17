package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/getsarde/sarde/internal/consts"
	"github.com/getsarde/sarde/internal/license"
	"github.com/getsarde/sarde/internal/plugin/external"
	"github.com/spf13/cobra"
)

var pluginInfoCmd = &cobra.Command{
	Use:   "info <slug>",
	Short: "Show details about an installed external plugin",
	Args:  cobra.ExactArgs(1),
	RunE:  runPluginInfo,
}

func runPluginInfo(cmd *cobra.Command, args []string) error {
	slug := args[0]

	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	dir := filepath.Join(projectDir, consts.DirPlugins, slug)
	m, err := external.LoadManifest(dir)
	if err != nil {
		return fmt.Errorf("plugin %q is not installed or has no readable manifest: %w", slug, err)
	}

	printField("Name", m.Name)
	printField("Slug", m.Slug)
	printField("Version", m.Version)
	printField("Description", m.Description)
	printField("Author", m.Author)
	printField("Homepage", m.Homepage)
	printField("Premium", fmt.Sprintf("%v", m.Premium))
	if m.Premium {
		status := "licensed"
		if err := license.VerifyFor(projectDir, slug, m.Version); err != nil {
			status = err.Error()
		}
		printField("License", status)
		printField("Purchase URL", m.PurchaseURL)
	}
	if m.Inject.HasAssets() {
		when := m.Inject.When
		switch when {
		case "layout":
			when += " " + m.Inject.Layout
		case "collection":
			when += " " + m.Inject.Collection
		}
		printField("Injects when", when)
		printField("Styles", strings.Join(m.Inject.Styles, ", "))
		printField("Scripts", strings.Join(append(m.Inject.Scripts, m.Inject.ModuleScripts...), ", "))
	}
	printField("Output", m.EffectivePrefix())
	return nil
}
