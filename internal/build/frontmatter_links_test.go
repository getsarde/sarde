package build

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/getsarde/sarde/embedded"
	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/engine"
)

// frontmatterLinkSite is the fixture site plus pages whose frontmatter
// names images and prev / next targets, good and bad.
func frontmatterLinkSite(t *testing.T) string {
	t.Helper()
	dir := createFixtureSite(t)
	writeFixture(t, dir, "public/images/cover.png", "png")
	writeFixture(t, dir, "content/blog/bundle/index.md", "---\ntitle: Bundle\nimage: photo.png\n---\nBody.\n")
	writeFixture(t, dir, "content/blog/bundle/photo.png", "png")
	writeFixture(t, dir, "content/blog/good.md", "---\ntitle: Good\nimage: /images/cover.png\nprev:\n  link: /docs/getting-started/\nnext: hello-world\n---\nBody.\n")
	writeFixture(t, dir, "content/blog/remote.md", "---\ntitle: Remote\nimage: https://cdn.example.com/cover.png\n---\nBody.\n")
	writeFixture(t, dir, "content/blog/bad-image.md", "---\ntitle: Bad image\nimage: /images/missing.png\n---\nBody.\n")
	writeFixture(t, dir, "content/blog/bad-prev.md", "---\ntitle: Bad prev\nprev:\n  link: ../docs/advanced.md\n---\nBody.\n")
	writeFixture(t, dir, "content/blog/bad-anchor.md", "---\ntitle: Bad anchor\nnext:\n  link: /docs/advanced/#no-such-heading\n---\nBody.\n")
	writeFixture(t, dir, "content/blog/bad-slug.md", "---\ntitle: Bad slug\nnext: no-such-post\n---\nBody.\n")
	return dir
}

func buildFrontmatterLinkSite(t *testing.T, dir string) []engine.ValidationWarning {
	t.Helper()
	cfg := config.Defaults()
	// Every finding as a warning, so one build reports them all.
	cfg.LinkValidation.OnBroken = "warn"
	cfg.LinkValidation.OnBrokenAnchor = "warn"
	builder := NewSiteBuilder(BuildOptions{
		ProjectDir:  dir,
		Config:      cfg,
		ThemeConfig: buildThemeConfig(),
		EmbeddedFS:  embedded.ThemeFS(),
	})
	result, err := builder.Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	var out []engine.ValidationWarning
	for _, w := range result.Warnings {
		if w.Field == "link" && strings.Contains(w.Message, "(frontmatter ") {
			out = append(out, w)
		}
	}
	return out
}

func TestLinkCheck_FrontmatterFields(t *testing.T) {
	warnings := buildFrontmatterLinkSite(t, frontmatterLinkSite(t))
	byFile := map[string]string{}
	for _, w := range warnings {
		byFile[filepath.Base(w.File)] += w.Message + "\n"
	}

	want := map[string]string{
		"bad-image.md":  "/images/missing.png (frontmatter image)",
		"bad-prev.md":   "broken target: ../docs/advanced.md (frontmatter prev.link)",
		"bad-anchor.md": "broken anchor: /docs/advanced/#no-such-heading (frontmatter next.link)",
		"bad-slug.md":   "broken target: no-such-post (frontmatter next)",
	}
	for file, msg := range want {
		if !strings.Contains(byFile[file], msg) {
			t.Errorf("%s: warnings %q, want one containing %q", file, byFile[file], msg)
		}
	}
	for _, file := range []string{"good.md", "remote.md", "index.md"} {
		if byFile[file] != "" {
			t.Errorf("%s: unexpected frontmatter warnings %q", file, byFile[file])
		}
	}
}

// A second build reuses the page cache for unchanged bodies; the frontmatter
// findings must not depend on it.
func TestLinkCheck_FrontmatterFieldsOnAWarmBuild(t *testing.T) {
	dir := frontmatterLinkSite(t)
	first := buildFrontmatterLinkSite(t, dir)
	second := buildFrontmatterLinkSite(t, dir)
	if len(first) == 0 || len(second) != len(first) {
		t.Errorf("warm build reported %d frontmatter findings, cold build %d", len(second), len(first))
	}
}

// A dev rebuild of one page checks its frontmatter links too.
func TestContentRebuild_FrontmatterLinkWarning(t *testing.T) {
	dir := createIncrementalSite(t)
	builder := newIncrementalBuilder(dir, config.Defaults())
	if _, err := builder.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	postPath := filepath.Join(dir, "content", "blog", "one.md")
	writeFixture(t, dir, "content/blog/one.md", "---\ntitle: One\ndate: 2025-01-02T00:00:00Z\nimage: /images/missing.png\n---\n# One\nBody.\n")
	result, err := builder.ContentRebuild([]string{postPath})
	if err != nil {
		t.Fatalf("ContentRebuild failed: %v", err)
	}
	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w.Message, "/images/missing.png (frontmatter image)") {
			found = true
		}
	}
	if !found {
		t.Errorf("rebuild warnings %v, want the missing frontmatter image", result.Warnings)
	}
}

func TestSitePath(t *testing.T) {
	page := &engine.Page{PageIdentity: engine.PageIdentity{RelPermalink: "/blog/post/"}}
	for raw, want := range map[string]string{
		"/images/a.png":  "/images/a.png",
		"cover.png":      "/blog/post/cover.png",
		"../other/":      "/blog/other/",
		"../../#top":     "/",
		"/docs/x/?q=1#y": "/docs/x/",
	} {
		if got, _ := sitePath(page, raw); got != want {
			t.Errorf("sitePath(%q) = %q, want %q", raw, got, want)
		}
	}
}
