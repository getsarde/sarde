package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/getsarde/sarde/embedded"
	"github.com/getsarde/sarde/internal/config"
)

const testcardManifest = `name: TestCard
slug: testcard
version: 1.0.0
description: Test fixture plugin
author: tester
inject:
  when: collection
  collection: blog
  styles: [css/testcard.css]
`

func writeExternalPluginFixture(t *testing.T, projDir string) {
	t.Helper()
	writeFixture(t, projDir, "plugins/testcard/plugin.yaml", testcardManifest)
	writeFixture(t, projDir, "plugins/testcard/assets/css/testcard.css", ".testcard{color:red}\n")
	writeFixture(t, projDir, "plugins/testcard/templates/shortcodes/testcard.html",
		`<div class="testcard-shortcode">Hello from plugin</div>`)
}

func TestBuild_ExternalPlugin_AssetsAndShortcode(t *testing.T) {
	projDir := createFixtureSite(t)
	writeExternalPluginFixture(t, projDir)
	writeFixture(t, projDir, "content/blog/with-shortcode.md",
		"---\ntitle: With Shortcode\ndate: 2025-02-01T00:00:00Z\n---\n\n{{< testcard />}}\n")

	cfg := config.Defaults()
	themeCfg := buildThemeConfig()

	builder := NewSiteBuilder(BuildOptions{
		ProjectDir:  projDir,
		Config:      cfg,
		ThemeConfig: themeCfg,
		EmbeddedFS:  embedded.ThemeFS(),
	})

	result, err := builder.Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	for _, w := range result.Warnings {
		if w.Field == "plugin" {
			t.Errorf("unexpected plugin warning: %+v", w)
		}
	}

	distDir := filepath.Join(projDir, "dist")

	// The plugin's asset tree is copied under the default vendor prefix.
	if _, err := os.Stat(filepath.Join(distDir, "assets", "vendor", "testcard", "css", "testcard.css")); err != nil {
		t.Errorf("plugin CSS not copied to dist: %v", err)
	}

	// Collection-gated injection: blog pages get the stylesheet link.
	blogHTML := readFixture(t, distDir, "blog/hello-world/index.html")
	if !strings.Contains(blogHTML, "/assets/vendor/testcard/css/testcard.css") {
		t.Error("expected testcard stylesheet link on blog page")
	}

	// Docs pages are a different collection: no injection.
	docsHTML := readFixture(t, distDir, "docs/getting-started/index.html")
	if strings.Contains(docsHTML, "testcard.css") {
		t.Error("did not expect testcard stylesheet link on docs page")
	}

	// The plugin-shipped shortcode expands in markdown.
	scHTML := readFixture(t, distDir, "blog/with-shortcode/index.html")
	if !strings.Contains(scHTML, "testcard-shortcode") {
		t.Error("expected plugin shortcode output in rendered page")
	}
}

func TestBuild_ExternalPlugin_DisabledViaConfig(t *testing.T) {
	projDir := createFixtureSite(t)
	writeExternalPluginFixture(t, projDir)

	cfg := config.Defaults()
	cfg.Plugins.Disabled = []string{"testcard"}
	themeCfg := buildThemeConfig()

	builder := NewSiteBuilder(BuildOptions{
		ProjectDir:  projDir,
		Config:      cfg,
		ThemeConfig: themeCfg,
		EmbeddedFS:  embedded.ThemeFS(),
	})

	if _, err := builder.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	distDir := filepath.Join(projDir, "dist")
	if _, err := os.Stat(filepath.Join(distDir, "assets", "vendor", "testcard")); err == nil {
		t.Error("disabled plugin's assets should not be copied to dist")
	}
	blogHTML := readFixture(t, distDir, "blog/hello-world/index.html")
	if strings.Contains(blogHTML, "testcard.css") {
		t.Error("disabled plugin should not inject assets")
	}
}

func TestBuild_ExternalPlugin_PremiumWithoutLicenseWarns(t *testing.T) {
	projDir := createFixtureSite(t)
	writeFixture(t, projDir, "plugins/premo/plugin.yaml",
		"name: Premo\nslug: premo\nversion: 1.0.0\npremium: true\ninject:\n  styles: [css/premo.css]\n")
	writeFixture(t, projDir, "plugins/premo/assets/css/premo.css", ".premo{}\n")

	cfg := config.Defaults()
	themeCfg := buildThemeConfig()

	builder := NewSiteBuilder(BuildOptions{
		ProjectDir:  projDir,
		Config:      cfg,
		ThemeConfig: themeCfg,
		EmbeddedFS:  embedded.ThemeFS(),
	})

	result, err := builder.Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	found := false
	for _, w := range result.Warnings {
		if w.Field == "plugin" && strings.Contains(w.Message, "premo") && strings.Contains(w.Message, "license") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a license warning for premium plugin, warnings: %+v", result.Warnings)
	}

	// The build succeeds and the plugin stays inactive.
	distDir := filepath.Join(projDir, "dist")
	if _, err := os.Stat(filepath.Join(distDir, "assets", "vendor", "premo")); err == nil {
		t.Error("unlicensed premium plugin's assets should not be copied")
	}
}
