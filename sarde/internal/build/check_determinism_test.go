package build

import (
	"fmt"
	"sort"
	"testing"

	"github.com/frostybee/sarde/embedded"
	"github.com/frostybee/sarde/internal/config"
	"github.com/frostybee/sarde/internal/links"
)

// createCollisionFixture builds a multi-language, versioned-docs site that
// reproduces the permalink collision behind the nondeterministic broken_anchor
// bug. The unversioned docs/guide/link-test.md and the latest-version
// docs/v2/guide/link-test.md both resolve to /docs/guide/link-test/ (and their
// fr/ar fallbacks both to /<lang>/docs/guide/link-test/), but carry DIFFERENT
// headings: the unversioned page has "Broken Targets"; the v2 page does not.
//
// Each fallback page renders a same-page anchor #broken-targets. Whether that
// anchor validates depends on which colliding page populated the heading index
// — which, before the fix, flipped run-to-run because GenerateFallbacks
// iterated a Go map. After the fix the ordering (and the first-match index
// policy) is deterministic.
func createCollisionFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	writeFixture(t, dir, "content/_index.md", "---\ntitle: Home\n---\n# Home\n")
	writeFixture(t, dir, "content/docs/_index.md", "---\ntitle: Docs\n---\n")

	// Unversioned page: HAS a "Broken Targets" heading → #broken-targets is valid here.
	writeFixture(t, dir, "content/docs/guide/link-test.md",
		"---\ntitle: Link Test (unversioned)\n---\n"+
			"## Broken Targets\n\nAnchor to [section](#broken-targets).\n")

	// v2 (latest) page: NO "Broken Targets" heading → #broken-targets is broken here.
	writeFixture(t, dir, "content/docs/v2/_index.md", "---\ntitle: Docs v2\n---\n")
	writeFixture(t, dir, "content/docs/v2/guide/link-test.md",
		"---\ntitle: Link Test (v2)\n---\n"+
			"## Broken Anchor\n\nAnchor to [section](#broken-targets).\n")

	// v1 page so the versions list is fully populated.
	writeFixture(t, dir, "content/docs/v1/_index.md", "---\ntitle: Docs v1\n---\n")
	writeFixture(t, dir, "content/docs/v1/guide/link-test.md",
		"---\ntitle: Link Test (v1)\n---\n## Intro\n")

	return dir
}

func collisionCheckConfig() *config.SiteConfig {
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
	// Record anchor findings (not "ignore") so the flip is observable as a count.
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

// TestCheck_BrokenAnchorCountIsDeterministic guards the nondeterministic
// broken_anchor regression at the full pipeline level: a fresh SiteBuilder runs
// Check() many times on the same colliding fixture and must return an identical
// broken_anchor count and finding set every time. Before the fix, the fr/ar
// fallback collision flipped the count between runs (observed 5/3/4 on the real
// test site).
func TestCheck_BrokenAnchorCountIsDeterministic(t *testing.T) {
	dir := createCollisionFixture(t)

	run := func() (int, []string) {
		builder := NewSiteBuilder(BuildOptions{
			ProjectDir:  dir,
			Config:      collisionCheckConfig(),
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
