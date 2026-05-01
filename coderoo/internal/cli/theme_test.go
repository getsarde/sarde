package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/coderoo-dev/coderoo/internal/consts"
)

func TestRunThemeEject(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	cmd := rootCmd
	cmd.SetArgs([]string{"theme", "eject"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("theme eject failed: %v", err)
	}

	themeDir := filepath.Join(dir, consts.DirThemes, "default")

	mustExist := []string{
		filepath.Join(consts.FileThemeConfig),
		filepath.Join(consts.DirLayouts, consts.DirDefault, consts.TemplateBaseOf),
		filepath.Join(consts.DirLayouts, consts.DirDefault, "home.html"),
		filepath.Join(consts.DirLayouts, consts.DirDefault, "single.html"),
		filepath.Join(consts.DirLayouts, consts.DirDocs, consts.TemplateBaseOf),
		filepath.Join(consts.DirLayouts, consts.DirComponents, "Header.html"),
		filepath.Join(consts.DirLayouts, consts.DirPartials, "seo.html"),
		filepath.Join("css", "tokens.css"),
		filepath.Join("assets", "js", "tabs.js"),
	}

	for _, rel := range mustExist {
		full := filepath.Join(themeDir, rel)
		if _, err := os.Stat(full); err != nil {
			t.Errorf("expected ejected file %s, got: %v", rel, err)
		}
	}
}

func TestRunThemeEject_AlreadyExists(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	os.MkdirAll(filepath.Join(dir, consts.DirThemes, "default"), 0o755)

	cmd := rootCmd
	cmd.SetArgs([]string{"theme", "eject"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error for existing themes/default/, got nil")
	}
}

func TestRunThemeEject_Force(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	// Eject once.
	cmd := rootCmd
	cmd.SetArgs([]string{"theme", "eject"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("first eject failed: %v", err)
	}

	// Eject again with --force.
	cmd.SetArgs([]string{"theme", "eject", "--force"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("force eject failed: %v", err)
	}

	themeYAML := filepath.Join(dir, consts.DirThemes, "default", consts.FileThemeConfig)
	if _, err := os.Stat(themeYAML); err != nil {
		t.Errorf("theme.yaml not present after force eject: %v", err)
	}
}

func TestRemapThemePath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"_default/baseof.html", filepath.Join(consts.DirLayouts, "_default/baseof.html")},
		{"_docs/single.html", filepath.Join(consts.DirLayouts, "_docs/single.html")},
		{"_blog/list.html", filepath.Join(consts.DirLayouts, "_blog/list.html")},
		{"components/Header.html", filepath.Join(consts.DirLayouts, "components/Header.html")},
		{"partials/seo.html", filepath.Join(consts.DirLayouts, "partials/seo.html")},
		{"css/tokens.css", "css/tokens.css"},
		{"assets/js/tabs.js", "assets/js/tabs.js"},
		{"theme.yaml", "theme.yaml"},
	}

	for _, tt := range tests {
		got := remapThemePath(tt.input)
		if got != tt.want {
			t.Errorf("remapThemePath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
