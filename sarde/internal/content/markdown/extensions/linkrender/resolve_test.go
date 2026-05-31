package linkrender

import (
	"testing"

	"github.com/frostybee/sarde/internal/engine"
)

// mockPageIndex implements PageLookup for testing.
type mockPageIndex struct {
	pages map[laneEntry]*engine.Page
}

type laneEntry struct {
	relPermalink string
	lang         string
	version      string
}

func (m *mockPageIndex) LookupInLane(relPermalink, lang, version string) *engine.Page {
	return m.pages[laneEntry{relPermalink, lang, version}]
}

// mockURLResolver returns a predictable URL for testing.
func mockURLResolver(relPath, lang, version string) string {
	result := ""
	if lang != "" && lang != "en" {
		result += "/" + lang
	}
	if version != "" {
		// Insert version after collection mount (simplified: just append).
		result += relPath
		// For testing, insert version before the path tail.
		// Simplified: basePath + lang + version + relPath
		return result[:0] + "/" + lang + relPath[:len(relPath)-1] + "/" // skip
	}
	result += relPath
	return result
}

// simpleResolver just prepends /base + lang prefix for non-default lang.
func simpleResolver(relPath, lang, version string) string {
	url := "/base"
	if lang != "" && lang != "en" {
		url += "/" + lang
	}
	if version != "" {
		// Insert version in the path after the collection mount.
		// For simplicity in tests, just prepend version segment.
		url += relPath
		// Strip trailing slash, add version.
		if len(url) > 0 && url[len(url)-1] == '/' {
			url = url[:len(url)-1]
		}
		return url + "/"
	}
	url += relPath
	return url
}

// identityResolver returns relPath unchanged (no basePath, no prefixes).
func identityResolver(relPath, lang, version string) string {
	return relPath
}

func makeCollection(name string, versioned bool, lastVersion string) *engine.Collection {
	col := &engine.Collection{
		Name: name,
		Config: &engine.CollectionConfig{
			Versioning: nil,
		},
	}
	if versioned {
		col.Config.Versioning = &engine.VersionConfig{
			Enabled:     true,
			LastVersion: lastVersion,
		}
	}
	return col
}

func TestResolveInternalLink_External(t *testing.T) {
	page := &engine.Page{
		PageIdentity: engine.PageIdentity{RelPermalink: "/docs/intro/"},
		PageI18n:     engine.PageI18n{Lang: "en", LangRelPath: "docs/intro.md"},
	}

	dest := ClassifyDest("https://example.com/page")
	result := ResolveInternalLink(dest, page, &mockPageIndex{}, identityResolver)

	if !result.Found {
		t.Fatal("external link should be Found")
	}
	if result.URL != "https://example.com/page" {
		t.Errorf("URL = %q, want external URL unchanged", result.URL)
	}
}

func TestResolveInternalLink_AnchorOnly(t *testing.T) {
	page := &engine.Page{
		PageIdentity: engine.PageIdentity{RelPermalink: "/docs/intro/", Permalink: "/docs/intro/"},
		PageI18n:     engine.PageI18n{Lang: "en", LangRelPath: "docs/intro.md"},
	}

	dest := ClassifyDest("#setup")
	result := ResolveInternalLink(dest, page, &mockPageIndex{}, identityResolver)

	if !result.Found {
		t.Fatal("anchor-only link should be Found")
	}
	if result.URL != "/docs/intro/#setup" {
		t.Errorf("URL = %q, want /docs/intro/#setup", result.URL)
	}
	if result.TargetPermalink != "/docs/intro/" {
		t.Errorf("TargetPermalink = %q, want /docs/intro/", result.TargetPermalink)
	}
}

func TestResolveInternalLink_Ambiguous(t *testing.T) {
	page := &engine.Page{
		PageIdentity: engine.PageIdentity{RelPermalink: "/docs/intro/"},
		PageI18n:     engine.PageI18n{Lang: "en", LangRelPath: "docs/intro.md"},
	}

	dest := ClassifyDest("auth.md")
	result := ResolveInternalLink(dest, page, &mockPageIndex{}, identityResolver)

	if result.Found {
		t.Error("ambiguous link should not be Found")
	}
}

