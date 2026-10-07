package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/getsarde/sarde/embedded"
	"github.com/getsarde/sarde/internal/config"
)

// TestBuild_LabsPageTags: a lab's index page shows the tags from its
// frontmatter and links them to the tag pages; a step without tags shows no
// tag list. The course template used to cover this; it now lives in the
// templates repository, so the engine behavior is pinned here.
func TestBuild_LabsPageTags(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "content/_index.md", "---\ntitle: Home\n---\n# Home\n")
	writeFixture(t, dir, "content/labs/_index.md", "---\ntitle: Labs\n---\n")
	writeFixture(t, dir, "content/labs/web/_index.md", "---\ntitle: Web Labs\n---\n")
	writeFixture(t, dir, "content/labs/web/build/_index.md",
		"---\ntitle: Build a Webpage\ntags: [html, css]\n---\n\nIntro.\n")
	writeFixture(t, dir, "content/labs/web/build/scaffold.md",
		"---\ntitle: Scaffold\nsidebar:\n  order: 1\n---\n\nStep one.\n")

	cfg := config.Defaults()
	cfg.Build.Minify = config.BoolPtr(false)
	cfg.Build.Cache = config.BoolPtr(false)
	cfg.Build.LastUpdated = "false"

	builder := NewSiteBuilder(BuildOptions{
		ProjectDir:  dir,
		Config:      cfg,
		ThemeConfig: buildThemeConfig(),
		EmbeddedFS:  embedded.ThemeFS(),
	})
	if _, err := builder.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	distDir := filepath.Join(dir, "dist")
	assertFixtureFileContains(t, distDir, "labs/web/build/index.html", "sarde-page-tags")
	assertFixtureFileContains(t, distDir, "labs/web/build/index.html", `href="/tags/html/"`)

	step, err := os.ReadFile(filepath.Join(distDir, "labs", "web", "build", "scaffold", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(step), "sarde-page-tags") {
		t.Error("lab step without tags renders a tag list")
	}
}
