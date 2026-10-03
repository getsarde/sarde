package build

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/getsarde/sarde/embedded"
	"github.com/getsarde/sarde/internal/config"
)

// TestBuild_CourseTemplate builds the site that "sarde new site --template
// course" scaffolds, so an engine change that breaks the template fails here
// rather than for a user's first build.
func TestBuild_CourseTemplate(t *testing.T) {
	tmpl, ok := embedded.SiteTemplate("course")
	if !ok {
		t.Fatal("course template not embedded")
	}
	projDir := t.TempDir()
	if err := os.CopyFS(projDir, tmpl); err != nil {
		t.Fatalf("copying template: %v", err)
	}
	writeFixture(t, projDir, "public/images/hero-light.svg", string(embedded.ScaffoldHeroLight))
	writeFixture(t, projDir, "public/images/hero-dark.svg", string(embedded.ScaffoldHeroDark))

	cfg, err := config.Resolve(config.ResolveOptions{
		ConfigPath:   filepath.Join(projDir, "sarde.yaml"),
		KnownPlugins: KnownPluginNames(projDir),
		Strict:       true,
	})
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	builder := NewSiteBuilder(BuildOptions{
		ProjectDir:  projDir,
		Config:      cfg,
		ThemeConfig: buildThemeConfig(),
		EmbeddedFS:  embedded.ThemeFS(),
	})
	if _, err := builder.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	distDir := filepath.Join(projDir, "dist")
	for _, page := range []string{
		"index.html",
		"announcements/index.html",
		"courses/python-essentials/index.html",
		"courses/web-fundamentals/assignments/build-a-page/index.html",
		"labs/index.html",
		"labs/web-fundamentals/build-a-webpage/scaffold/index.html",
		"labs/web-fundamentals/hello-world/index.html",
		"labs/python-essentials/cli-todo-app/testing/index.html",
	} {
		assertFixtureFileExists(t, distDir, page)
	}
	assertFixtureFileContains(t, distDir, "courses/python-essentials/index.html", "sarde-tab-switcher")
}
