package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/coderoo-dev/coderoo/internal/consts"
)

func TestRunThemeRemove(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	themeDir := filepath.Join(dir, consts.DirThemes, "my-theme")
	os.MkdirAll(themeDir, 0o755)
	os.WriteFile(filepath.Join(themeDir, consts.FileThemeConfig), []byte("name: my-theme"), 0o644)

	cmd := rootCmd
	cmd.SetArgs([]string{"theme", "remove", "my-theme"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("theme remove failed: %v", err)
	}

	if _, err := os.Stat(themeDir); err == nil {
		t.Error("theme directory should have been removed")
	}
}

func TestRunThemeRemove_NotInstalled(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	cmd := rootCmd
	cmd.SetArgs([]string{"theme", "remove", "nonexistent"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error for nonexistent theme, got nil")
	}
}

func TestRunThemeRemove_Default(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	cmd := rootCmd
	cmd.SetArgs([]string{"theme", "remove", "default"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error when removing default theme, got nil")
	}
}

func TestRunThemeRemove_ActiveTheme_RequiresForce(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	// Create a site.yaml referencing this theme as active.
	os.WriteFile(filepath.Join(dir, consts.FileSiteConfig), []byte("theme:\n  name: active-theme\n"), 0o644)

	themeDir := filepath.Join(dir, consts.DirThemes, "active-theme")
	os.MkdirAll(themeDir, 0o755)
	os.WriteFile(filepath.Join(themeDir, consts.FileThemeConfig), []byte("name: active-theme"), 0o644)

	// Without --force should fail.
	cmd := rootCmd
	cmd.SetArgs([]string{"theme", "remove", "active-theme"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error when removing active theme without --force")
	}

	// With --force should succeed.
	cmd.SetArgs([]string{"theme", "remove", "active-theme", "--force"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("theme remove --force failed: %v", err)
	}

	if _, err := os.Stat(themeDir); err == nil {
		t.Error("theme directory should have been removed with --force")
	}
}
