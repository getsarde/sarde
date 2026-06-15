package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/frostybee/sarde/embedded"
	"github.com/frostybee/sarde/internal/config"
)

func newIncrementalBuilder(projectDir string, cfg *config.SiteConfig) *SiteBuilder {
	return NewSiteBuilder(BuildOptions{
		ProjectDir:  projectDir,
		Config:      cfg,
		ThemeConfig: buildThemeConfig(),
		EmbeddedFS:  embedded.ThemeFS(),
		DevMode:     true,
	})
}

func createIncrementalSite(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeFixture(t, dir, "content/_index.md", "---\ntitle: Home\n---\n# Home\n")
	writeFixture(t, dir, "content/about.md", "---\ntitle: About\n---\nAbout page.\n")
	writeFixture(t, dir, "content/blog/_index.md", "---\ntitle: Blog\n---\n")
	writeFixture(t, dir, "content/blog/one.md", "---\ntitle: One\ndate: 2025-01-02T00:00:00Z\n---\n# One\nOriginal body with [about](/about/).\n")
	writeFixture(t, dir, "content/blog/two.md", "---\ntitle: Two\ndate: 2025-01-01T00:00:00Z\n---\n# Two\nSecond body.\n")
	return dir
}

func TestContentRebuild_BodyOnlyUpdatesPageSearchAndValidation(t *testing.T) {
	dir := createIncrementalSite(t)
	cfg := config.Defaults()
	cfg.Plugins.Enabled = []string{"search", "link_validator"}
	cfg.LinkValidation.Enabled = config.BoolPtr(true)

	builder := newIncrementalBuilder(dir, cfg)
	if _, err := builder.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	postPath := filepath.Join(dir, "content", "blog", "one.md")
	writeFixture(t, dir, "content/blog/one.md", "---\ntitle: One\ndate: 2025-01-02T00:00:00Z\n---\n# One\nUpdated body with [missing](/missing/).\n")

	result, err := builder.ContentRebuild([]string{postPath})
	if err != nil {
		t.Fatalf("ContentRebuild failed: %v", err)
	}
	if result.PageCount >= 5 {
		t.Fatalf("expected incremental rebuild, got likely full rebuild page count %d", result.PageCount)
	}

	distDir := filepath.Join(dir, "dist")
	assertFixtureFileContains(t, distDir, "blog/one/index.html", "Updated body")
	assertFixtureFileContains(t, distDir, "search-index.en.json", "Updated body")

	entry, ok := builder.lastValidationData["/blog/one/"]
	if !ok {
		t.Fatal("expected validation data for changed page")
	}
	if len(entry.Links) != 1 || entry.Links[0].Href != "/missing/" {
		t.Fatalf("validation links = %+v, want only /missing/", entry.Links)
	}
	if len(result.Warnings) == 0 {
		t.Fatal("expected link-validator warning for /missing/")
	}
	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w.Message, "/missing/") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("warnings did not mention /missing/: %+v", result.Warnings)
	}
}

func TestContentRebuild_BodyOnlyRendersCollectionPagination(t *testing.T) {
	dir := createIncrementalSite(t)
	cfg := config.Defaults()
	cfg.Collections = map[string]*config.CollectionSiteConfig{
		"blog": {Paginate: 1},
	}

	builder := newIncrementalBuilder(dir, cfg)
	if _, err := builder.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	postPath := filepath.Join(dir, "content", "blog", "one.md")
	writeFixture(t, dir, "content/blog/one.md", "---\ntitle: One\ndate: 2025-01-02T00:00:00Z\n---\n# One\nPagination-safe body edit.\n")

	result, err := builder.ContentRebuild([]string{postPath})
	if err != nil {
		t.Fatalf("ContentRebuild failed: %v", err)
	}
	if result.PageCount < 3 {
		t.Fatalf("expected changed page plus collection list pagination, got %d rendered pages", result.PageCount)
	}
	assertFixtureFileExists(t, filepath.Join(dir, "dist"), "blog/page/2/index.html")
}

