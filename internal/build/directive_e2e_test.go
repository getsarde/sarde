package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/getsarde/sarde/embedded"
	"github.com/getsarde/sarde/internal/config"
)

// createDirectiveFixtureSite creates a site with one generic container
// directive (pullquote) with a CSS sidecar, and a page that uses it.
func createDirectiveFixtureSite(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	writeFixture(t, dir, "content/_index.md", "---\ntitle: Home\n---\n# Welcome\n")
	writeFixture(t, dir, "content/docs/_index.md", "---\ntitle: Docs\n---\n")
	writeFixture(t, dir, "content/docs/quotes.md", `---
title: Quotes
---
# Quotes

:::pullquote author="Ada Lovelace"

The engine weaves **algebraic patterns**.

:::
`)

	writeFixture(t, dir, "directives/pullquote.yaml", `name: pullquote
kind: container
label: "Pull Quote"
description: "A styled pull quote"
fields:
  - { name: author, label: Author, type: string }
`)
	writeFixture(t, dir, "directives/pullquote.html",
		`<blockquote class="sarde-pullquote">{{.Body}}<cite>{{.Attrs.author}}</cite></blockquote>`)
	writeFixture(t, dir, "directives/pullquote.css",
		".sarde-pullquote { border-inline-start: 3px solid var(--sd-accent); }")

	return dir
}

func readBuiltPage(t *testing.T, projDir, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(projDir, "dist", filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("reading built page %s: %v", rel, err)
	}
	return string(data)
}

func readBundledCSS(t *testing.T, projDir string) string {
	t.Helper()
	cssDir := filepath.Join(projDir, "dist", "assets", "css")
	entries, err := os.ReadDir(cssDir)
	if err != nil {
		t.Fatalf("reading css dir: %v", err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "sarde.") && strings.HasSuffix(e.Name(), ".css") {
			data, err := os.ReadFile(filepath.Join(cssDir, e.Name()))
			if err != nil {
				t.Fatalf("reading %s: %v", e.Name(), err)
			}
			return string(data)
		}
	}
	t.Fatal("no sarde.<hash>.css bundle written")
	return ""
}

func TestBuild_GenericDirective_EndToEnd(t *testing.T) {
	projDir := createDirectiveFixtureSite(t)
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
		if w.Field == "directive" {
			t.Errorf("unexpected directive warning: %+v", w)
		}
	}

	html := readBuiltPage(t, projDir, "docs/quotes/index.html")
	for _, want := range []string{
		`class="sarde-pullquote"`,
		"<cite>Ada Lovelace</cite>",
		"<strong>algebraic patterns</strong>",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("built page missing %q", want)
		}
	}

	css := readBundledCSS(t, projDir)
	if !strings.Contains(css, "sarde-pullquote") {
		t.Error("directive CSS missing from the sarde.<hash>.css bundle")
	}
}

func TestBuild_GenericDirective_BuiltinCollision(t *testing.T) {
	projDir := createDirectiveFixtureSite(t)
	// A site definition colliding with the built-in card directive.
	writeFixture(t, projDir, "directives/card.yaml", `name: card
kind: container
label: "My Card"
description: "colliding definition"
`)
	writeFixture(t, projDir, "directives/card.html", `<div class="my-fake-card">{{.Body}}</div>`)
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
		t.Errorf("expected a collision warning for card, got %+v", result.Warnings)
	}

	html := readBuiltPage(t, projDir, "docs/carded/index.html")
	if !strings.Contains(html, "sarde-card") {
		t.Error("built-in card must still render on collision")
	}
	if strings.Contains(html, "my-fake-card") {
		t.Error("colliding site definition must not render")
	}
}

func TestBuild_GenericDirective_MalformedWarns(t *testing.T) {
	projDir := createDirectiveFixtureSite(t)
	writeFixture(t, projDir, "directives/broken.yaml", "name: broken\nkind: nonsense\nlabel: B\ndescription: d\n")
	writeFixture(t, projDir, "directives/broken.html", "<div></div>")

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
		if w.Field == "directive" && strings.Contains(w.Message, "invalid kind") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected an invalid-kind warning for broken.yaml, got %+v", result.Warnings)
	}

	// The valid directive still loads and renders.
	html := readBuiltPage(t, projDir, "docs/quotes/index.html")
	if !strings.Contains(html, `class="sarde-pullquote"`) {
		t.Error("valid directive must survive a sibling's failure")
	}
}
