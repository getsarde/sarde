package cli

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/getsarde/sarde/embedded"
	"github.com/getsarde/sarde/internal/consts"
)

// resetEjectFlags clears flag state that would otherwise persist on the
// shared rootCmd between Execute calls.
func resetEjectFlags(t *testing.T) {
	t.Helper()
	for k, v := range map[string]string{"force": "false", "name": "default", "list": "false"} {
		if err := themeEjectCmd.Flags().Set(k, v); err != nil {
			t.Fatalf("resetting flag %s: %v", k, err)
		}
	}
}

// chdirTemp (shared test helper) lives in plugin_install_test.go.

func runEject(args ...string) error {
	rootCmd.SetArgs(append([]string{"theme", "eject"}, args...))
	return rootCmd.Execute()
}

func mustExist(t *testing.T, base string, rels ...string) {
	t.Helper()
	for _, rel := range rels {
		if _, err := os.Stat(filepath.Join(base, filepath.FromSlash(rel))); err != nil {
			t.Errorf("expected %s to exist: %v", rel, err)
		}
	}
}

func mustNotExist(t *testing.T, base string, rels ...string) {
	t.Helper()
	for _, rel := range rels {
		if _, err := os.Stat(filepath.Join(base, filepath.FromSlash(rel))); err == nil {
			t.Errorf("expected %s to be absent", rel)
		}
	}
}

// countFiles returns the number of regular files under dir (0 if missing).
func countFiles(t *testing.T, dir string) int {
	t.Helper()
	n := 0
	filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			n++
		}
		return nil
	})
	return n
}

func TestRunThemeEject(t *testing.T) {
	resetEjectFlags(t)
	dir := chdirTemp(t)

	if err := runEject(); err != nil {
		t.Fatalf("theme eject failed: %v", err)
	}

	themeDir := filepath.Join(dir, consts.DirThemes, "default")
	mustExist(t, themeDir,
		consts.FileThemeConfig,
		"layouts/_default/"+consts.TemplateBaseOf,
		"layouts/_default/home.html",
		"layouts/_default/single.html",
		"layouts/_docs/"+consts.TemplateBaseOf,
		"layouts/components/Header.html",
		"layouts/partials/seo.html",
		"layouts/_taxonomy/term.html",
		"layouts/_labs/"+consts.TemplateBaseOf,
		"layouts/_presentation/"+consts.TemplateBaseOf,
		"layouts/_slides/list.html",
		"layouts/shortcodes/alert.html",
		"css/tokens.css",
		"assets/js/tabs.js",
	)

	// Layout directories must land under layouts/ only; nothing at the theme root.
	mustNotExist(t, themeDir, themeLayoutDirs...)
}

// TestEmbeddedThemeRootIsFullyMapped guards against a new embedded layout
// directory being added without teaching eject to place it under layouts/.
func TestEmbeddedThemeRootIsFullyMapped(t *testing.T) {
	entries, err := fs.ReadDir(embedded.ThemeFS(), ".")
	if err != nil {
		t.Fatalf("reading embedded theme root: %v", err)
	}

	passthrough := map[string]bool{
		"css":                  true,
		consts.DirAssets:       true,
		consts.FileThemeConfig: true,
	}

	for _, e := range entries {
		name := e.Name()
		if passthrough[name] {
			continue
		}
		if !e.IsDir() {
			t.Errorf("unexpected file at embedded theme root: %s", name)
			continue
		}
		want := filepath.Join(consts.DirLayouts, name)
		if got := remapThemePath(name); got != want {
			t.Errorf("embedded root dir %q is not remapped: remapThemePath(%q) = %q, want %q", name, name, got, want)
		}
	}
}

func TestRunThemeEject_AlreadyExists(t *testing.T) {
	resetEjectFlags(t)
	dir := chdirTemp(t)

	os.MkdirAll(filepath.Join(dir, consts.DirThemes, "default"), 0o755)

	if err := runEject(); err == nil {
		t.Fatal("expected error for existing themes/default/, got nil")
	}
}

func TestRunThemeEject_Force(t *testing.T) {
	resetEjectFlags(t)
	dir := chdirTemp(t)

	if err := runEject(); err != nil {
		t.Fatalf("first eject failed: %v", err)
	}
	if err := runEject("--force"); err != nil {
		t.Fatalf("force eject failed: %v", err)
	}

	mustExist(t, filepath.Join(dir, consts.DirThemes, "default"), consts.FileThemeConfig)
}

func TestRunThemeEject_FullWithName(t *testing.T) {
	resetEjectFlags(t)
	dir := chdirTemp(t)

	if err := runEject("--name", "magazine"); err != nil {
		t.Fatalf("eject --name failed: %v", err)
	}

	themeDir := filepath.Join(dir, consts.DirThemes, "magazine")
	mustExist(t, themeDir, consts.FileThemeConfig, "layouts/_blog/single.html")
	mustNotExist(t, dir, "themes/default")

	data, _ := os.ReadFile(filepath.Join(themeDir, consts.FileThemeConfig))
	if !strings.Contains(string(data), "slug: magazine") {
		t.Errorf("theme.yaml slug not rewritten:\n%s", data)
	}
}