func TestContentRebuild_StructuralChangesFallBackToFullBuild(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		write      string
		removeFile bool
	}{
		{
			name:  "new file",
			path:  "content/blog/new.md",
			write: "---\ntitle: New\ndate: 2025-01-03T00:00:00Z\n---\nNew post.\n",
		},
		{
			name:       "deleted file",
			path:       "content/blog/one.md",
			removeFile: true,
		},
		{
			name:  "section index",
			path:  "content/blog/_index.md",
			write: "---\ntitle: Blog Updated\n---\n",
		},
		{
			name:  "slug",
			path:  "content/blog/one.md",
			write: "---\ntitle: One\nslug: renamed\ndate: 2025-01-02T00:00:00Z\n---\n# One\nBody.\n",
		},
		{
			name:  "alias",
			path:  "content/blog/one.md",
			write: "---\ntitle: One\ndate: 2025-01-02T00:00:00Z\naliases: [/old-one/]\n---\n# One\nBody.\n",
		},
		{
			name:  "taxonomy",
			path:  "content/blog/one.md",
			write: "---\ntitle: One\ndate: 2025-01-02T00:00:00Z\ntags: [go]\n---\n# One\nBody.\n",
		},
		{
			name:  "featured",
			path:  "content/blog/one.md",
			write: "---\ntitle: One\ndate: 2025-01-02T00:00:00Z\nfeatured: true\n---\n# One\nBody.\n",
		},
		{
			name:  "title",
			path:  "content/blog/one.md",
			write: "---\ntitle: One Updated\ndate: 2025-01-02T00:00:00Z\n---\n# One Updated\nBody.\n",
		},
		{
			name:  "order",
			path:  "content/blog/one.md",
			write: "---\ntitle: One\ndate: 2025-01-02T00:00:00Z\nsidebar:\n  order: 10\n---\n# One\nBody.\n",
		},
		{
			name:  "render flag",
			path:  "content/blog/one.md",
			write: "---\ntitle: One\ndate: 2025-01-02T00:00:00Z\nrender: false\n---\n# One\nBody.\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := createIncrementalSite(t)
			cfg := config.Defaults()
			builder := newIncrementalBuilder(dir, cfg)
			if _, err := builder.Build(); err != nil {
				t.Fatalf("Build failed: %v", err)
			}

			absPath := filepath.Join(dir, filepath.FromSlash(tt.path))
			if tt.removeFile {
				if err := os.Remove(absPath); err != nil {
					t.Fatalf("removing fixture: %v", err)
				}
			} else {
				writeFixture(t, dir, tt.path, tt.write)
			}

			result, err := builder.ContentRebuild([]string{absPath})
			if err != nil {
				t.Fatalf("ContentRebuild failed: %v", err)
			}
			if result.PageCount <= 2 {
				t.Fatalf("expected full rebuild fallback, got %d rendered pages", result.PageCount)
			}
		})
	}
}

func TestContentRebuild_FallbackPrunesStaleAliasOutput(t *testing.T) {
	dir := createIncrementalSite(t)
	writeFixture(t, dir, "content/blog/one.md", "---\ntitle: One\ndate: 2025-01-02T00:00:00Z\naliases: [/old-one/]\n---\n# One\nBody.\n")

	cfg := config.Defaults()
	builder := newIncrementalBuilder(dir, cfg)
	if _, err := builder.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	distDir := filepath.Join(dir, "dist")
	assertFixtureFileExists(t, distDir, "old-one/index.html")

	postPath := filepath.Join(dir, "content", "blog", "one.md")
	writeFixture(t, dir, "content/blog/one.md", "---\ntitle: One\ndate: 2025-01-02T00:00:00Z\naliases: [/new-one/]\n---\n# One\nBody.\n")
	if _, err := builder.ContentRebuild([]string{postPath}); err != nil {
		t.Fatalf("ContentRebuild failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(distDir, "old-one", "index.html")); !os.IsNotExist(err) {
		t.Fatalf("old alias output should be pruned, stat err = %v", err)
	}
	assertFixtureFileExists(t, distDir, "new-one/index.html")
}