func TestResolveInternalLink_RelativeSameDir(t *testing.T) {
	col := makeCollection("docs", false, "")
	targetPage := &engine.Page{
		PageIdentity:   engine.PageIdentity{RelPermalink: "/docs/guides/auth/", Permalink: "/docs/guides/auth/"},
		PageI18n:       engine.PageI18n{Lang: "en"},
		PageVersioning: engine.PageVersioning{Version: ""},
	}

	idx := &mockPageIndex{
		pages: map[laneEntry]*engine.Page{
			{"/docs/guides/auth/", "en", ""}: targetPage,
		},
	}

	currentPage := &engine.Page{
		PageIdentity:    engine.PageIdentity{RelPermalink: "/docs/guides/intro/"},
		PageI18n:        engine.PageI18n{Lang: "en", LangRelPath: "docs/guides/intro.md"},
		PageRelationships: engine.PageRelationships{Collection: col},
	}

	dest := ClassifyDest("./auth.md")
	result := ResolveInternalLink(dest, currentPage, idx, identityResolver)

	if !result.Found {
		t.Fatal("relative link should be Found")
	}
	if result.URL != "/docs/guides/auth/" {
		t.Errorf("URL = %q, want /docs/guides/auth/", result.URL)
	}
}

func TestResolveInternalLink_RelativeParent(t *testing.T) {
	targetPage := &engine.Page{
		PageIdentity: engine.PageIdentity{RelPermalink: "/docs/intro/", Permalink: "/docs/intro/"},
		PageI18n:     engine.PageI18n{Lang: "en"},
	}

	idx := &mockPageIndex{
		pages: map[laneEntry]*engine.Page{
			{"/docs/intro/", "en", ""}: targetPage,
		},
	}

	currentPage := &engine.Page{
		PageIdentity: engine.PageIdentity{RelPermalink: "/docs/guides/auth/"},
		PageI18n:     engine.PageI18n{Lang: "en", LangRelPath: "docs/guides/auth.md"},
	}

	dest := ClassifyDest("../intro.md")
	result := ResolveInternalLink(dest, currentPage, idx, identityResolver)

	if !result.Found {
		t.Fatal("relative parent link should be Found")
	}
	if result.URL != "/docs/intro/" {
		t.Errorf("URL = %q, want /docs/intro/", result.URL)
	}
}

func TestResolveInternalLink_RelativeWithFragment(t *testing.T) {
	targetPage := &engine.Page{
		PageIdentity: engine.PageIdentity{RelPermalink: "/docs/guides/auth/", Permalink: "/docs/guides/auth/"},
		PageI18n:     engine.PageI18n{Lang: "en"},
	}

	idx := &mockPageIndex{
		pages: map[laneEntry]*engine.Page{
			{"/docs/guides/auth/", "en", ""}: targetPage,
		},
	}

	currentPage := &engine.Page{
		PageIdentity: engine.PageIdentity{RelPermalink: "/docs/guides/intro/"},
		PageI18n:     engine.PageI18n{Lang: "en", LangRelPath: "docs/guides/intro.md"},
	}

	dest := ClassifyDest("./auth.md#api-keys")
	result := ResolveInternalLink(dest, currentPage, idx, identityResolver)

	if !result.Found {
		t.Fatal("link with fragment should be Found")
	}
	if result.URL != "/docs/guides/auth/#api-keys" {
		t.Errorf("URL = %q, want /docs/guides/auth/#api-keys", result.URL)
	}
	if result.TargetPermalink != "/docs/guides/auth/" {
		t.Errorf("TargetPermalink = %q, want /docs/guides/auth/", result.TargetPermalink)
	}
}

