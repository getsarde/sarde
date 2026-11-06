package build

import (
	"path/filepath"
	"testing"

	"github.com/getsarde/sarde/embedded"
	"github.com/getsarde/sarde/internal/config"
)

// TestBuild_VersionedLatestLandingWithoutRootIndex: only the older version
// has a root _index.md. The latest version still gets a landing page at the
// collection root, and the older version's index stays on its own URL
// without lending the collection its title.
func TestBuild_VersionedLatestLandingWithoutRootIndex(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "content/_index.md", "---\ntitle: Home\n---\n")
	writeFixture(t, dir, "content/docs/intro.md", "---\ntitle: Introduction\n---\nLatest introduction.\n")
	writeFixture(t, dir, "content/docs/v1/_index.md", "---\ntitle: Version One Docs\n---\nVersion one overview.\n")
	writeFixture(t, dir, "content/docs/v1/intro.md", "---\ntitle: Introduction\n---\nVersion one introduction.\n")

	cfg := config.Defaults()
	cfg.Build.Minify = config.BoolPtr(false)
	cfg.Build.LastUpdated = "false"
	cfg.Collections = map[string]*config.CollectionSiteConfig{
		"docs": {Versioning: &config.VersioningConfig{
			Enabled: config.BoolPtr(true), LastVersion: "v2",
			Versions: []config.VersionEntry{{ID: "v2"}, {ID: "v1"}},
		}},
	}
	builder := NewSiteBuilder(BuildOptions{
		ProjectDir:  dir,
		Config:      cfg,
		ThemeConfig: buildThemeConfig(),
		EmbeddedFS:  embedded.ThemeFS(),
	})
	if _, err := builder.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	dist := filepath.Join(dir, "dist")

	assertFixtureFileExists(t, dist, "docs/index.html")
	assertFixtureFileNotContains(t, dist, "docs/index.html", "Version one overview.")
	assertFixtureFileContains(t, dist, "docs/v1/index.html", "Version one overview.")
	assertFixtureFileNotContains(t, dist, "docs/intro/index.html", "Version One Docs")
}
