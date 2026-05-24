package cli

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/frostybee/sarde/embedded"
	"github.com/frostybee/sarde/internal/config"
	"github.com/frostybee/sarde/internal/consts"
	"github.com/frostybee/sarde/internal/theme"
	"github.com/spf13/cobra"
)

var themeCmd = &cobra.Command{
	Use:   "theme",
	Short: "Manage themes",
}

var themeEjectCmd = &cobra.Command{
	Use:   "eject",
	Short: "Eject the embedded default theme to themes/default/",
	Long:  "Copy the full embedded theme (templates, CSS, JS, components, partials, theme.yaml) to themes/default/ for customization.",
	RunE:  runThemeEject,
}

var themeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available themes",
	Long:  "List all available themes: the built-in default theme and any themes installed in themes/.",
	RunE:  runThemeList,
}

func init() {
	themeEjectCmd.Flags().Bool("force", false, "Overwrite existing themes/default/ directory")
	themeCmd.AddCommand(themeEjectCmd)
	themeCmd.AddCommand(themeListCmd)
	themeCmd.AddCommand(themeAddCmd)
	themeCmd.AddCommand(themeRemoveCmd)
	themeCmd.AddCommand(themeInfoCmd)
	rootCmd.AddCommand(themeCmd)
}

func runThemeEject(cmd *cobra.Command, args []string) error {
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	targetDir := filepath.Join(projectDir, consts.DirThemes, "default")
	if _, err := os.Stat(targetDir); err == nil {
		force, _ := cmd.Flags().GetBool("force")
		if !force {
			return fmt.Errorf("themes/default/ already exists; use --force to overwrite")
		}
		if err := os.RemoveAll(targetDir); err != nil {
			return fmt.Errorf("removing existing themes/default/: %w", err)
		}
	}

	themeFS := embedded.ThemeFS()
	fileCount := 0

	err = fs.WalkDir(themeFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == "." {
			return nil
		}

		destPath := filepath.Join(targetDir, remapThemePath(path))

		if d.IsDir() {
			return os.MkdirAll(destPath, 0o755)
		}

		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			return fmt.Errorf("creating directory for %s: %w", path, err)
		}

		data, err := fs.ReadFile(themeFS, path)
		if err != nil {
			return fmt.Errorf("reading embedded %s: %w", path, err)
		}

		if err := os.WriteFile(destPath, data, 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", destPath, err)
		}

		fileCount++
		return nil
	})
	if err != nil {
		return fmt.Errorf("ejecting theme: %w", err)
	}

	if quiet, _ := cmd.Flags().GetBool("quiet"); !quiet {
		fmt.Printf("Ejected default theme to themes/default/ (%d files)\n", fileCount)
	}
	return nil
}

func runThemeList(cmd *cobra.Command, args []string) error {
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	configPath, _ := cmd.Flags().GetString("config")
	cfg, err := config.Resolve(config.ResolveOptions{ConfigPath: configPath})
	if err != nil {
		cfg = config.Defaults()
	}
	activeTheme := cfg.Theme.Name

	embeddedTheme, _ := theme.LoadFromFS(embedded.ThemeFS(), ".")
	embeddedDesc := ""
	if embeddedTheme != nil && embeddedTheme.Description != "" {
		embeddedDesc = embeddedTheme.Description
	}

	marker := " "
	if activeTheme == "" || activeTheme == "default" {
		marker = "*"
	}
	fmt.Printf("  %s default (embedded)", marker)
	if embeddedDesc != "" {
		fmt.Printf("  %s", embeddedDesc)
	}
	fmt.Println()

	themesDir := filepath.Join(projectDir, consts.DirThemes)
	entries, err := os.ReadDir(themesDir)
	if err != nil {
		return nil
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		thm, _ := theme.LoadFromDir(filepath.Join(themesDir, name))
		if thm == nil {
			continue
		}
		marker := " "
		if name == activeTheme {
			marker = "*"
		}
		fmt.Printf("  %s %s", marker, name)
		if thm.Description != "" {
			fmt.Printf("  %s", thm.Description)
		}
		fmt.Println()
	}

	return nil
}

func remapThemePath(embeddedPath string) string {
	switch {
	case strings.HasPrefix(embeddedPath, consts.DirDefault+"/"),
		strings.HasPrefix(embeddedPath, consts.DirDocs+"/"),
		strings.HasPrefix(embeddedPath, consts.DirBlog+"/"),
		strings.HasPrefix(embeddedPath, consts.DirComponents+"/"),
		strings.HasPrefix(embeddedPath, consts.DirPartials+"/"):
		return filepath.Join(consts.DirLayouts, embeddedPath)
	default:
		return embeddedPath
	}
}
