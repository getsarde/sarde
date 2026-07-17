package cli

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	"github.com/getsarde/sarde/internal/consts"
)

const testPluginManifest = "name: TestCard\nslug: testcard\nversion: 1.0.0\ninject:\n  styles: [css/testcard.css]\n"

func writeTestPluginDir(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, "assets", "css"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, consts.FilePluginManifest), []byte(testPluginManifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "assets", "css", "testcard.css"), []byte(".x{}"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeTestPluginZip(t *testing.T, zipPath string, topFolder string) {
	t.Helper()
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	w := zip.NewWriter(f)
	prefix := ""
	if topFolder != "" {
		prefix = topFolder + "/"
	}
	files := map[string]string{
		prefix + "plugin.yaml":             testPluginManifest,
		prefix + "assets/css/testcard.css": ".x{}",
	}
	for name, content := range files {
		fw, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := fw.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
}

func chdirTemp(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origWd) })
	return dir
}

func TestRunPluginInstall_LocalDir(t *testing.T) {
	dir := chdirTemp(t)

	srcDir := filepath.Join(dir, "some-folder-name")
	writeTestPluginDir(t, srcDir)

	cmd := rootCmd
	cmd.SetArgs([]string{"plugin", "install", srcDir})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("plugin install failed: %v", err)
	}

	// Destination is named after the manifest slug, not the source folder.
	installed := filepath.Join(dir, consts.DirPlugins, "testcard")
	if _, err := os.Stat(filepath.Join(installed, consts.FilePluginManifest)); err != nil {
		t.Error("plugin.yaml not found in installed plugin")
	}
	if _, err := os.Stat(filepath.Join(installed, "assets", "css", "testcard.css")); err != nil {
		t.Error("asset not found in installed plugin")
	}
}

func TestRunPluginInstall_LocalZipWithTopFolder(t *testing.T) {
	dir := chdirTemp(t)

	zipPath := filepath.Join(dir, "testcard-1.0.0.zip")
	writeTestPluginZip(t, zipPath, "testcard-release")

	cmd := rootCmd
	cmd.SetArgs([]string{"plugin", "install", zipPath})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("plugin install from zip failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, consts.DirPlugins, "testcard", consts.FilePluginManifest)); err != nil {
		t.Error("plugin not installed from zip with wrapping folder")
	}
}

func TestRunPluginInstall_AlreadyExists(t *testing.T) {
	dir := chdirTemp(t)

	srcDir := filepath.Join(dir, "src")
	writeTestPluginDir(t, srcDir)
	if err := os.MkdirAll(filepath.Join(dir, consts.DirPlugins, "testcard"), 0o755); err != nil {
		t.Fatal(err)
	}

	cmd := rootCmd
	cmd.SetArgs([]string{"plugin", "install", srcDir})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error for already-installed plugin, got nil")
	}
}

func TestRunPluginInstall_NoManifest(t *testing.T) {
	dir := chdirTemp(t)

	srcDir := filepath.Join(dir, "not-a-plugin")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "readme.md"), []byte("# nope"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := rootCmd
	cmd.SetArgs([]string{"plugin", "install", srcDir})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error for directory without plugin.yaml, got nil")
	}

	entries, _ := os.ReadDir(filepath.Join(dir, consts.DirPlugins))
	if len(entries) != 0 {
		t.Error("nothing should have been installed after a failed install")
	}
}

func TestRunPluginInstall_ReservedSlug(t *testing.T) {
	dir := chdirTemp(t)

	srcDir := filepath.Join(dir, "src")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := "name: Fake\nslug: sitemap\nversion: 1.0.0\n"
	if err := os.WriteFile(filepath.Join(srcDir, consts.FilePluginManifest), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := rootCmd
	cmd.SetArgs([]string{"plugin", "install", srcDir})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error for reserved slug, got nil")
	}
}

func TestRunPluginRemove(t *testing.T) {
	dir := chdirTemp(t)

	writeTestPluginDir(t, filepath.Join(dir, consts.DirPlugins, "testcard"))

	cmd := rootCmd
	cmd.SetArgs([]string{"plugin", "remove", "testcard"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("plugin remove failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, consts.DirPlugins, "testcard")); err == nil {
		t.Error("plugin directory should have been removed")
	}

	cmd.SetArgs([]string{"plugin", "remove", "testcard"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error when removing a plugin that is not installed")
	}
}