func TestRunThemeEject_SingleTemplate(t *testing.T) {
	resetEjectFlags(t)
	dir := chdirTemp(t)

	if err := runEject("layouts/_blog/single.html"); err != nil {
		t.Fatalf("selective eject failed: %v", err)
	}

	themeDir := filepath.Join(dir, consts.DirThemes, "default")
	mustExist(t, themeDir, "layouts/_blog/single.html", consts.FileThemeConfig)
	mustNotExist(t, themeDir, "layouts/_blog/list.html", "layouts/_default", "css", "assets")

	if n := countFiles(t, themeDir); n != 2 {
		t.Errorf("expected exactly 2 files (template + theme.yaml), got %d", n)
	}
}

func TestRunThemeEject_Directory(t *testing.T) {
	resetEjectFlags(t)
	dir := chdirTemp(t)

	if err := runEject("layouts/components"); err != nil {
		t.Fatalf("directory eject failed: %v", err)
	}

	themeDir := filepath.Join(dir, consts.DirThemes, "default")
	mustExist(t, themeDir, "layouts/components/Header.html", "layouts/components/Footer.html")
	mustNotExist(t, themeDir, "layouts/partials", "layouts/_default", "css")

	embeddedComponents, _ := fs.ReadDir(embedded.ThemeFS(), consts.DirComponents)
	if got := countFiles(t, filepath.Join(themeDir, "layouts", "components")); got != len(embeddedComponents) {
		t.Errorf("expected %d components, got %d", len(embeddedComponents), got)
	}
}

func TestRunThemeEject_EmbeddedFormPathAccepted(t *testing.T) {
	resetEjectFlags(t)
	dir := chdirTemp(t)

	if err := runEject("_blog/single.html", `components\Footer.html`); err != nil {
		t.Fatalf("eject with embedded-form paths failed: %v", err)
	}

	themeDir := filepath.Join(dir, consts.DirThemes, "default")
	mustExist(t, themeDir, "layouts/_blog/single.html", "layouts/components/Footer.html")
	mustNotExist(t, themeDir, "_blog", "components")
}

func TestRunThemeEject_SingleStylesheet(t *testing.T) {
	resetEjectFlags(t)
	dir := chdirTemp(t)

	if err := runEject("css/blog.css"); err != nil {
		t.Fatalf("css eject failed: %v", err)
	}

	themeDir := filepath.Join(dir, consts.DirThemes, "default")
	mustExist(t, themeDir, "css/blog.css")
	mustNotExist(t, themeDir, "css/tokens.css")
}

func TestRunThemeEject_AssetsSubpathRefused(t *testing.T) {
	resetEjectFlags(t)
	dir := chdirTemp(t)

	err := runEject("assets/js")
	if err == nil || !strings.Contains(err.Error(), "as a whole") {
		t.Fatalf("expected assets sub-path to be refused, got: %v", err)
	}
	mustNotExist(t, dir, "themes")

	if err := runEject("assets"); err != nil {
		t.Fatalf("whole assets eject failed: %v", err)
	}
	mustExist(t, filepath.Join(dir, consts.DirThemes, "default"), "assets/js/tabs.js", "assets/fonts")
}

func TestRunThemeEject_UnknownPath(t *testing.T) {
	resetEjectFlags(t)
	dir := chdirTemp(t)

	err := runEject("layouts/_blog/nope.html")
	if err == nil || !strings.Contains(err.Error(), "no such file in the embedded theme") {
		t.Fatalf("expected unknown-path error, got: %v", err)
	}
	mustNotExist(t, dir, "themes")
}

func TestRunThemeEject_ConflictAndForce(t *testing.T) {
	resetEjectFlags(t)
	dir := chdirTemp(t)

	themeDir := filepath.Join(dir, consts.DirThemes, "default")
	target := filepath.Join(themeDir, "layouts", "_blog", "single.html")
	sibling := filepath.Join(themeDir, "layouts", "_blog", "list.html")
	os.MkdirAll(filepath.Dir(target), 0o755)
	os.WriteFile(target, []byte("mine"), 0o644)
	os.WriteFile(sibling, []byte("keep"), 0o644)

	err := runEject("layouts/_blog/single.html")
	if err == nil || !strings.Contains(err.Error(), "layouts/_blog/single.html") {
		t.Fatalf("expected conflict error naming the file, got: %v", err)
	}
	if data, _ := os.ReadFile(target); string(data) != "mine" {
		t.Error("existing file was overwritten without --force")
	}

	if err := runEject("--force", "layouts/_blog/single.html"); err != nil {
		t.Fatalf("force eject failed: %v", err)
	}
	if data, _ := os.ReadFile(target); string(data) == "mine" {
		t.Error("--force did not overwrite the requested file")
	}
	if data, _ := os.ReadFile(sibling); string(data) != "keep" {
		t.Error("--force touched a file that was not requested")
	}
}