func TestResolveInternalLink_ContentRoot(t *testing.T) {
	col := makeCollection("docs", false, "")
	targetPage := &engine.Page{
		PageIdentity: engine.PageIdentity{RelPermalink: "/docs/guides/auth/", Permalink: "/docs/guides/auth/"},
		PageI18n:     engine.PageI18n{Lang: "en"},
	}

	idx := &mockPageIndex{
		pages: map[laneEntry]*engine.Page{
			{"/docs/guides/auth/", "en", ""}: targetPage,
		},
	}

	currentPage := &engine.Page{
		PageIdentity:    engine.PageIdentity{RelPermalink: "/docs/intro/"},
		PageI18n:        engine.PageI18n{Lang: "en", LangRelPath: "docs/intro.md"},
		PageRelationships: engine.PageRelationships{Collection: col},
	}

	dest := ClassifyDest("/guides/auth.md")
	result := ResolveInternalLink(dest, currentPage, idx, identityResolver)

	if !result.Found {
		t.Fatal("content-root link should be Found")
	}
	if result.URL != "/docs/guides/auth/" {
		t.Errorf("URL = %q, want /docs/guides/auth/", result.URL)
	}
}

func TestResolveInternalLink_NotFound(t *testing.T) {
	idx := &mockPageIndex{pages: map[laneEntry]*engine.Page{}}

	currentPage := &engine.Page{
		PageIdentity: engine.PageIdentity{RelPermalink: "/docs/intro/"},
		PageI18n:     engine.PageI18n{Lang: "en", LangRelPath: "docs/intro.md"},
	}

	dest := ClassifyDest("./missing.md")
	result := ResolveInternalLink(dest, currentPage, idx, identityResolver)

	if result.Found {
		t.Error("link to missing page should not be Found")
	}
}

func TestResolveInternalLink_VersionedLatest(t *testing.T) {
	col := makeCollection("docs", true, "v2")
	targetPage := &engine.Page{
		PageIdentity:   engine.PageIdentity{RelPermalink: "/docs/guides/auth/", Permalink: "/docs/guides/auth/"},
		PageI18n:       engine.PageI18n{Lang: "en"},
		PageVersioning: engine.PageVersioning{Version: "v2"},
	}

	idx := &mockPageIndex{
		pages: map[laneEntry]*engine.Page{
			{"/docs/guides/auth/", "en", "v2"}: targetPage,
		},
	}

	currentPage := &engine.Page{
		PageIdentity:      engine.PageIdentity{RelPermalink: "/docs/intro/"},
		PageI18n:          engine.PageI18n{Lang: "en", LangRelPath: "docs/intro.md"},
		PageVersioning:    engine.PageVersioning{Version: "v2"},
		PageRelationships: engine.PageRelationships{Collection: col},
	}

	// For the latest version, resolvePageVersion returns "" so urlResolver gets version="".
	dest := ClassifyDest("./guides/auth.md")
	result := ResolveInternalLink(dest, currentPage, idx, identityResolver)

	if !result.Found {
		t.Fatal("versioned latest link should be Found")
	}
	// identityResolver ignores version, so URL is just the relPermalink.
	if result.URL != "/docs/guides/auth/" {
		t.Errorf("URL = %q, want /docs/guides/auth/", result.URL)
	}
}

func TestResolveInternalLink_VersionedOlder(t *testing.T) {
	col := makeCollection("docs", true, "v2")
	targetPage := &engine.Page{
		PageIdentity:   engine.PageIdentity{RelPermalink: "/docs/guides/auth/", Permalink: "/docs/v1/guides/auth/"},
		PageI18n:       engine.PageI18n{Lang: "en"},
		PageVersioning: engine.PageVersioning{Version: "v1"},
	}

	idx := &mockPageIndex{
		pages: map[laneEntry]*engine.Page{
			{"/docs/guides/auth/", "en", "v1"}: targetPage,
		},
	}

	// Current page is v1 (older version).
	currentPage := &engine.Page{
		PageIdentity:      engine.PageIdentity{RelPermalink: "/docs/intro/"},
		PageI18n:          engine.PageI18n{Lang: "en", LangRelPath: "docs/intro.md"},
		PageVersioning:    engine.PageVersioning{Version: "v1"},
		PageRelationships: engine.PageRelationships{Collection: col},
	}

	// Use a resolver that includes version in URL when non-empty.
	versionResolver := func(relPath, lang, version string) string {
		if version != "" {
			return "/docs/" + version + "/" + relPath[len("/docs/"):]
		}
		return relPath
	}

	dest := ClassifyDest("./guides/auth.md")
	result := ResolveInternalLink(dest, currentPage, idx, versionResolver)

	if !result.Found {
		t.Fatal("versioned older link should be Found")
	}
	if result.URL != "/docs/v1/guides/auth/" {
		t.Errorf("URL = %q, want /docs/v1/guides/auth/", result.URL)
	}
}

