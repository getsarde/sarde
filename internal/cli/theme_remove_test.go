package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/getsarde/sarde/internal/consts"
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

func TestValidThemeName(t *testing.T) {
	rejected := []string{
		"", ".", "..", "../..", "../foo", "a/b", `a\b`, `..\..`,
		"/abs/path", `C:\Windows`, "themes/../..",
	}
	for _, name := range rejected {
		if validThemeName(name) {
			t.Errorf("validThemeName(%q) = true, want false", name)
		}
	}
	accepted := []string{"minimal", "my-theme", "theme_2", "Theme.Name"}
	for _, name := range accepted {
		if !validThemeName(name) {
			t.Errorf("validThemeName(%q) = false, want true", name)
		}
	}
}

// Regression test: "theme remove .." used to resolve themes/.. to the project
// root and os.RemoveAll it. It must be rejected before any filesystem access.
func TestRunThemeRemove_TraversalRejected(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	// A sentinel file in the project root; it must survive.
	sentinel := filepath.Join(dir, "sarde.yaml")
	os.WriteFile(sentinel, []byte("title: keep me\n"), 0o644)

	for _, name := range []string{"..", "../..", "../outside", `..\..`} {
		cmd := rootCmd
		cmd.SetArgs([]string{"theme", "remove", name})
		if err := cmd.Execute(); err == nil {
			t.Errorf("expected error for theme name %q, got nil", name)
		}
	}

	if _, err := os.Stat(sentinel); err != nil {
		t.Fatalf("project root content was deleted by traversal name: %v", err)
	}
}

func TestRunThemeRemove_ActiveTheme_RequiresForce(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	// Create a sarde.yaml referencing this theme as active.
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
