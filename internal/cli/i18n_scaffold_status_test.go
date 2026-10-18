package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const i18nTestConfig = "site:\n  title: School\ni18n:\n  default_language: en\n  languages:\n    en: {weight: 1}\n    fr: {weight: 2}\n    de: {weight: 3}\n"

func writeSiteFile(t *testing.T, dir, rel, body string) {
	t.Helper()
	path := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// runI18nCLI runs an i18n subcommand and returns its stdout and stderr.
func runI18nCLI(t *testing.T, args ...string) (stdout, stderr string) {
	t.Helper()
	var errBuf bytes.Buffer
	rootCmd.SetErr(&errBuf)
	defer rootCmd.SetErr(nil)
	out, err := captureStdout(t, func() error {
		rootCmd.SetArgs(append([]string{"i18n"}, args...))
		return rootCmd.Execute()
	})
	if err != nil {
		t.Fatalf("i18n %v: %v (%s)", args, err, out)
	}
	return out, errBuf.String()
}

type i18nStatusOut struct {
	OK     bool                                 `json:"ok"`
	Status map[string]map[string]map[string]int `json:"status"`
}

// Status pairs pages by their path without the language folder, as the
// build does: a French page with no English source is not a translation.
func TestI18nStatus_PairsPagesWithTheirSource(t *testing.T) {
	dir := t.TempDir()
	writeSiteFile(t, dir, "sarde.yaml", i18nTestConfig)
	writeSiteFile(t, dir, "content/docs/_index.md", "---\ntitle: Docs\n---\n")
	writeSiteFile(t, dir, "content/docs/setup.md", "# Setup\n")
	writeSiteFile(t, dir, "content/docs/deploy.md", "# Deploy\n")
	writeSiteFile(t, dir, "content/fr/docs/setup.md", "# Installation\n")
	writeSiteFile(t, dir, "content/fr/docs/only-in-french.md", "# Seulement\n")

	out, _ := runI18nCLI(t, "status", dir)
	var got i18nStatusOut
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("decode %q: %v", out, err)
	}
	docs := got.Status["docs"]
	if docs["en"]["total"] != 3 || docs["fr"]["translated"] != 1 || docs["fr"]["total"] != 3 {
		t.Errorf("docs status = %v, want 3 pages with 1 French translation", docs)
	}
	if docs["de"]["translated"] != 0 {
		t.Errorf("de = %v, want nothing translated", docs["de"])
	}
	if _, ok := got.Status["fr"]; ok {
		t.Errorf("the fr folder is listed as a collection: %v", got.Status)
	}
}

// On a root layout the other language folders sit inside content/; they
// are not collections, so scaffolding de writes no content/de/fr/.
func TestI18nScaffold_SkipsLanguageFolders(t *testing.T) {
	dir := t.TempDir()
	writeSiteFile(t, dir, "sarde.yaml", i18nTestConfig)
	writeSiteFile(t, dir, "content/docs/setup.md", "# Setup\n")
	writeSiteFile(t, dir, "content/blog/news.md", "# News\n")
	writeSiteFile(t, dir, "content/fr/docs/setup.md", "# Installation\n")
	writeSiteFile(t, dir, "content/_drafts/idea.md", "# Idea\n")

	out, _ := runI18nCLI(t, "scaffold", dir, "de")
	if !strings.Contains(out, `"collectionsScaffolded":2`) {
		t.Errorf("output = %s, want 2 collections scaffolded", out)
	}
	for _, rel := range []string{"content/de/docs/_index.md", "content/de/blog/_index.md"} {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(rel))); err != nil {
			t.Errorf("%s not written: %v", rel, err)
		}
	}
	for _, rel := range []string{"content/de/fr", "content/de/_drafts"} {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(rel))); err == nil {
			t.Errorf("%s was scaffolded", rel)
		}
	}
}

// content.dir moves the content root; both commands follow it.
func TestI18nScaffold_HonorsContentDir(t *testing.T) {
	dir := t.TempDir()
	writeSiteFile(t, dir, "sarde.yaml", i18nTestConfig+"content:\n  dir: pages\n")
	writeSiteFile(t, dir, "pages/docs/setup.md", "# Setup\n")

	runI18nCLI(t, "scaffold", dir, "fr")
	if _, err := os.Stat(filepath.Join(dir, "pages", "fr", "docs", "_index.md")); err != nil {
		t.Errorf("scaffold ignored content.dir: %v", err)
	}
	out, _ := runI18nCLI(t, "status", dir)
	if !strings.Contains(out, `"docs"`) {
		t.Errorf("status ignored content.dir: %s", out)
	}
}

// A content/<default>/ folder is not where the default language lives; the
// commands say so and read it as the build does, as a collection.
func TestI18nStatus_WarnsAboutADefaultLanguageFolder(t *testing.T) {
	dir := t.TempDir()
	writeSiteFile(t, dir, "sarde.yaml", i18nTestConfig)
	writeSiteFile(t, dir, "content/en/docs/setup.md", "# Setup\n")

	out, stderr := runI18nCLI(t, "status", dir)
	if !strings.Contains(stderr, "content/en/ is read as a collection") {
		t.Errorf("stderr = %q, want the default-language folder warning", stderr)
	}
	var got i18nStatusOut
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("decode %q: %v", out, err)
	}
	if _, ok := got.Status["en"]; !ok {
		t.Errorf("status = %v, want en listed as a collection", got.Status)
	}
}
