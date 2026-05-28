package cli

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/frostybee/sarde/internal/consts"
	"github.com/frostybee/sarde/internal/download"
	"github.com/frostybee/sarde/internal/theme"
	"github.com/spf13/cobra"
)

var themeAddCmd = &cobra.Command{
	Use:   "add <source>",
	Short: "Install a theme from a URL or local path",
	Long: `Install a theme into themes/<name>/ from:
  - A GitHub repository: github.com/user/theme-name
  - A direct zip/tar.gz URL
  - A local directory path`,
	Args: cobra.ExactArgs(1),
	RunE: runThemeAdd,
}

func init() {
	themeAddCmd.Flags().String("name", "", "Override the theme directory name")
}

func runThemeAdd(cmd *cobra.Command, args []string) error {
	src := args[0]

	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	kind := download.InferSourceKind(src)
	if kind == download.SourceUnknown {
		return fmt.Errorf("cannot determine source type for %q; provide a GitHub URL, zip/tar.gz URL, or local directory", src)
	}

	name, err := resolveThemeName(cmd, src, kind)
	if err != nil {
		return err
	}

	destDir := filepath.Join(projectDir, consts.DirThemes, name)
	if _, err := os.Stat(destDir); err == nil {
		return fmt.Errorf("theme %q already exists; remove it first with 'sarde theme remove %s'", name, name)
	}

	switch kind {
	case download.SourceLocalDir:
		err = installFromLocalDir(src, destDir)
	case download.SourceGitHub:
		err = installFromGitHub(src, destDir)
	case download.SourceZipURL:
		err = installFromZipURL(src, destDir)
	case download.SourceTarGzURL:
		err = installFromTarGzURL(src, destDir)
	}
	if err != nil {
		return err
	}

	thm, _ := theme.LoadFromDir(destDir)
	if thm == nil {
		os.RemoveAll(destDir)
		return fmt.Errorf("installed directory does not contain a valid theme.yaml; not a valid sarde theme")
	}

	if quiet, _ := cmd.Flags().GetBool("quiet"); !quiet {
		fmt.Printf("Installed theme %q to themes/%s/\n", name, name)
	}
	return nil
}

func resolveThemeName(cmd *cobra.Command, src string, kind download.SourceKind) (string, error) {
	if nameFlag, _ := cmd.Flags().GetString("name"); nameFlag != "" {
		return nameFlag, nil
	}

	switch kind {
	case download.SourceGitHub:
		ref, err := download.ParseGitHubURL(src)
		if err != nil {
			return "", err
		}
		if ref.Subpath != "" {
			return path.Base(ref.Subpath), nil
		}
		return ref.Repo, nil
	case download.SourceLocalDir:
		return filepath.Base(src), nil
	case download.SourceZipURL, download.SourceTarGzURL:
		base := path.Base(src)
		base = strings.TrimSuffix(base, ".tar.gz")
		base = strings.TrimSuffix(base, ".tgz")
		base = strings.TrimSuffix(base, ".zip")
		if base == "" || base == "." {
			return "", fmt.Errorf("cannot infer theme name from URL; use --name to specify")
		}
		return base, nil
	}

	return "", fmt.Errorf("cannot infer theme name; use --name to specify")
}

func installFromLocalDir(src, destDir string) error {
	if err := download.CopyDir(src, destDir); err != nil {
		return fmt.Errorf("copying theme from %s: %w", src, err)
	}
	return nil
}

func installFromGitHub(src, destDir string) error {
	ref, err := download.ParseGitHubURL(src)
	if err != nil {
		return err
	}

	tmpPath, err := download.DownloadFile(ref.ArchiveURL())
	if err != nil {
		return fmt.Errorf("downloading theme: %w", err)
	}
	defer os.Remove(tmpPath)

	if ref.Subpath == "" {
		return download.ExtractZip(tmpPath, destDir, 1)
	}

	scratchDir, err := os.MkdirTemp("", "sd-theme-extract-*")
	if err != nil {
		return fmt.Errorf("creating temp directory: %w", err)
	}
	defer os.RemoveAll(scratchDir)

	if err := download.ExtractZip(tmpPath, scratchDir, 1); err != nil {
		return err
	}

	subDir := filepath.Join(scratchDir, filepath.FromSlash(ref.Subpath))
	if _, err := os.Stat(subDir); err != nil {
		return fmt.Errorf("subpath %q not found in repository", ref.Subpath)
	}

	return download.CopyDir(subDir, destDir)
}

func installFromZipURL(src, destDir string) error {
	tmpPath, err := download.DownloadFile(src)
	if err != nil {
		return fmt.Errorf("downloading theme: %w", err)
	}
	defer os.Remove(tmpPath)

	return download.ExtractZip(tmpPath, destDir, 0)
}

func installFromTarGzURL(src, destDir string) error {
	tmpPath, err := download.DownloadFile(src)
	if err != nil {
		return fmt.Errorf("downloading theme: %w", err)
	}
	defer os.Remove(tmpPath)

	return download.ExtractTarGz(tmpPath, destDir, 0)
}
