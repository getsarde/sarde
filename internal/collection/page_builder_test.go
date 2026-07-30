package collection

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/getsarde/sarde/internal/content"
	"github.com/getsarde/sarde/internal/engine"
)

// buildTestPage writes a markdown file and runs it through buildPage.
func buildTestPage(t *testing.T, markdown string) *engine.Page {
	t.Helper()
	contentDir := t.TempDir()
	blogDir := filepath.Join(contentDir, "blog")
	if err := os.MkdirAll(blogDir, 0o755); err != nil {
		t.Fatal(err)
	}
	filePath := filepath.Join(blogDir, "post.md")
	if err := os.WriteFile(filePath, []byte(markdown), 0o644); err != nil {
		t.Fatal(err)
	}

	cf := content.ContentFile{
		FilePath:       filePath,
		RelPath:        "blog/post.md",
		Kind:           engine.KindPage,
		CollectionName: "blog",
	}
	page, _, err := buildPage(cf, contentDir, nil, nil, 0, "mtime", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	return page
}

func TestBuildPage_DateExplicit_FromFrontmatter(t *testing.T) {
	page := buildTestPage(t, "---\ntitle: Post\ndate: 2024-01-01\n---\n\nBody.\n")
	if !page.DateExplicit {
		t.Error("a frontmatter date must set DateExplicit")
	}
	if page.Date.IsZero() {
		t.Error("frontmatter date should populate Date")
	}
}

func TestBuildPage_DateExplicit_MtimeFallback(t *testing.T) {
	page := buildTestPage(t, "---\ntitle: Post\n---\n\nBody.\n")
	if page.DateExplicit {
		t.Error("an mtime-inferred date must not set DateExplicit")
	}
	if page.Date.IsZero() {
		t.Error("mtime fallback should still populate Date")
	}
}

func TestBuildPage_OGCard(t *testing.T) {
	page := buildTestPage(t, `---
title: Post
og_card:
  bg_color: "#0d1117"
  accent_color: "#58a6ff"
  hide_watermark: true
---

Body.
`)
	oc, ok := page.Params["og_card"].(*engine.OGCard)
	if !ok {
		t.Fatalf("og_card should land in Params as *engine.OGCard, got %T", page.Params["og_card"])
	}
	if oc.BgColor != "#0d1117" {
		t.Errorf("bg_color = %q, want %q", oc.BgColor, "#0d1117")
	}
	if oc.AccentColor != "#58a6ff" {
		t.Errorf("accent_color = %q, want %q", oc.AccentColor, "#58a6ff")
	}
	if oc.AccentColor2 != "" || oc.TextColor != "" {
		t.Error("omitted color fields should stay empty")
	}
	if !oc.HideWatermark {
		t.Error("hide_watermark should parse true")
	}
	if oc.HideLogo {
		t.Error("omitted hide_logo should stay false")
	}
}

func TestBuildPage_OGCard_AbsentLeavesParamUnset(t *testing.T) {
	page := buildTestPage(t, "---\ntitle: Post\n---\n\nBody.\n")
	if _, ok := page.Params["og_card"]; ok {
		t.Error("pages without an og_card block must not set the param")
	}
}
