package build

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/getsarde/sarde/embedded"
	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/content"
	"github.com/getsarde/sarde/internal/engine"
	"github.com/getsarde/sarde/internal/links"
)

// replayFixture builds a two-page index (intro links to auth) with a resolver
// and collection registry, mirroring the linkrender test fixtures.
func replayFixture(t *testing.T) (current, target *engine.Page, idx *content.PageIndex, resolver *engine.URLResolver, cols map[string]*engine.Collection) {
	t.Helper()
	docsCol := &engine.Collection{Name: "docs", Config: &engine.CollectionConfig{}}
	target = &engine.Page{
		PageIdentity:      engine.PageIdentity{RelPermalink: "/docs/guide/auth/", Permalink: "/docs/guide/auth/", Slug: "auth"},
		PageI18n:          engine.PageI18n{Lang: "en", LangRelPath: "docs/guide/auth.md"},
		PageRelationships: engine.PageRelationships{Collection: docsCol},
	}
	current = &engine.Page{
		PageIdentity:      engine.PageIdentity{RelPermalink: "/docs/guide/intro/", Permalink: "/docs/guide/intro/", Slug: "intro", FilePath: "content/docs/guide/intro.md"},
		PageI18n:          engine.PageI18n{Lang: "en", LangRelPath: "docs/guide/intro.md"},
		PageRelationships: engine.PageRelationships{Collection: docsCol},
	}
	idx = content.BuildPageIndex([]*engine.Page{target, current})
	resolver = &engine.URLResolver{BasePath: "/", DefaultLang: "en"}
	cols = map[string]*engine.Collection{"docs": docsCol}
	return current, target, idx, resolver, cols
}

func TestTargetLinkStale(t *testing.T) {
	_, _, idx, _, _ := replayFixture(t)

	if targetLinkStale(idx, "/docs/guide/auth/", "/docs/guide/auth/") {
		t.Error("unchanged target reported stale")
	}
	if !targetLinkStale(idx, "/docs/guide/gone/", "/docs/guide/gone/") {
		t.Error("deleted target not reported stale")
	}
	// Permalink takeover: same permalink now occupied by a page with a
	// different RelPermalink.
	if !targetLinkStale(idx, "/docs/guide/auth/", "/old-location/") {
		t.Error("RelPermalink mismatch not reported stale")
	}
}

