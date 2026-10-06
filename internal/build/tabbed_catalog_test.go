package build

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/getsarde/sarde/internal/config"
)

// writeTabbedSite writes a "courses" collection with two tabs. rootBody is
// the body of courses/_index.md; empty keeps it frontmatter-only.
func writeTabbedSite(t *testing.T, rootBody string) string {
	t.Helper()
	dir := t.TempDir()
	writeFixture(t, dir, "content/_index.md", "---\ntitle: Home\n---\n# Home\n")
	writeFixture(t, dir, "content/courses/_index.md", "---\ntitle: Courses\ndescription: All courses.\n---\n"+rootBody)
	writeFixture(t, dir, "content/courses/web/_index.md", "---\ntitle: Web Basics\ndescription: HTML and CSS.\nsidebar:\n  order: 1\n  badge:\n    text: Beginner\n    variant: tip\n---\nWeb overview.\n")
	writeFixture(t, dir, "content/courses/web/html.md", "---\ntitle: HTML\n---\nHTML lesson.\n")
	writeFixture(t, dir, "content/courses/go/_index.md", "---\ntitle: Go Basics\ndescription: The Go language.\nsidebar:\n  order: 2\n---\nGo overview.\n")
	writeFixture(t, dir, "content/courses/go/types.md", "---\ntitle: Types\n---\nTypes lesson.\n")
	return dir
}

func TestBuild_TabbedRootWithoutBodyRedirects(t *testing.T) {
	dir := writeTabbedSite(t, "")
	builder := newIncrementalBuilder(dir, config.Defaults())
	if _, err := builder.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	html := readFixture(t, filepath.Join(dir, "dist"), "courses/index.html")
	if !strings.Contains(html, `http-equiv="refresh" content="0;url=/courses/web/"`) {
		t.Errorf("expected a redirect to the first tab, got:\n%s", html)
	}
}

func TestBuild_TabbedRootWithBodyIsCatalog(t *testing.T) {
	dir := writeTabbedSite(t, "Pick a course to start.\n")
	builder := newIncrementalBuilder(dir, config.Defaults())
	if _, err := builder.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	html := readFixture(t, filepath.Join(dir, "dist"), "courses/index.html")
	if strings.Contains(html, `http-equiv="refresh"`) {
		t.Fatal("a root with a body must render as a page, not redirect")
	}
	for _, want := range []string{
		"Pick a course to start.",
		`class="sarde-card-grid sarde-catalog-grid"`,
		`<span class="sarde-badge sarde-badge-tip">Beginner</span>`,
		"The Go language.",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("catalog missing %q", want)
		}
	}
	web := strings.Index(html, `href="/courses/web/" class="sarde-doc-card sarde-catalog-card"`)
	goTab := strings.Index(html, `href="/courses/go/" class="sarde-doc-card sarde-catalog-card"`)
	if web == -1 || goTab == -1 || web > goTab {
		t.Errorf("want one card per tab in tab order (web before go), got positions %d, %d", web, goTab)
	}
	for _, unwanted := range []string{`id="sarde-sidebar"`, "sarde-tab-switcher"} {
		if strings.Contains(html, unwanted) {
			t.Errorf("catalog should have no sidebar or tab switcher, found %q", unwanted)
		}
	}
}

// In sarde dev, editing a course description must refresh its catalog card,
// which reads tab data copied when the collection registry builds. An
// _index.md edit falls back to a full rebuild, which rebuilds the tabs.
func TestContentRebuild_TabDescriptionRefreshesCatalog(t *testing.T) {
	dir := writeTabbedSite(t, "Pick a course to start.\n")
	builder := newIncrementalBuilder(dir, config.Defaults())
	if _, err := builder.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	writeFixture(t, dir, "content/courses/go/_index.md", "---\ntitle: Go Basics\ndescription: Learn Go from scratch.\nsidebar:\n  order: 2\n---\nGo overview.\n")
	if _, err := builder.ContentRebuild([]string{filepath.Join(dir, "content", "courses", "go", "_index.md")}); err != nil {
		t.Fatalf("ContentRebuild failed: %v", err)
	}

	html := readFixture(t, filepath.Join(dir, "dist"), "courses/index.html")
	if !strings.Contains(html, "Learn Go from scratch.") {
		t.Error("catalog card still shows the old course description after a dev rebuild")
	}
}