func TestResolveInternalLink_DirectoryFallback(t *testing.T) {
	// Link to ./guides/ should resolve to the section index (guides/_index.md).
	targetPage := &engine.Page{
		PageIdentity: engine.PageIdentity{RelPermalink: "/docs/guides/", Permalink: "/docs/guides/"},
		PageI18n:     engine.PageI18n{Lang: "en"},
	}

	idx := &mockPageIndex{
		pages: map[laneEntry]*engine.Page{
			{"/docs/guides/", "en", ""}: targetPage,
		},
	}

	currentPage := &engine.Page{
		PageIdentity: engine.PageIdentity{RelPermalink: "/docs/intro/"},
		PageI18n:     engine.PageI18n{Lang: "en", LangRelPath: "docs/intro.md"},
	}

	dest := ClassifyDest("./guides/")
	result := ResolveInternalLink(dest, currentPage, idx, identityResolver)

	if !result.Found {
		t.Fatal("directory link should resolve to section index")
	}
	if result.URL != "/docs/guides/" {
		t.Errorf("URL = %q, want /docs/guides/", result.URL)
	}
}

func TestResolveInternalLink_LaneIsolation(t *testing.T) {
	// French page should NOT find an English-only page.
	enPage := &engine.Page{
		PageIdentity: engine.PageIdentity{RelPermalink: "/docs/guides/auth/", Permalink: "/docs/guides/auth/"},
		PageI18n:     engine.PageI18n{Lang: "en"},
	}

	idx := &mockPageIndex{
		pages: map[laneEntry]*engine.Page{
			{"/docs/guides/auth/", "en", ""}: enPage,
		},
	}

	currentPage := &engine.Page{
		PageIdentity: engine.PageIdentity{RelPermalink: "/docs/intro/"},
		PageI18n:     engine.PageI18n{Lang: "fr", LangRelPath: "docs/intro.md"},
	}

	dest := ClassifyDest("./guides/auth.md")
	result := ResolveInternalLink(dest, currentPage, idx, identityResolver)

	if result.Found {
		t.Error("French page should NOT find English-only target (lane isolation)")
	}
}

func TestResolveInternalLink_WithQuery(t *testing.T) {
	targetPage := &engine.Page{
		PageIdentity: engine.PageIdentity{RelPermalink: "/docs/auth/", Permalink: "/docs/auth/"},
		PageI18n:     engine.PageI18n{Lang: "en"},
	}

	idx := &mockPageIndex{
		pages: map[laneEntry]*engine.Page{
			{"/docs/auth/", "en", ""}: targetPage,
		},
	}

	currentPage := &engine.Page{
		PageIdentity: engine.PageIdentity{RelPermalink: "/docs/intro/"},
		PageI18n:     engine.PageI18n{Lang: "en", LangRelPath: "docs/intro.md"},
	}

	dest := ClassifyDest("./auth.md?highlight=true#section")
	result := ResolveInternalLink(dest, currentPage, idx, identityResolver)

	if !result.Found {
		t.Fatal("link with query+fragment should be Found")
	}
	if result.URL != "/docs/auth/#section?highlight=true" {
		t.Errorf("URL = %q, want /docs/auth/#section?highlight=true", result.URL)
	}
}
