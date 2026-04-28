package cli

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/coderoo-dev/coderoo/embedded"
	"github.com/coderoo-dev/coderoo/internal/consts"
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

func init() {
	themeCmd.AddCommand(themeEjectCmd)
	rootCmd.AddCommand(themeCmd)
}

func runThemeEject(cmd *cobra.Command, args []string) error {
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	targetDir := filepath.Join(projectDir, consts.DirThemes, "default")
	if _, err := os.Stat(targetDir); err == nil {
		return fmt.Errorf("themes/default/ already exists; remove it first to re-eject")
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
