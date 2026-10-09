package build

import (
	"path/filepath"
	"testing"

	"github.com/getsarde/sarde/embedded"
	"github.com/getsarde/sarde/internal/config"
)

// TestBuild_LabsLandingCards: the labs collection root and each course page
// render the page title, description and body above the card grid. Cards
// carry an initials tile and the index page's sidebar badge. Counts are
// pluralized and a lab's step count includes the Overview, so it matches the
// "Step x of N" progress bar inside the lab; a single-page lab reads "1 step".
func TestBuild_LabsLandingCards(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "content/_index.md", "---\ntitle: Home\n---\n# Home\n")
	writeFixture(t, dir, "content/labs/_index.md",
		"---\ntitle: Labs\ndescription: Hands-on exercises.\n---\n\nPick a course below.\n")
	writeFixture(t, dir, "content/labs/web/_index.md",
		"---\ntitle: Web Fundamentals\ndescription: Web labs.\nsidebar:\n  badge:\n    text: Beginner\n    variant: tip\n---\n")
	writeFixture(t, dir, "content/labs/web/build/_index.md",
		"---\ntitle: Build a Webpage\ndescription: Two steps.\n---\n\nIntro.\n")
	writeFixture(t, dir, "content/labs/web/build/scaffold.md",
		"---\ntitle: Scaffold\nsidebar:\n  order: 1\n---\n\nStep one.\n")
	writeFixture(t, dir, "content/labs/web/build/style.md",
		"---\ntitle: Style\nsidebar:\n  order: 2\n---\n\nStep two.\n")
	writeFixture(t, dir, "content/labs/web/hello/_index.md",
		"---\ntitle: Hello World\ndescription: One page.\n---\n\nAll in one page.\n")
	writeFixture(t, dir, "content/labs/py/_index.md",
		"---\ntitle: Python Essentials\n---\n")
	writeFixture(t, dir, "content/labs/py/cli/_index.md",
		"---\ntitle: CLI Todo App\n---\n\nIntro.\n")
	writeFixture(t, dir, "content/labs/py/cli/setup.md",
		"---\ntitle: Setup\nsidebar:\n  order: 1\n---\n\nStep one.\n")

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

	// Collection root: title, description, body, then one card per course.
	root := "labs/index.html"
	assertFixtureFileContains(t, distDir, root, `<h1 class="sarde-page-title">Labs</h1>`)
	assertFixtureFileContains(t, distDir, root, `<p class="sarde-page-description">Hands-on exercises.</p>`)
	assertFixtureFileContains(t, distDir, root, "Pick a course below.")
	assertFixtureFileContains(t, distDir, root, `<span class="sarde-tab-tile" aria-hidden="true">WF</span>`)
	assertFixtureFileContains(t, distDir, root, `<span class="sarde-badge sarde-badge-tip">Beginner</span>`)
	assertFixtureFileContains(t, distDir, root, `<span class="sarde-doc-card__count">2 labs</span>`)
	assertFixtureFileContains(t, distDir, root, `<span class="sarde-doc-card__count">1 lab</span>`)
	assertFixtureFileNotContains(t, distDir, root, "1 labs")

	// Course page: title, then one card per lab with a step count that
	// includes the Overview.
	course := "labs/web/index.html"
	assertFixtureFileContains(t, distDir, course, `<h1 class="sarde-page-title">Web Fundamentals</h1>`)
	assertFixtureFileContains(t, distDir, course, `href="/labs/web/build/"`)
	assertFixtureFileContains(t, distDir, course, `href="/labs/web/hello/"`)
	assertFixtureFileContains(t, distDir, course, `<span class="sarde-doc-card__count">3 steps</span>`)
	assertFixtureFileContains(t, distDir, course, `<span class="sarde-doc-card__count">1 step</span>`)
	assertFixtureFileNotContains(t, distDir, course, "1 steps")
}
