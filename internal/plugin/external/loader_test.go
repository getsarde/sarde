package external

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/engine"
	"github.com/getsarde/sarde/internal/license"
	"github.com/getsarde/sarde/internal/plugin"
)

func writePluginFixture(t *testing.T, projectDir, slug, manifest string) string {
	t.Helper()
	dir := filepath.Join(projectDir, "plugins", slug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "plugin.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func freeManifest(slug string) string {
	return "name: " + strings.ToUpper(slug) + "\nslug: " + slug + "\nversion: 1.0.0\ninject:\n  styles: [css/x.css]\n"
}

func TestLoadAllIndependentPlugins(t *testing.T) {
	project := t.TempDir()

	// One valid free plugin with a templates dir.
	goodDir := writePluginFixture(t, project, "good", freeManifest("good"))
	if err := os.MkdirAll(filepath.Join(goodDir, "templates", "partials"), 0o755); err != nil {
		t.Fatal(err)
	}

	// One malformed manifest.
	writePluginFixture(t, project, "broken", "name: [unclosed\n")

	// One premium plugin without a license.
	writePluginFixture(t, project, "premium", "name: P\nslug: premium\nversion: 1.0.0\npremium: true\npurchase_url: https://example.com/buy\n")

	// One plugin whose slug collides with a built-in name.
	writePluginFixture(t, project, "sitemap", "name: S\nslug: sitemap\nversion: 1.0.0\n")

	mgr := plugin.NewManager()
	cfg := config.Defaults()
	tplDirs, dirDirs, warnings := LoadAll(mgr, project, cfg, []string{"sitemap", "search"})

	if len(tplDirs) != 1 || !strings.HasSuffix(tplDirs[0], filepath.Join("good", "templates")) {
		t.Errorf("expected only good's templates dir, got %v", tplDirs)
	}
	if len(dirDirs) != 0 {
		t.Errorf("expected no directive dirs, got %v", dirDirs)
	}

	wantWarnings := map[string]bool{"broken": false, "premium": false, "sitemap": false}
	for _, w := range warnings {
		for slug := range wantWarnings {
			if strings.Contains(w.File, slug) {
				wantWarnings[slug] = true
			}
		}
	}
	for slug, found := range wantWarnings {
		if !found {
			t.Errorf("expected a warning mentioning %q, warnings: %+v", slug, warnings)
		}
	}
	if len(warnings) != 3 {
		t.Errorf("expected exactly 3 warnings, got %d: %+v", len(warnings), warnings)
	}
}

func TestLoadAllDisabledSkipped(t *testing.T) {
	project := t.TempDir()
	writePluginFixture(t, project, "muted", freeManifest("muted"))

	mgr := plugin.NewManager()
	cfg := config.Defaults()
	cfg.Plugins.Disabled = []string{"muted"}
	tplDirs, dirDirs, warnings := LoadAll(mgr, project, cfg, nil)
	if len(tplDirs) != 0 || len(dirDirs) != 0 || len(warnings) != 0 {
		t.Errorf("disabled plugin should be silently skipped, got dirs=%v directiveDirs=%v warnings=%v", tplDirs, dirDirs, warnings)
	}
}

func TestLoadAllPremiumWithValidLicense(t *testing.T) {
	project := t.TempDir()
	writePluginFixture(t, project, "prem", "name: P\nslug: prem\nversion: 1.0.0\npremium: true\n")

	// A valid license only exists relative to the embedded public key, which
	// is a placeholder. Write a license file signed with a throwaway key and
	// confirm the plugin is still skipped (invalid signature), proving the
	// license path is exercised.
	licDir := filepath.Join(project, ".sarde", "licenses")
	if err := os.MkdirAll(licDir, 0o755); err != nil {
		t.Fatal(err)
	}
	f := &license.File{V: 1, Slug: "prem", Licensee: "x@example.com", Issued: "2026-01-01", Sig: "AAAA"}
	data, _ := json.Marshal(f)
	if err := os.WriteFile(filepath.Join(licDir, "prem.license"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	mgr := plugin.NewManager()
	tplDirs, dirDirs, warnings := LoadAll(mgr, project, config.Defaults(), nil)
	if len(tplDirs) != 0 || len(dirDirs) != 0 {
		t.Errorf("expected no template or directive dirs, got %v / %v", tplDirs, dirDirs)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0].Message, "signature") {
		t.Errorf("expected an invalid-signature warning, got %+v", warnings)
	}
}

func TestLoadAllDirectiveDirs(t *testing.T) {
	project := t.TempDir()

	// A free plugin shipping a directive.
	withDir := writePluginFixture(t, project, "withdir", freeManifest("withdir"))
	if err := os.MkdirAll(filepath.Join(withDir, "directives"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(withDir, "directives", "callout.yaml"), []byte("name: callout\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// A premium plugin without a license also shipping a directive: its dir
	// must not be collected.
	premDir := writePluginFixture(t, project, "premdir", "name: P\nslug: premdir\nversion: 1.0.0\npremium: true\n")
	if err := os.MkdirAll(filepath.Join(premDir, "directives"), 0o755); err != nil {
		t.Fatal(err)
	}

	mgr := plugin.NewManager()
	_, dirDirs, warnings := LoadAll(mgr, project, config.Defaults(), nil)
	if len(dirDirs) != 1 || !strings.HasSuffix(dirDirs[0], filepath.Join("withdir", "directives")) {
		t.Errorf("expected only withdir's directives dir, got %v", dirDirs)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0].File, "premdir") {
		t.Errorf("expected exactly the premium warning, got %+v", warnings)
	}
}

func TestDirectiveCollisionWarnings(t *testing.T) {
	project := t.TempDir()
	for _, slug := range []string{"alpha", "beta"} {
		dir := writePluginFixture(t, project, slug, freeManifest(slug))
		if err := os.MkdirAll(filepath.Join(dir, "directives"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "directives", "callout.yaml"), []byte("name: callout\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	mgr := plugin.NewManager()
	_, dirDirs, warnings := LoadAll(mgr, project, config.Defaults(), nil)
	if len(dirDirs) != 2 {
		t.Fatalf("expected both directive dirs, got %v", dirDirs)
	}
	if len(warnings) != 1 {
		t.Fatalf("expected exactly one collision warning, got %+v", warnings)
	}
	msg := warnings[0].Message
	if !strings.Contains(msg, "callout.yaml") || !strings.Contains(msg, "alpha, beta") || !strings.Contains(msg, "beta wins") {
		t.Errorf("unexpected collision message: %q", msg)
	}
}

func TestNewExternalPluginHooks(t *testing.T) {
	project := t.TempDir()
	dir := writePluginFixture(t, project, "inj",
		"name: I\nslug: inj\nversion: 1.0.0\ninject:\n  when: layout\n  layout: presentation\n  styles: [css/inj.css]\n  module_scripts: [js/inj.js]\n")

	m, err := LoadManifest(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Validate("inj"); err != nil {
		t.Fatal(err)
	}
	p := newExternalPlugin(m, dir, nil)

	page := &engine.Page{}
	page.Kind = engine.KindPage

	// Non-matching layout: nothing injected.
	rd := &engine.RouteData{Layout: engine.LayoutDocs}
	if err := p.Hooks.BeforeRender(&plugin.BeforeRenderContext{Page: page, RouteData: rd}); err != nil {
		t.Fatal(err)
	}
	if len(rd.Styles) != 0 || len(rd.ModuleScripts) != 0 {
		t.Errorf("expected no injection on docs layout, got styles=%v modules=%v", rd.Styles, rd.ModuleScripts)
	}

	// Matching layout: URLs appended under the default vendor prefix.
	rd = &engine.RouteData{Layout: engine.LayoutPresentation}
	if err := p.Hooks.BeforeRender(&plugin.BeforeRenderContext{Page: page, RouteData: rd}); err != nil {
		t.Fatal(err)
	}
	if len(rd.Styles) != 1 || rd.Styles[0] != "/assets/vendor/inj/css/inj.css" {
		t.Errorf("unexpected styles: %v", rd.Styles)
	}
	if len(rd.ModuleScripts) != 1 || rd.ModuleScripts[0] != "/assets/vendor/inj/js/inj.js" {
		t.Errorf("unexpected module scripts: %v", rd.ModuleScripts)
	}

	// Config always: true overrides the layout gate.
	pAlways := newExternalPlugin(m, dir, map[string]any{"always": true})
	rd = &engine.RouteData{Layout: engine.LayoutDocs}
	if err := pAlways.Hooks.BeforeRender(&plugin.BeforeRenderContext{Page: page, RouteData: rd}); err != nil {
		t.Fatal(err)
	}
	if len(rd.Styles) != 1 {
		t.Errorf("expected injection with always: true, got %v", rd.Styles)
	}
}
