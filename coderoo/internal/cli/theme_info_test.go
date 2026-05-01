package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/coderoo-dev/coderoo/internal/consts"
)

func TestRunThemeInfo(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	themeDir := filepath.Join(dir, consts.DirThemes, "test-theme")
	os.MkdirAll(themeDir, 0o755)
	os.WriteFile(filepath.Join(themeDir, consts.FileThemeConfig), []byte(`
name: Test Theme
slug: test-theme
version: "1.0.0"
author: Tester
description: A test theme
`), 0o644)

	cmd := rootCmd
	cmd.SetArgs([]string{"theme", "info", "test-theme"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("theme info failed: %v", err)
	}
}

func TestRunThemeInfo_Default(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	cmd := rootCmd
	cmd.SetArgs([]string{"theme", "info", "default"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("theme info default failed: %v", err)
	}
}

func TestRunThemeInfo_NotFound(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	cmd := rootCmd
	cmd.SetArgs([]string{"theme", "info", "nonexistent"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error for nonexistent theme, got nil")
	}
}
