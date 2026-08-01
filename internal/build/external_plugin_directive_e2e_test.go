package build

import (
	"strings"
	"testing"

	"github.com/getsarde/sarde/embedded"
	"github.com/getsarde/sarde/internal/config"
)

const calloutPluginManifest = `name: CalloutPack
slug: calloutpack
version: 1.0.0
description: Test fixture plugin shipping a directive
`

// writePluginDirectiveFixture writes an external plugin that ships one
// generic container directive (callout) with a CSS sidecar, plus a docs page
// that uses it.
func writePluginDirectiveFixture(t *testing.T, projDir string) {
	t.Helper()
	writeFixture(t, projDir, "plugins/calloutpack/plugin.yaml", calloutPluginManifest)
	writeFixture(t, projDir, "plugins/calloutpack/directives/callout.yaml", `name: callout
kind: container
label: "Callout"
description: "A plugin-shipped callout box"
fields:
  - { name: tone, label: Tone, type: string }
`)
	writeFixture(t, projDir, "plugins/calloutpack/directives/callout.html",
		`<div class="plugin-callout" data-tone="{{.Attrs.tone}}">{{.Body}}</div>`)
	writeFixture(t, projDir, "plugins/calloutpack/directives/callout.css",
		".plugin-callout { border: 1px solid var(--sd-accent); }")
	writeFixture(t, projDir, "content/docs/callouts.md", `---
title: Callouts
---
# Callouts

:::callout tone="info"

Plugin directives are **live**.

:::
`)
}

func TestBuild_ExternalPluginDirective_RendersWithCSS(t *testing.T) {
	projDir := createFixtureSite(t)
	writePluginDirectiveFixture(t, projDir)

	builder := NewSiteBuilder(BuildOptions{
		ProjectDir:  projDir,
		Config:      config.Defaults(),
		ThemeConfig: buildThemeConfig(),
		EmbeddedFS:  embedded.ThemeFS(),
	})
	result, err := builder.Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	for _, w := range result.Warnings {
		if w.Field == "plugin" || w.Field == "directive" {
			t.Errorf("unexpected warning: %+v", w)
		}
	}

	html := readBuiltPage(t, projDir, "docs/callouts/index.html")
	for _, want := range []string{
		`class="plugin-callout"`,
		`data-tone="info"`,
		"<strong>live</strong>",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("built page missing %q", want)
		}
	}

	css := readBundledCSS(t, projDir)
	if !strings.Contains(css, "plugin-callout") {
		t.Error("plugin directive CSS missing from the sarde.<hash>.css bundle")
	}
}

func TestBuild_ExternalPluginDirective_SiteOverrideWins(t *testing.T) {
	projDir := createFixtureSite(t)
	writePluginDirectiveFixture(t, projDir)

	// Same directive name defined at site level: site wins, silently
	// (intentional overriding, not a collision).
	writeFixture(t, projDir, "directives/callout.yaml", `name: callout
kind: container
label: "Site Callout"
description: "site-level override"
`)
	writeFixture(t, projDir, "directives/callout.html",
		`<div class="site-callout">{{.Body}}</div>`)

	builder := NewSiteBuilder(BuildOptions{
		ProjectDir:  projDir,
		Config:      config.Defaults(),
		ThemeConfig: buildThemeConfig(),
		EmbeddedFS:  embedded.ThemeFS(),
	})
	result, err := builder.Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	for _, w := range result.Warnings {
		if w.Field == "plugin" || w.Field == "directive" {
			t.Errorf("unexpected warning: %+v", w)
		}
	}

	html := readBuiltPage(t, projDir, "docs/callouts/index.html")
	if !strings.Contains(html, "site-callout") {
		t.Error("site override must win over the plugin directive")
	}
	if strings.Contains(html, "plugin-callout") {
		t.Error("shadowed plugin directive must not render")
	}
}

func TestBuild_ExternalPluginDirective_DisabledPlugin(t *testing.T) {
	projDir := createFixtureSite(t)
	writePluginDirectiveFixture(t, projDir)

	cfg := config.Defaults()
	cfg.Plugins.Disabled = []string{"calloutpack"}

	builder := NewSiteBuilder(BuildOptions{
		ProjectDir:  projDir,
		Config:      cfg,
		ThemeConfig: buildThemeConfig(),
		EmbeddedFS:  embedded.ThemeFS(),
	})
	if _, err := builder.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Unregistered directives fall through as plain text; the plugin's
	// markup and CSS must be absent.
	html := readBuiltPage(t, projDir, "docs/callouts/index.html")
	if strings.Contains(html, "plugin-callout") {
		t.Error("disabled plugin's directive must not render")
	}
	css := readBundledCSS(t, projDir)
	if strings.Contains(css, "plugin-callout") {
		t.Error("disabled plugin's directive CSS must not ship")
	}
}

func TestBuild_ExternalPluginDirective_BuiltinCollision(t *testing.T) {
	projDir := createFixtureSite(t)
	writeFixture(t, projDir, "plugins/cardpack/plugin.yaml",
		"name: CardPack\nslug: cardpack\nversion: 1.0.0\n")
	writeFixture(t, projDir, "plugins/cardpack/directives/card.yaml", `name: card
kind: container
label: "Fake Card"
description: "collides with the built-in card"
`)
	writeFixture(t, projDir, "plugins/cardpack/directives/card.html",
		`<div class="plugin-fake-card">{{.Body}}</div>`)
	writeFixture(t, projDir, "content/docs/carded.md", `---
title: Carded
---
:::card[Hello]

body

:::
`)

	builder := NewSiteBuilder(BuildOptions{
		ProjectDir:  projDir,
		Config:      config.Defaults(),
		ThemeConfig: buildThemeConfig(),
		EmbeddedFS:  embedded.ThemeFS(),
	})
	result, err := builder.Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	found := false
	for _, w := range result.Warnings {
		if w.Field == "directive" && strings.Contains(w.Message, "card") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a built-in collision warning for card, got %+v", result.Warnings)
	}

	html := readBuiltPage(t, projDir, "docs/carded/index.html")
	if !strings.Contains(html, "sarde-card") {
		t.Error("built-in card must still render on collision")
	}
	if strings.Contains(html, "plugin-fake-card") {
		t.Error("colliding plugin definition must not render")
	}
}
