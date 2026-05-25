package build

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/frostybee/sarde/embedded"
	"github.com/frostybee/sarde/internal/config"
	"github.com/frostybee/sarde/internal/engine"
)

func TestBuild_ParallelMatchesSerialOutput(t *testing.T) {
	dir := createParallelFixtureSite(t)

	serialResult := runParallelFixtureBuild(t, dir, "dist-serial", false)
	parallelResult := runParallelFixtureBuild(t, dir, "dist-parallel", true)

	serialFiles := readOutputTree(t, filepath.Join(dir, "dist-serial"))
	parallelFiles := readOutputTree(t, filepath.Join(dir, "dist-parallel"))
	if len(serialFiles) != len(parallelFiles) {
		t.Fatalf("file count mismatch: serial=%d parallel=%d", len(serialFiles), len(parallelFiles))
	}
	for rel, serialData := range serialFiles {
		parallelData, ok := parallelFiles[rel]
		if !ok {
			t.Fatalf("parallel output missing %s", rel)
		}
		if !bytes.Equal(serialData, parallelData) {
			t.Fatalf("output mismatch for %s", rel)
		}
	}

	if serialResult.PageCount != parallelResult.PageCount {
		t.Fatalf("PageCount mismatch: serial=%d parallel=%d", serialResult.PageCount, parallelResult.PageCount)
	}
	for _, phase := range []string{
		"Discovering content",
		"Parsing content",
		"Assembling site",
		"Asset preparation",
		"Rendering markdown",
		"Template setup",
		"Rendering templates",
		"Rendering synthetic pages",
		"Rendering 404 pages",
		"Minifying HTML",
		"Writing output",
		"Writing assets",
		"Running plugins",
		"Pruning output",
	} {
		if !hasPhase(parallelResult.PhaseTimings, phase) {
			t.Fatalf("parallel build missing phase timing %q", phase)
		}
	}

	enPage := string(parallelFiles["blog/post-01/index.html"])
	frPage := string(parallelFiles["fr/blog/post-01/index.html"])
	if !strings.Contains(enPage, "Search") || strings.Contains(enPage, "Rechercher") {
		t.Fatalf("English page used wrong translation: %s", enPage)
	}
	if !strings.Contains(frPage, "Rechercher") || strings.Contains(frPage, ">Search<") {
		t.Fatalf("French page used wrong translation: %s", frPage)
	}
}

