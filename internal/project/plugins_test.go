package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/getsarde/sarde/embedded"
)

func writeProjectPlugin(t *testing.T, projectDir, slug, manifest string) {
	t.Helper()
	dir := filepath.Join(projectDir, "plugins", slug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "plugin.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestProjectManager_PluginsRoundTrip(t *testing.T) {
	dir := createTestProject(t)
	writeProjectPlugin(t, dir, "democard", "name: DemoCard\nslug: democard\nversion: 1.0.0\n")
	writeProjectPlugin(t, dir, "premo", "name: Premo\nslug: premo\nversion: 2.0.0\npremium: true\n")

	hub := NewEventHub()
	pm := NewProjectManager(hub, embedded.ThemeFS(), nil)
	if _, err := pm.OpenProject(dir); err != nil {
		t.Fatalf("OpenProject failed: %v", err)
	}
	defer pm.CloseProject()

	infos, err := pm.ListPlugins()
	if err != nil {
		t.Fatalf("ListPlugins failed: %v", err)
	}
	if len(infos) != 2 {
		t.Fatalf("expected 2 plugins, got %d: %+v", len(infos), infos)
	}

	byName := make(map[string]PluginInfo)
	for _, info := range infos {
		byName[info.Slug] = info
	}
	if info := byName["democard"]; !info.Enabled || !info.LicenseOK || info.Version != "1.0.0" {
		t.Errorf("unexpected democard info: %+v", info)
	}
	if info := byName["premo"]; !info.Premium || info.LicenseOK || info.LicenseMsg == "" {
		t.Errorf("premium plugin without license should report LicenseMsg: %+v", info)
	}

	// Disable, verify sarde.yaml and ListPlugins reflect it.
	if err := pm.DisablePlugin("democard"); err != nil {
		t.Fatalf("DisablePlugin failed: %v", err)
	}
	yamlData, _ := os.ReadFile(filepath.Join(dir, "sarde.yaml"))
	if !strings.Contains(string(yamlData), "democard") {
		t.Errorf("sarde.yaml should list democard under plugins.disabled:\n%s", yamlData)
	}
	infos, _ = pm.ListPlugins()
	for _, info := range infos {
		if info.Slug == "democard" && info.Enabled {
			t.Error("democard should be disabled after DisablePlugin")
		}
	}

	// Re-enable, the disabled entry disappears again.
	if err := pm.EnablePlugin("democard"); err != nil {
		t.Fatalf("EnablePlugin failed: %v", err)
	}
	yamlData, _ = os.ReadFile(filepath.Join(dir, "sarde.yaml"))
	if strings.Contains(string(yamlData), "democard") {
		t.Errorf("sarde.yaml should no longer mention democard:\n%s", yamlData)
	}

	// Remove deletes the plugin directory.
	if err := pm.RemovePlugin("democard"); err != nil {
		t.Fatalf("RemovePlugin failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "plugins", "democard")); err == nil {
		t.Error("plugin directory should have been removed")
	}
}

func TestProjectManager_PluginsClosedErrors(t *testing.T) {
	pm := NewProjectManager(NewEventHub(), embedded.ThemeFS(), nil)
	if _, err := pm.ListPlugins(); err == nil {
		t.Error("ListPlugins should fail with no project open")
	}
	if err := pm.DisablePlugin("x"); err == nil {
		t.Error("DisablePlugin should fail with no project open")
	}
}
