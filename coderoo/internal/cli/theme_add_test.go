package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/coderoo-dev/coderoo/internal/consts"
)

func TestRunThemeAdd_LocalDir(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	// Create a source theme directory.
	srcDir := filepath.Join(dir, "src-theme")
	os.MkdirAll(srcDir, 0o755)
	os.WriteFile(filepath.Join(srcDir, consts.FileThemeConfig), []byte("name: src-theme\nslug: src-theme"), 0o644)
	os.MkdirAll(filepath.Join(srcDir, "css"), 0o755)
	os.WriteFile(filepath.Join(srcDir, "css", "tokens.css"), []byte("body {}"), 0o644)

	cmd := rootCmd
	cmd.SetArgs([]string{"theme", "add", srcDir})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("theme add failed: %v", err)
	}

	installed := filepath.Join(dir, consts.DirThemes, "src-theme")
	if _, err := os.Stat(filepath.Join(installed, consts.FileThemeConfig)); err != nil {
		t.Error("theme.yaml not found in installed theme")
	}
	if _, err := os.Stat(filepath.Join(installed, "css", "tokens.css")); err != nil {
		t.Error("css/tokens.css not found in installed theme")
	}
}

func TestRunThemeAdd_LocalDir_WithName(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	srcDir := filepath.Join(dir, "src-theme")
	os.MkdirAll(srcDir, 0o755)
	os.WriteFile(filepath.Join(srcDir, consts.FileThemeConfig), []byte("name: original"), 0o644)

	cmd := rootCmd
	cmd.SetArgs([]string{"theme", "add", srcDir, "--name", "custom-name"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("theme add --name failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, consts.DirThemes, "custom-name", consts.FileThemeConfig)); err != nil {
		t.Error("theme not installed under custom name")
	}
}

func TestRunThemeAdd_AlreadyExists(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	srcDir := filepath.Join(dir, "src-theme")
	os.MkdirAll(srcDir, 0o755)
	os.WriteFile(filepath.Join(srcDir, consts.FileThemeConfig), []byte("name: src-theme"), 0o644)

	// Pre-create the destination.
	os.MkdirAll(filepath.Join(dir, consts.DirThemes, "my-existing"), 0o755)

	cmd := rootCmd
	cmd.SetArgs([]string{"theme", "add", srcDir, "--name", "my-existing"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error for already-existing theme, got nil")
	}
}

func TestRunThemeAdd_NoThemeYAML(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	// Source directory without theme.yaml.
	srcDir := filepath.Join(dir, "not-a-theme")
	os.MkdirAll(srcDir, 0o755)
	os.WriteFile(filepath.Join(srcDir, "readme.md"), []byte("# Not a theme"), 0o644)

	cmd := rootCmd
	cmd.SetArgs([]string{"theme", "add", srcDir})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error for directory without theme.yaml, got nil")
	}

	// Verify cleanup — destination should not exist.
	if _, err := os.Stat(filepath.Join(dir, consts.DirThemes, "not-a-theme")); err == nil {
		t.Error("destination should have been cleaned up after validation failure")
	}
}

func TestRunThemeAdd_UnknownSource(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	cmd := rootCmd
	cmd.SetArgs([]string{"theme", "add", "not-a-url-or-dir"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error for unknown source, got nil")
	}
}