func createParallelFixtureSite(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	writeFixture(t, dir, "i18n/en.yaml", "nav:\n  search: Search\nannouncements:\n  maint: Maintenance notice\n  dismiss: Dismiss\n")
	writeFixture(t, dir, "i18n/fr.yaml", "nav:\n  search: Rechercher\nannouncements:\n  maint: Avis de maintenance\n  dismiss: Fermer\n")

	writeFixture(t, dir, "assets/css/site.css", "body { color: #123456; }\n")
	writeFixture(t, dir, "assets/js/site.js", "console.log('fixture');\n")

	writeFixture(t, dir, "layouts/_default/baseof.html", `<!doctype html><html lang="{{ .Lang }}"><head><title>{{ t "nav.search" }}</title><link rel="stylesheet" href="{{ fingerprint "css/site.css" }}"></head><body>{{ partial "banner.html" . }}{{ component "Search" . }}{{ announcementBanner }}{{ block "content" . }}{{ end }}</body></html>`)
	writeFixture(t, dir, "layouts/partials/banner.html", `<aside>{{ t "nav.search" }}:{{ .Lang }}</aside>`)
	writeFixture(t, dir, "layouts/_default/home.html", `{{ define "content" }}<main>{{ .Page.Content }}</main>{{ end }}`)
	writeFixture(t, dir, "layouts/_default/single.html", `{{ define "content" }}<article>{{ .Page.Content }}</article>{{ end }}`)
	writeFixture(t, dir, "layouts/_default/list.html", `{{ define "content" }}<section>{{ .Page.Title }}{{ range .Paginator.CurrentPages }}<a href="{{ .RelPermalink }}">{{ .Title }}</a>{{ end }}</section>{{ end }}`)
	writeFixture(t, dir, "layouts/_blog/list.html", `{{ define "content" }}<section>{{ t "nav.search" }}{{ range .Paginator.CurrentPages }}<a href="{{ .RelPermalink }}">{{ .Title }}</a>{{ end }}{{ if .Paginator.HasNext }}<a href="{{ .Paginator.NextURL }}">next</a>{{ end }}</section>{{ end }}`)
	writeFixture(t, dir, "layouts/_taxonomy/list.html", `{{ define "content" }}<section>{{ .Page.Title }}{{ range .TermEntries }}<a href="{{ .Permalink }}">{{ .Label }}</a>{{ end }}</section>{{ end }}`)
	writeFixture(t, dir, "layouts/_taxonomy/term.html", `{{ define "content" }}<section>{{ .Page.Title }}{{ range .Paginator.CurrentPages }}<a href="{{ .RelPermalink }}">{{ .Title }}</a>{{ end }}</section>{{ end }}`)

	writeFixture(t, dir, "content/_index.md", "---\ntitle: Home\ndate: 2025-01-01T00:00:00Z\n---\n# Home\n")
	writeFixture(t, dir, "content/about.md", "---\ntitle: About\ndate: 2025-01-01T00:00:00Z\naliases: [\"/about-us/\"]\n---\n# About\n")
	writeFixture(t, dir, "content/blog/_index.md", "---\ntitle: Blog\ndate: 2025-01-01T00:00:00Z\n---\n")
	writeFixture(t, dir, "content/fr/blog/_index.md", "---\ntitle: Blog FR\ndate: 2025-01-01T00:00:00Z\n---\n")
	for i := 1; i <= 6; i++ {
		writeFixture(t, dir, filepath.ToSlash(filepath.Join("content", "blog", "post-"+twoDigit(i)+".md")),
			"---\ntitle: Post "+twoDigit(i)+"\ndate: 2025-01-"+twoDigit(i)+"T00:00:00Z\ntags: [go]\n---\n# Post "+twoDigit(i)+"\n")
		writeFixture(t, dir, filepath.ToSlash(filepath.Join("content", "fr", "blog", "post-"+twoDigit(i)+".md")),
			"---\ntitle: Article "+twoDigit(i)+"\ndate: 2025-01-"+twoDigit(i)+"T00:00:00Z\ntags: [go]\n---\n# Article "+twoDigit(i)+"\n")
	}
	writeFixture(t, dir, "content/blog/bundled/index.md", "---\ntitle: Bundled\ndate: 2025-01-20T00:00:00Z\ntags: [assets]\n---\n# Bundled\n")
	writeFixture(t, dir, "content/blog/bundled/asset.txt", "bundle asset\n")

	return dir
}

func runParallelFixtureBuild(t *testing.T, dir, output string, parallel bool) *engine.BuildResult {
	t.Helper()
	cfg := config.Defaults()
	cfg.Build.Output = output
	cfg.Build.Parallel = boolForTest(parallel)
	cfg.Build.Minify = boolForTest(false)
	cfg.Build.Cache = boolForTest(false)
	cfg.Build.LastUpdated = "false"
	cfg.Head.CustomCSS = []string{"css/site.css"}
	cfg.Head.CustomJS = []string{"js/site.js"}
	cfg.I18n.DefaultLanguage = "en"
	cfg.I18n.Languages = map[string]config.LanguageConfig{
		"en": {Name: "English", Weight: 1, Dir: "ltr"},
		"fr": {Name: "French", Weight: 2, Dir: "ltr"},
	}
	if cfg.Collections == nil {
		cfg.Collections = make(map[string]*config.CollectionSiteConfig)
	}
	cfg.Collections["blog"] = &config.CollectionSiteConfig{Paginate: 3}
	cfg.Plugins.Enabled = []string{"announcements"}
	cfg.Plugins.Config = map[string]map[string]any{
		"announcements": {
			"items": []any{
				map[string]any{"id": "maint", "message": "announcements.maint", "active": true},
			},
		},
	}

	builder := NewSiteBuilder(BuildOptions{
		ProjectDir:  dir,
		Config:      cfg,
		ThemeConfig: buildThemeConfig(),
		EmbeddedFS:  embedded.ThemeFS(),
	})
	result, err := builder.Build()
	if err != nil {
		t.Fatalf("Build(%s, parallel=%v) failed: %v", output, parallel, err)
	}
	return result
}

func boolForTest(v bool) *bool { return &v }

func twoDigit(i int) string {
	return fmt.Sprintf("%02d", i)
}

func readOutputTree(t *testing.T, root string) map[string][]byte {
	t.Helper()
	files := make(map[string][]byte)
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(rel)] = data
		return nil
	})
	if err != nil {
		t.Fatalf("walking %s: %v", root, err)
	}
	return files
}

func hasPhase(timings []engine.PhaseTiming, phase string) bool {
	for _, timing := range timings {
		if timing.Phase == phase {
			return true
		}
	}
	return false
}
