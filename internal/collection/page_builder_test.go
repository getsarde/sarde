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

// buildNamedPage is buildTestPage with a chosen file name and collection
// config.
func buildNamedPage(t *testing.T, name, markdown string, collCfg *engine.CollectionConfig) *engine.Page {
	t.Helper()
	contentDir := t.TempDir()
	blogDir := filepath.Join(contentDir, "blog")
	if err := os.MkdirAll(blogDir, 0o755); err != nil {
		t.Fatal(err)
	}
	filePath := filepath.Join(blogDir, name)
	if err := os.WriteFile(filePath, []byte(markdown), 0o644); err != nil {
		t.Fatal(err)
	}
	cf := content.ContentFile{
		FilePath:       filePath,
		RelPath:        "blog/" + name,
		LangRelPath:    "blog/" + name,
		Kind:           engine.KindPage,
		CollectionName: "blog",
	}
	page, _, err := buildPage(cf, contentDir, collCfg, nil, 0, "mtime", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	return page
}

// A YYYY-MM-DD- file name dates the page (the mtime prefill used to hide
// it), and its URL and slug agree on the name without the date.
func TestBuildPage_DatePrefixedName(t *testing.T) {
	page := buildNamedPage(t, "2024-03-15-hello-world.md", "---\ntitle: Hello\n---\n\nBody.\n", nil)
	if page.Date.Format("2006-01-02") != "2024-03-15" || !page.DateExplicit {
		t.Errorf("Date = %v (explicit %v), want 2024-03-15 from the name", page.Date, page.DateExplicit)
	}
	if page.Slug != "hello-world" || page.RelPermalink != "/blog/hello-world/" {
		t.Errorf("Slug = %q, RelPermalink = %q, want hello-world and /blog/hello-world/", page.Slug, page.RelPermalink)
	}
}

// Two posts named in non-Latin script get distinct :slug URLs instead of
// both collapsing to /blog/2024/.
func TestBuildPage_PatternPermalinkNonLatinSlug(t *testing.T) {
	cfg := &engine.CollectionConfig{Permalink: "/blog/:year/:slug/"}
	md := "---\ntitle: Post\ndate: 2024-05-01\n---\n\nBody.\n"
	a := buildNamedPage(t, "日本語.md", md, cfg)
	b := buildNamedPage(t, "Привет.md", md, cfg)
	if a.RelPermalink == "/blog/2024/" || b.RelPermalink == "/blog/2024/" || a.RelPermalink == b.RelPermalink {
		t.Errorf("RelPermalinks = %q, %q, want two distinct /blog/2024/<hash>/ URLs", a.RelPermalink, b.RelPermalink)
	}
}

// A page without a frontmatter title takes it from its first H1 or its file
// name, and that inferred title satisfies the title check.
func TestBuildPage_NoFrontmatterTitle_NoTitleWarning(t *testing.T) {
	cases := []struct {
		name, markdown, wantTitle string
	}{
		{"h1", "---\ndescription: x\n---\n# From Heading\n\nBody.\n", "From Heading"},
		{"filename", "Body only.\n", "Post"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			contentDir := t.TempDir()
			blogDir := filepath.Join(contentDir, "blog")
			if err := os.MkdirAll(blogDir, 0o755); err != nil {
				t.Fatal(err)
			}
			filePath := filepath.Join(blogDir, "post.md")
			if err := os.WriteFile(filePath, []byte(tc.markdown), 0o644); err != nil {
				t.Fatal(err)
			}
			cf := content.ContentFile{
				FilePath:       filePath,
				RelPath:        "blog/post.md",
				Kind:           engine.KindPage,
				CollectionName: "blog",
			}
			page, warnings, err := buildPage(cf, contentDir, nil, nil, 0, "mtime", nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			if page.Title != tc.wantTitle {
				t.Errorf("Title = %q, want %q", page.Title, tc.wantTitle)
			}
			for _, w := range warnings {
				if w.Field == "title" {
					t.Errorf("unexpected title warning: %+v", w)
				}
			}
		})
	}
}
