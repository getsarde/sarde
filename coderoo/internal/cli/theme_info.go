package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/coderoo-dev/coderoo/embedded"
	"github.com/coderoo-dev/coderoo/internal/consts"
	"github.com/coderoo-dev/coderoo/internal/theme"
	"github.com/spf13/cobra"
)

var themeInfoCmd = &cobra.Command{
	Use:   "info <name>",
	Short: "Show details about an installed theme",
	Args:  cobra.ExactArgs(1),
	RunE:  runThemeInfo,
}

func runThemeInfo(cmd *cobra.Command, args []string) error {
	name := args[0]

	var thm *theme.Theme
	var err error

	if name == "default" {
		thm, err = theme.LoadFromFS(embedded.ThemeFS(), ".")
	} else {
		projectDir, e := os.Getwd()
		if e != nil {
			return fmt.Errorf("getting working directory: %w", e)
		}
		themeDir := filepath.Join(projectDir, consts.DirThemes, name)
		if _, statErr := os.Stat(themeDir); statErr != nil {
			return fmt.Errorf("theme %q not found", name)
		}
		thm, err = theme.LoadFromDir(themeDir)
	}

	if err != nil {
		return fmt.Errorf("loading theme %q: %w", name, err)
	}
	if thm == nil {
		return fmt.Errorf("theme %q not found", name)
	}

	printField("Name", thm.Name)
	printField("Slug", thm.Slug)
	printField("Version", thm.Version)
	printField("Author", thm.Author)
	printField("License", thm.License)
	printField("Description", thm.Description)

	if len(thm.Tokens) > 0 {
		fmt.Printf("%-14s%d\n", "Tokens:", len(thm.Tokens))
	}
	if len(thm.DarkTokens) > 0 {
		fmt.Printf("%-14s%d\n", "Dark tokens:", len(thm.DarkTokens))
	}
	if len(thm.Presets) > 0 {
		names := make([]string, 0, len(thm.Presets))
		for k := range thm.Presets {
			names = append(names, k)
		}
		sort.Strings(names)
		fmt.Printf("%-14s%s\n", "Presets:", strings.Join(names, ", "))
	}

	return nil
}

func printField(label, value string) {
	if value != "" {
		fmt.Printf("%-14s%s\n", label+":", value)
	}
}
