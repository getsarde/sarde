package build

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/coderoo-dev/coderoo/embedded"
	"github.com/coderoo-dev/coderoo/internal/config"
)

// createLargeSite creates a test site with the given number of pages
// split across blog and docs collections.
func createLargeSite(b *testing.B, pageCount int) string {
	b.Helper()
	dir := b.TempDir()

	writeBenchFixture(dir, "content/_index.md", "---\ntitle: Home\n---\n# Welcome\n")
	writeBenchFixture(dir, "content/docs/_index.md", "---\ntitle: Docs\n---\n")
	writeBenchFixture(dir, "content/blog/_index.md", "---\ntitle: Blog\n---\n")

	for i := 0; i < pageCount; i++ {
		if i%2 == 0 {
			writeBenchFixture(dir, fmt.Sprintf("content/docs/page-%04d.md", i),
				fmt.Sprintf("---\ntitle: Doc Page %d\nweight: %d\n---\n# Doc Page %d\n\nThis is documentation page number %d with some **markdown** content.\n\n## Section\n\nMore content here with `code` and [links](/docs/).\n", i, i, i, i))
		} else {
			writeBenchFixture(dir, fmt.Sprintf("content/blog/post-%04d.md", i),
				fmt.Sprintf("---\ntitle: Blog Post %d\ndate: 2025-01-%02dT00:00:00Z\n---\n# Blog Post %d\n\nThis is blog post number %d with some **markdown** content.\n\n## Section\n\nMore content here with `code` and [links](/blog/).\n", i, (i%28)+1, i, i))
		}
	}

	return dir
}

func writeBenchFixture(base, rel, body string) {
	path := filepath.Join(base, filepath.FromSlash(rel))
	os.MkdirAll(filepath.Dir(path), 0o755)
	os.WriteFile(path, []byte(body), 0o644)
}

func benchmarkBuild(b *testing.B, pageCount int, parallel bool) {
	dir := createLargeSite(b, pageCount)
	cfg := config.Defaults()
	cfg.Build.Parallel = &parallel
	themeCfg := buildThemeConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		builder := NewSiteBuilder(BuildOptions{
			ProjectDir:  dir,
			Config:      cfg,
			ThemeConfig: themeCfg,
			EmbeddedFS:  embedded.ThemeFS(),
		})
		result, err := builder.Build()
		if err != nil {
			b.Fatal(err)
		}
		if result.PageCount == 0 {
			b.Fatal("no pages rendered")
		}
	}
}

func BenchmarkBuild_100Pages_Parallel(b *testing.B) {
	benchmarkBuild(b, 100, true)
}

func BenchmarkBuild_100Pages_Serial(b *testing.B) {
	benchmarkBuild(b, 100, false)
}

func BenchmarkBuild_500Pages_Parallel(b *testing.B) {
	benchmarkBuild(b, 500, true)
}

func BenchmarkBuild_500Pages_Serial(b *testing.B) {
	benchmarkBuild(b, 500, false)
}