func TestRefStale(t *testing.T) {
	current, _, idx, resolver, cols := replayFixture(t)

	cases := []struct {
		name string
		ref  CachedLinkRef
		want bool
	}{
		{"OK unchanged", CachedLinkRef{Status: links.StatusOK, TargetPermalink: "/docs/guide/auth/", TargetRelPermalink: "/docs/guide/auth/"}, false},
		{"OK target deleted", CachedLinkRef{Status: links.StatusOK, TargetPermalink: "/docs/guide/gone/", TargetRelPermalink: "/docs/guide/gone/"}, true},
		{"OK permalink takeover", CachedLinkRef{Status: links.StatusOK, TargetPermalink: "/docs/guide/auth/", TargetRelPermalink: "/old-location/"}, true},
		{"BrokenTarget still broken", CachedLinkRef{Status: links.StatusBrokenTarget, RawDest: "./missing.md"}, false},
		{"BrokenTarget now resolves", CachedLinkRef{Status: links.StatusBrokenTarget, RawDest: "./auth.md"}, true},
		{"Unverified still unresolved", CachedLinkRef{Status: links.StatusUnverified, RawDest: "/docs/guide/nope"}, false},
		{"Unverified now resolves site-absolute", CachedLinkRef{Status: links.StatusUnverified, RawDest: "/docs/guide/auth"}, true},
		// Collection-relative content-root form: only resolvable through
		// ResolveInternalLink, not the site-absolute lane lookup. This is the
		// docs-site regression where healed targets were never re-detected.
		{"Unverified now resolves content-root", CachedLinkRef{Status: links.StatusUnverified, RawDest: "/guide/auth"}, true},
		{"External never stale", CachedLinkRef{Status: links.StatusExternal, RawDest: "https://example.com/x", TargetPermalink: "/garbage/"}, false},
	}
	for _, tc := range cases {
		if got := refStale(tc.ref, current, idx, resolver, cols); got != tc.want {
			t.Errorf("%s: refStale = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestEntryStale(t *testing.T) {
	current, _, idx, resolver, cols := replayFixture(t)

	clean := &CacheEntry{
		Refs:           []CachedLinkRef{{Status: links.StatusOK, TargetPermalink: "/docs/guide/auth/", TargetRelPermalink: "/docs/guide/auth/"}},
		PendingAnchors: []CachedAnchorCheck{{TargetPermalink: "/docs/guide/auth/", TargetRelPermalink: "/docs/guide/auth/", Fragment: "setup"}},
	}
	if entryStale(clean, current, idx, resolver, cols) {
		t.Error("clean entry reported stale")
	}

	// A stale anchor target marks the entry stale even with zero refs.
	anchorOnly := &CacheEntry{
		PendingAnchors: []CachedAnchorCheck{{TargetPermalink: "/docs/guide/gone/", TargetRelPermalink: "/docs/guide/gone/", Fragment: "setup"}},
	}
	if !entryStale(anchorOnly, current, idx, resolver, cols) {
		t.Error("entry with deleted anchor target not reported stale")
	}

	staleRef := &CacheEntry{
		Refs: []CachedLinkRef{{Status: links.StatusOK, TargetPermalink: "/docs/guide/gone/", TargetRelPermalink: "/docs/guide/gone/"}},
	}
	if !entryStale(staleRef, current, idx, resolver, cols) {
		t.Error("entry with deleted ref target not reported stale")
	}

	if entryStale(staleRef, current, nil, resolver, cols) {
		t.Error("nil index must disable staleness checks")
	}
}

func TestCachedRefsRoundTrip(t *testing.T) {
	current, target, idx, _, _ := replayFixture(t)

	recorded := []links.LinkRef{{
		FromPage:   target, // deliberately wrong page pointer: replay must rebuild from the live page
		FromFile:   "stale/path.md",
		RawDest:    "./auth.md",
		Kind:       links.KindRelative,
		Resolved:   "/docs/guide/auth/",
		TargetPage: target,
		Status:     links.StatusOK,
	}}
	cached := toCachedRefs(recorded)
	if len(cached) != 1 {
		t.Fatalf("cached %d refs, want 1", len(cached))
	}
	if cached[0].TargetPermalink != "/docs/guide/auth/" || cached[0].TargetRelPermalink != "/docs/guide/auth/" {
		t.Errorf("target fields not captured: %+v", cached[0])
	}

	replayed := replayCachedRefs(cached, current, idx)
	if len(replayed) != 1 {
		t.Fatalf("replayed %d refs, want 1", len(replayed))
	}
	r := replayed[0]
	if r.FromPage != current || r.FromFile != current.FilePath {
		t.Errorf("page-derived fields not rebuilt live: FromFile = %q", r.FromFile)
	}
	if r.Dim != (links.DimKey{Collection: "docs", Lang: "en"}) {
		t.Errorf("Dim = %+v, want docs/en", r.Dim)
	}
	if r.TargetPage != target {
		t.Errorf("TargetPage not re-looked-up from live index")
	}
	if r.RawDest != "./auth.md" || r.Kind != links.KindRelative || r.Resolved != "/docs/guide/auth/" || r.Status != links.StatusOK {
		t.Errorf("content-derived fields not preserved: %+v", r)
	}
}

// --- integration tests ---

func cacheReplayFixtureDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeFixture(t, dir, "content/_index.md", "---\ntitle: Home\n---\n# Home\n")
	writeFixture(t, dir, "content/docs/_index.md", "---\ntitle: Docs\n---\n")
	writeFixture(t, dir, "content/docs/a.md",
		"---\ntitle: A\n---\nSee [B](./b.md), [section](./b.md#setup), [root form](/b), and [ext](https://example.com/x).\n")
	writeFixture(t, dir, "content/docs/b.md", "---\ntitle: B\n---\n## Setup\n\nBody.\n")
	return dir
}

func cacheReplayConfig() *config.SiteConfig {
	cfg := config.Defaults()
	cfg.Build.Minify = config.BoolPtr(false)
	// Build.Cache stays at its default (enabled): warm-cache behavior is what
	// these tests exercise.
	cfg.LinkValidation.OnBroken = "warn"
	cfg.LinkValidation.OnBrokenAnchor = "warn"
	return cfg
}

func newCacheReplayBuilder(dir string) *SiteBuilder {
	return NewSiteBuilder(BuildOptions{
		ProjectDir:  dir,
		Config:      cacheReplayConfig(),
		ThemeConfig: buildThemeConfig(),
		EmbeddedFS:  embedded.ThemeFS(),
	})
}

func brokenTargetCount(findings []links.Finding) int {
	n := 0
	for _, f := range findings {
		if f.Type == links.FindingBrokenTarget {
			n++
		}
	}
	return n
}

// TestBuild_WarmCacheCoverageParity is the regression guard for the "checked
// 132 links cold vs 58 warm" symptom: a warm-cache build must report the same
// link coverage and findings as a cold build.
func TestBuild_WarmCacheCoverageParity(t *testing.T) {
	dir := cacheReplayFixtureDir(t)
	b := newCacheReplayBuilder(dir)

	if _, err := b.Build(); err != nil {
		t.Fatalf("cold build failed: %v", err)
	}
	coldLinks := b.lastCoverage.TotalLinks
	if coldLinks < 3 {
		t.Fatalf("fixture should produce at least 3 links, got %d", coldLinks)
	}
	if b.checkReportResult == nil {
		t.Fatal("cold build produced no link report")
	}
	coldFindings := findingKeys(b.checkReportResult.Findings)

	if _, err := b.Build(); err != nil {
		t.Fatalf("warm build failed: %v", err)
	}
	if warm := b.lastCoverage.TotalLinks; warm != coldLinks {
		t.Errorf("warm TotalLinks = %d, cold = %d: page cache starved the link graph", warm, coldLinks)
	}
	warmFindings := findingKeys(b.checkReportResult.Findings)
	if !slices.Equal(coldFindings, warmFindings) {
		t.Errorf("warm findings %v differ from cold findings %v", warmFindings, coldFindings)
	}
}

// TestBuild_WarmCache_DeletedTargetSelfHeals: deleting a target page must be
// re-detected even though the linking source page is served from the on-disk
// cache by a fresh builder process, and the source page must be re-rendered
// so its output no longer bakes the stale URL.
func TestBuild_WarmCache_DeletedTargetSelfHeals(t *testing.T) {
	dir := cacheReplayFixtureDir(t)

	b1 := newCacheReplayBuilder(dir)
	if _, err := b1.Build(); err != nil {
		t.Fatalf("first build failed: %v", err)
	}
	if n := brokenTargetCount(b1.checkReportResult.Findings); n != 0 {
		t.Fatalf("clean fixture produced %d broken-target findings", n)
	}

	if err := os.Remove(filepath.Join(dir, "content", "docs", "b.md")); err != nil {
		t.Fatal(err)
	}

	b2 := newCacheReplayBuilder(dir)
	res, err := b2.Build()
	if err != nil {
		t.Fatalf("second build failed: %v", err)
	}
	if n := brokenTargetCount(b2.checkReportResult.Findings); n == 0 {
		t.Error("deleted target behind cached source page was not re-detected")
	}
	html, err := os.ReadFile(filepath.Join(res.OutputDir, "docs", "a", "index.html"))
	if err != nil {
		t.Fatalf("reading source page output: %v", err)
	}
	if strings.Contains(string(html), `href="/docs/b/"`) {
		t.Error("source page output still bakes the deleted target's URL: cache entry was not self-healed")
	}
}

// TestBuild_WarmCache_RenamedTargetSelfHeals: renaming a target page changes
// its permalink; the cached source page must re-render and report the broken
// link instead of silently replaying the old resolution.
func TestBuild_WarmCache_RenamedTargetSelfHeals(t *testing.T) {
	dir := cacheReplayFixtureDir(t)

	b1 := newCacheReplayBuilder(dir)
	if _, err := b1.Build(); err != nil {
		t.Fatalf("first build failed: %v", err)
	}
	// Baseline is not empty: ./b.md links produce relative_link policy
	// warnings under the default on_relative_links: warn.
	baseline := findingKeys(b1.checkReportResult.Findings)

	oldPath := filepath.Join(dir, "content", "docs", "b.md")
	newPath := filepath.Join(dir, "content", "docs", "c.md")
	if err := os.Rename(oldPath, newPath); err != nil {
		t.Fatal(err)
	}

	b2 := newCacheReplayBuilder(dir)
	if _, err := b2.Build(); err != nil {
		t.Fatalf("second build failed: %v", err)
	}
	if n := brokenTargetCount(b2.checkReportResult.Findings); n == 0 {
		t.Error("renamed target behind cached source page was not re-detected")
	}

	// Restoring the target must heal the cached broken/unverified refs: a
	// third build has to return to the baseline findings, not replay the
	// failure snapshots forever.
	if err := os.Rename(newPath, oldPath); err != nil {
		t.Fatal(err)
	}
	b3 := newCacheReplayBuilder(dir)
	if _, err := b3.Build(); err != nil {
		t.Fatalf("restored build failed: %v", err)
	}
	restored := findingKeys(b3.checkReportResult.Findings)
	if !slices.Equal(restored, baseline) {
		t.Errorf("restored build findings %v differ from baseline %v", restored, baseline)
	}
}