func TestRunThemeEject_SelectiveWithName(t *testing.T) {
	resetEjectFlags(t)
	dir := chdirTemp(t)

	if err := runEject("--name", "magazine", "css/blog.css"); err != nil {
		t.Fatalf("eject --name failed: %v", err)
	}

	themeDir := filepath.Join(dir, consts.DirThemes, "magazine")
	mustExist(t, themeDir, "css/blog.css", consts.FileThemeConfig)
	data, _ := os.ReadFile(filepath.Join(themeDir, consts.FileThemeConfig))
	if !strings.Contains(string(data), "slug: magazine") {
		t.Errorf("theme.yaml slug not rewritten:\n%s", data)
	}
	if strings.Contains(string(data), "slug: default") {
		t.Error("old slug line still present")
	}
}

func TestRunThemeEject_ExistingThemeYAMLUntouched(t *testing.T) {
	resetEjectFlags(t)
	dir := chdirTemp(t)

	themeDir := filepath.Join(dir, consts.DirThemes, "magazine")
	os.MkdirAll(themeDir, 0o755)
	os.WriteFile(filepath.Join(themeDir, consts.FileThemeConfig), []byte("name: Mine\nslug: magazine\n"), 0o644)

	if err := runEject("--name", "magazine", "layouts/_blog/single.html"); err != nil {
		t.Fatalf("eject failed: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(themeDir, consts.FileThemeConfig))
	if string(data) != "name: Mine\nslug: magazine\n" {
		t.Errorf("existing theme.yaml was modified:\n%s", data)
	}
}

func TestRunThemeEject_InvalidName(t *testing.T) {
	resetEjectFlags(t)
	chdirTemp(t)

	for _, bad := range []string{"../x", "a/b", "..", ""} {
		if err := runEject("--name", bad, "css/blog.css"); err == nil {
			t.Errorf("expected --name %q to be rejected", bad)
		}
	}
}

func TestEjectablePaths(t *testing.T) {
	paths, err := ejectablePaths()
	if err != nil {
		t.Fatal(err)
	}
	set := make(map[string]bool, len(paths))
	for _, p := range paths {
		set[p] = true
		if strings.Contains(p, `\`) {
			t.Errorf("path %q is not slash-separated", p)
		}
		for _, dir := range themeLayoutDirs {
			if strings.HasPrefix(p, dir+"/") {
				t.Errorf("path %q should be under layouts/", p)
			}
		}
	}
	for _, want := range []string{"layouts/_taxonomy/term.html", "layouts/shortcodes/alert.html", "css/tokens.css", "assets/js/tabs.js", consts.FileThemeConfig} {
		if !set[want] {
			t.Errorf("missing %q", want)
		}
	}
}

func TestNormalizeEjectPath(t *testing.T) {
	tests := map[string]string{
		"layouts/_blog/single.html": "layouts/_blog/single.html",
		"_blog/single.html":         "layouts/_blog/single.html",
		"components/Footer.html":    "layouts/components/Footer.html",
		"components":                "layouts/components",
		`layouts\_docs\single.html`: "layouts/_docs/single.html",
		"./css/blog.css":            "css/blog.css",
		"css/":                      "css",
		"theme.yaml":                "theme.yaml",
		"assets":                    "assets",
	}
	for in, want := range tests {
		if got := normalizeEjectPath(in); got != want {
			t.Errorf("normalizeEjectPath(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestEmbeddedPathFor(t *testing.T) {
	tests := map[string]string{
		"layouts/_blog/single.html":      "_blog/single.html",
		"layouts/components/Footer.html": "components/Footer.html",
		"layouts/shortcodes/alert.html":  "shortcodes/alert.html",
		"css/blog.css":                   "css/blog.css",
		"theme.yaml":                     "theme.yaml",
	}
	for in, want := range tests {
		if got := embeddedPathFor(in); got != want {
			t.Errorf("embeddedPathFor(%q) = %q, want %q", in, got, want)
		}
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
		{"_taxonomy/term.html", filepath.Join(consts.DirLayouts, "_taxonomy/term.html")},
		{"_labs/baseof.html", filepath.Join(consts.DirLayouts, "_labs/baseof.html")},
		{"_presentation/baseof.html", filepath.Join(consts.DirLayouts, "_presentation/baseof.html")},
		{"_slides/list.html", filepath.Join(consts.DirLayouts, "_slides/list.html")},
		{"shortcodes/alert.html", filepath.Join(consts.DirLayouts, "shortcodes/alert.html")},
		// Bare directory entries (as yielded by fs.WalkDir) must remap too,
		// otherwise eject leaves empty directories at the theme root.
		{"_default", filepath.Join(consts.DirLayouts, "_default")},
		{"shortcodes", filepath.Join(consts.DirLayouts, "shortcodes")},
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
