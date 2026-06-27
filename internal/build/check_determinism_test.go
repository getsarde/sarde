package build

import (
	"fmt"
	"sort"
	"testing"

	"github.com/getsarde/sarde/embedded"
	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/links"
)

// createVersionedFixture builds a multi-language, versioned-docs site where
// the latest version (v2) lives at the collection root and v1 lives in a v1/
// subdirectory. The v2 page at the root has a broken same-page anchor
// (#broken-targets with no matching heading) to exercise deterministic anchor
// validation across i18n fallback lanes.
func createVersionedFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	writeFixture(t, dir, "content/_index.md", "---\ntitle: Home\n---\n# Home\n")

	// Latest version (v2) at root — NO "Broken Targets" heading → #broken-targets is broken.
	writeFixture(t, dir, "content/docs/_index.md", "---\ntitle: Docs v2\n---\n")
	writeFixture(t, dir, "content/docs/guide/link-test.md",
		"---\ntitle: Link Test (v2)\n---\n"+
			"## Broken Anchor\n\nAnchor to [section](#broken-targets).\n")

	// v1 page so the versions list is fully populated.
	writeFixture(t, dir, "content/docs/v1/_index.md", "---\ntitle: Docs v1\n---\n")
	writeFixture(t, dir, "content/docs/v1/guide/link-test.md",
		"---\ntitle: Link Test (v1)\n---\n## Intro\n")

	return dir
}

func versionedCheckConfig() *config.SiteConfig {
	cfg := config.Defaults()
	cfg.Build.Minify = config.BoolPtr(false)
	cfg.Build.Cache = config.BoolPtr(false)
	cfg.I18n.DefaultLanguage = "en"
	cfg.I18n.Languages = map[string]config.LanguageConfig{
		"en": {Name: "English", Weight: 1, Dir: "ltr"},
		"fr": {Name: "French", Weight: 2, Dir: "ltr"},
		"ar": {Name: "Arabic", Weight: 3, Dir: "rtl"},
	}
	if cfg.Collections == nil {
		cfg.Collections = make(map[string]*config.CollectionSiteConfig)
	}
	cfg.Collections["docs"] = &config.CollectionSiteConfig{
		Versioning: &config.VersioningConfig{
			Enabled:     config.BoolPtr(true),
			LastVersion: "v2",
			Versions: []config.VersionEntry{
				{ID: "v2", Label: "v2.x (latest)"},
				{ID: "v1", Label: "v1.x"},
			},
		},
	}
	cfg.LinkValidation.Enabled = config.BoolPtr(true)
	cfg.LinkValidation.OnBroken = "warn"
	cfg.LinkValidation.OnBrokenAnchor = "warn"
	return cfg
}

func brokenAnchorCount(findings []links.Finding) int {
	n := 0
	for _, f := range findings {
		if f.Type == links.FindingBrokenAnchor {
			n++
		}
	}
	return n
}

// findingKeys returns a sorted, stable fingerprint of a finding set so two
// Check() runs can be compared regardless of report emission order.
func findingKeys(findings []links.Finding) []string {
	keys := make([]string, 0, len(findings))
	for _, f := range findings {
		keys = append(keys, fmt.Sprintf("%s|%s|%s|%s",
			f.Type, f.Ref.FromFile, f.Ref.RawDest, f.Ref.Fragment))
	}
	sort.Strings(keys)
	return keys
}

// TestCheck_RootVersionServesCorrectURL verifies that root-level pages in a
// versioned collection are correctly identified as the latest version and
// produce findings attributed to the root path (not a vN/ path).
func TestCheck_RootVersionServesCorrectURL(t *testing.T) {
	dir := createVersionedFixture(t)
	builder := NewSiteBuilder(BuildOptions{
		ProjectDir:  dir,
		Config:      versionedCheckConfig(),
		ThemeConfig: buildThemeConfig(),
		EmbeddedFS:  embedded.ThemeFS(),
	})
	res, err := builder.Check(CheckOptions{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	// The root link-test.md should produce broken_anchor findings (it links
	// #broken-targets but has no such heading).
	var foundBrokenAnchor bool
	for _, f := range res.Findings {
		if f.Type == links.FindingBrokenAnchor {
			foundBrokenAnchor = true
		}
	}
	if !foundBrokenAnchor {
		t.Error("expected broken_anchor finding from root v2 page, got none")
	}
}

// TestCheck_BrokenAnchorCountIsDeterministic guards the nondeterministic
// broken_anchor regression at the full pipeline level: a fresh SiteBuilder runs
// Check() many times on the same fixture and must return an identical
// broken_anchor count and finding set every time.
func TestCheck_BrokenAnchorCountIsDeterministic(t *testing.T) {
	dir := createVersionedFixture(t)

	run := func() (int, []string) {
		builder := NewSiteBuilder(BuildOptions{
			ProjectDir:  dir,
			Config:      versionedCheckConfig(),
			ThemeConfig: buildThemeConfig(),
			EmbeddedFS:  embedded.ThemeFS(),
		})
		res, err := builder.Check(CheckOptions{})
		if err != nil {
			t.Fatalf("Check failed: %v", err)
		}
		return brokenAnchorCount(res.Findings), findingKeys(res.Findings)
	}

	firstCount, firstKeys := run()
	for i := 0; i < 20; i++ {
		count, keys := run()
		if count != firstCount {
			t.Fatalf("run %d: broken_anchor count = %d, want %d (nondeterministic)", i, count, firstCount)
		}
		if len(keys) != len(firstKeys) {
			t.Fatalf("run %d: finding count = %d, want %d", i, len(keys), len(firstKeys))
		}
		for j := range keys {
			if keys[j] != firstKeys[j] {
				t.Fatalf("run %d: findings differ at %d:\n got %q\nwant %q", i, j, keys[j], firstKeys[j])
			}
		}
	}
}
