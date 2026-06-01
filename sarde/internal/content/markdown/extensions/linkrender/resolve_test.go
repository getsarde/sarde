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

// ctxOf builds a ResolveContext with no collection registry (nil Collections),
// exercising the same-lane fallback path.
func ctxOf(idx PageLookup, resolver URLResolverFunc) ResolveContext {
	return ResolveContext{PageIndex: idx, URLResolver: resolver}
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
	result := ResolveInternalLink(dest, page, ctxOf(&mockPageIndex{}, identityResolver))

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
	result := ResolveInternalLink(dest, page, ctxOf(&mockPageIndex{}, identityResolver))

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
	result := ResolveInternalLink(dest, page, ctxOf(&mockPageIndex{}, identityResolver))

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
	result := ResolveInternalLink(dest, currentPage, ctxOf(idx, identityResolver))

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
	result := ResolveInternalLink(dest, currentPage, ctxOf(idx, identityResolver))

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
	result := ResolveInternalLink(dest, currentPage, ctxOf(idx, identityResolver))

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
	result := ResolveInternalLink(dest, currentPage, ctxOf(idx, identityResolver))

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
	result := ResolveInternalLink(dest, currentPage, ctxOf(idx, identityResolver))

	if result.Found {
		t.Error("link to missing page should not be Found")
	}
}

func TestResolveInternalLink_VersionedLatest(t *testing.T) {
	col := makeCollection("docs", true, "v2")
	targetPage := &engine.Page{
		PageIdentity:      engine.PageIdentity{RelPermalink: "/docs/guides/auth/", Permalink: "/docs/guides/auth/"},
		PageI18n:          engine.PageI18n{Lang: "en"},
		PageVersioning:    engine.PageVersioning{Version: "v2"},
		PageRelationships: engine.PageRelationships{Collection: col},
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
	result := ResolveInternalLink(dest, currentPage, ctxOf(idx, identityResolver))

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
		PageIdentity:      engine.PageIdentity{RelPermalink: "/docs/guides/auth/", Permalink: "/docs/v1/guides/auth/"},
		PageI18n:          engine.PageI18n{Lang: "en"},
		PageVersioning:    engine.PageVersioning{Version: "v1"},
		PageRelationships: engine.PageRelationships{Collection: col},
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
	result := ResolveInternalLink(dest, currentPage, ctxOf(idx, versionResolver))

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
	result := ResolveInternalLink(dest, currentPage, ctxOf(idx, identityResolver))

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
	result := ResolveInternalLink(dest, currentPage, ctxOf(idx, identityResolver))

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
	result := ResolveInternalLink(dest, currentPage, ctxOf(idx, identityResolver))

	if !result.Found {
		t.Fatal("link with query+fragment should be Found")
	}
	if result.URL != "/docs/auth/?highlight=true#section" {
		t.Errorf("URL = %q, want /docs/auth/?highlight=true#section", result.URL)
	}
}

// versionInPath is a test resolver that inserts /<version>/ after the /docs mount.
func versionInPath(relPath, lang, version string) string {
	if version != "" {
		return "/docs/" + version + relPath[len("/docs"):]
	}
	return relPath
}

func TestResolveInternalLink_CrossDimension_VersionedToUnversioned(t *testing.T) {
	docs := makeCollection("docs", true, "v2")
	blog := makeCollection("blog", false, "")

	// The blog post lives in the unversioned (en, "") lane.
	blogPost := &engine.Page{
		PageIdentity:      engine.PageIdentity{RelPermalink: "/blog/hello/", Permalink: "/blog/hello/"},
		PageI18n:          engine.PageI18n{Lang: "en"},
		PageRelationships: engine.PageRelationships{Collection: blog},
	}
	idx := &mockPageIndex{
		pages: map[laneEntry]*engine.Page{
			{"/blog/hello/", "en", ""}: blogPost,
		},
	}

	// Source page is docs v1 (older version), linking out into the blog.
	currentPage := &engine.Page{
		PageIdentity:      engine.PageIdentity{RelPermalink: "/docs/guide/auth/"},
		PageI18n:          engine.PageI18n{Lang: "en", LangRelPath: "docs/guide/auth.md"},
		PageVersioning:    engine.PageVersioning{Version: "v1", VersionRelPath: "guide/auth"},
		PageRelationships: engine.PageRelationships{Collection: docs},
	}

	ctx := ResolveContext{
		PageIndex:   idx,
		URLResolver: versionInPath,
		Collections: map[string]*engine.Collection{"docs": docs, "blog": blog},
	}

	dest := ClassifyDest("../../blog/hello.md")
	result := ResolveInternalLink(dest, currentPage, ctx)

	if !result.Found {
		t.Fatal("docs(v1)→blog link should resolve into the unversioned blog")
	}
	if result.URL != "/blog/hello/" {
		t.Errorf("URL = %q, want /blog/hello/ (version coordinate dropped)", result.URL)
	}
}

func TestResolveInternalLink_CrossDimension_SameVersionedCollection(t *testing.T) {
	docs := makeCollection("docs", true, "v2")

	target := &engine.Page{
		PageIdentity:      engine.PageIdentity{RelPermalink: "/docs/guide/api/", Permalink: "/docs/v1/guide/api/"},
		PageI18n:          engine.PageI18n{Lang: "en"},
		PageVersioning:    engine.PageVersioning{Version: "v1"},
		PageRelationships: engine.PageRelationships{Collection: docs},
	}
	idx := &mockPageIndex{
		pages: map[laneEntry]*engine.Page{
			{"/docs/guide/api/", "en", "v1"}: target,
		},
	}

	currentPage := &engine.Page{
		PageIdentity:      engine.PageIdentity{RelPermalink: "/docs/guide/auth/"},
		PageI18n:          engine.PageI18n{Lang: "en", LangRelPath: "docs/guide/auth.md"},
		PageVersioning:    engine.PageVersioning{Version: "v1", VersionRelPath: "guide/auth"},
		PageRelationships: engine.PageRelationships{Collection: docs},
	}

	ctx := ResolveContext{
		PageIndex:   idx,
		URLResolver: versionInPath,
		Collections: map[string]*engine.Collection{"docs": docs},
	}

	dest := ClassifyDest("./api.md")
	result := ResolveInternalLink(dest, currentPage, ctx)

	if !result.Found {
		t.Fatal("same-collection v1 link should resolve in the v1 lane")
	}
	if result.URL != "/docs/v1/guide/api/" {
		t.Errorf("URL = %q, want /docs/v1/guide/api/ (version preserved)", result.URL)
	}
}

func TestResolveInternalLink_CrossDimension_TargetCollectionMissing(t *testing.T) {
	docs := makeCollection("docs", true, "v2")

	// Target exists only in the v1 lane; its collection is NOT in the registry.
	target := &engine.Page{
		PageIdentity:   engine.PageIdentity{RelPermalink: "/other/page/", Permalink: "/docs/v1/other/page/"},
		PageI18n:       engine.PageI18n{Lang: "en"},
		PageVersioning: engine.PageVersioning{Version: "v1"},
	}
	idx := &mockPageIndex{
		pages: map[laneEntry]*engine.Page{
			{"/other/page/", "en", "v1"}: target,
		},
	}

	currentPage := &engine.Page{
		PageIdentity:      engine.PageIdentity{RelPermalink: "/docs/guide/auth/"},
		PageI18n:          engine.PageI18n{Lang: "en", LangRelPath: "docs/guide/auth.md"},
		PageVersioning:    engine.PageVersioning{Version: "v1", VersionRelPath: "guide/auth"},
		PageRelationships: engine.PageRelationships{Collection: docs},
	}

	ctx := ResolveContext{
		PageIndex:   idx,
		URLResolver: identityResolver,
		Collections: map[string]*engine.Collection{"docs": docs}, // no "other"
	}

	dest := ClassifyDest("../../other/page.md")
	result := ResolveInternalLink(dest, currentPage, ctx)

	if result.Found {
		t.Error("unknown target collection must not silently jump to another version's lane")
	}
}

func TestResolveInternalLink_NilCollections_UsesCurrentVersion(t *testing.T) {
	docs := makeCollection("docs", true, "v2")

	target := &engine.Page{
		PageIdentity:      engine.PageIdentity{RelPermalink: "/docs/guide/api/", Permalink: "/docs/v1/guide/api/"},
		PageI18n:          engine.PageI18n{Lang: "en"},
		PageVersioning:    engine.PageVersioning{Version: "v1"},
		PageRelationships: engine.PageRelationships{Collection: docs},
	}
	idx := &mockPageIndex{
		pages: map[laneEntry]*engine.Page{
			{"/docs/guide/api/", "en", "v1"}: target,
		},
	}

	currentPage := &engine.Page{
		PageIdentity:      engine.PageIdentity{RelPermalink: "/docs/guide/auth/"},
		PageI18n:          engine.PageI18n{Lang: "en", LangRelPath: "docs/guide/auth.md"},
		PageVersioning:    engine.PageVersioning{Version: "v1", VersionRelPath: "guide/auth"},
		PageRelationships: engine.PageRelationships{Collection: docs},
	}

	// nil Collections ⇒ same-lane fallback uses currentPage.Version ("v1").
	dest := ClassifyDest("./api.md")
	result := ResolveInternalLink(dest, currentPage, ctxOf(idx, identityResolver))

	if !result.Found {
		t.Fatal("with nil Collections, same-version lookup should still resolve")
	}
}

func TestResolveInternalLink_SiteRoot(t *testing.T) {
	currentPage := &engine.Page{
		PageIdentity: engine.PageIdentity{RelPermalink: "/docs/guide/auth/"},
		PageI18n:     engine.PageI18n{Lang: "fr", LangRelPath: "docs/guide/auth.md"},
	}

	dest := ParsedDest{Kind: LinkSiteRoot, Path: "/pricing", Fragment: "plans", Raw: "site:/pricing#plans"}
	result := ResolveInternalLink(dest, currentPage, ctxOf(&mockPageIndex{}, identityResolver))

	if !result.Found {
		t.Fatal("site-root escape should always resolve")
	}
	// identityResolver echoes relPath with no lane segments; query/fragment order is path?query#fragment.
	if result.URL != "/pricing#plans" {
		t.Errorf("URL = %q, want /pricing#plans (no lane segments)", result.URL)
	}
}

// langVersionResolver inserts a non-default /<lang>/ outer prefix and a
// /<version>/ segment after the /docs mount, mirroring engine.URLResolver closely
// enough for assertions. lang "" or "en" yields no prefix; version "" yields none.
// (versionInPath ignores lang, so cross-lane ?lang= tests use this resolver.)
func langVersionResolver(relPath, lang, version string) string {
	out := relPath
	if version != "" {
		out = "/docs/" + version + out[len("/docs"):]
	}
	if lang != "" && lang != "en" {
		out = "/" + lang + out
	}
	return out
}

func TestExtractLaneOverrides(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		wantLang    string
		wantHasLang bool
		wantVer     string
		wantHasVer  bool
		wantRest    string
	}{
		{"empty", "", "", false, "", false, ""},
		{"ordinary", "highlight=true", "", false, "", false, "highlight=true"},
		{"lang only", "lang=de", "de", true, "", false, ""},
		{"version only", "version=v1", "", false, "v1", true, ""},
		{"both", "lang=de&version=v1", "de", true, "v1", true, ""},
		{"lang plus other", "lang=de&tab=2", "de", true, "", false, "tab=2"},
		{"false-positive substring", "clang=c", "", false, "", false, "clang=c"},
		{"empty version stripped, no override", "version=", "", false, "", false, ""},
		{"empty lang stripped, no override", "lang=", "", false, "", false, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lang, hasLang, ver, hasVer, rest := extractLaneOverrides(tt.query)
			if lang != tt.wantLang || hasLang != tt.wantHasLang {
				t.Errorf("lang = %q,%v; want %q,%v", lang, hasLang, tt.wantLang, tt.wantHasLang)
			}
			if ver != tt.wantVer || hasVer != tt.wantHasVer {
				t.Errorf("version = %q,%v; want %q,%v", ver, hasVer, tt.wantVer, tt.wantHasVer)
			}
			if rest != tt.wantRest {
				t.Errorf("rest = %q; want %q", rest, tt.wantRest)
			}
		})
	}
}

func TestResolveInternalLink_LangOverride_RelativeLink(t *testing.T) {
	// The French translation of auth lives in the (de, "") lane.
	dePage := &engine.Page{
		PageIdentity: engine.PageIdentity{RelPermalink: "/docs/auth/", Permalink: "/de/docs/auth/"},
		PageI18n:     engine.PageI18n{Lang: "de"},
	}
	idx := &mockPageIndex{
		pages: map[laneEntry]*engine.Page{
			{"/docs/auth/", "de", ""}: dePage,
		},
	}
	currentPage := &engine.Page{
		PageIdentity: engine.PageIdentity{RelPermalink: "/docs/intro/"},
		PageI18n:     engine.PageI18n{Lang: "en", LangRelPath: "docs/intro.md"},
	}

	dest := ClassifyDest("./auth.md?lang=de")
	result := ResolveInternalLink(dest, currentPage, ctxOf(idx, langVersionResolver))

	if !result.Found {
		t.Fatal("?lang=de should resolve into the de lane")
	}
	if result.URL != "/de/docs/auth/" {
		t.Errorf("URL = %q, want /de/docs/auth/", result.URL)
	}
	if result.TargetPermalink != "/de/docs/auth/" {
		t.Errorf("TargetPermalink = %q, want /de/docs/auth/", result.TargetPermalink)
	}
}

func TestResolveInternalLink_LangOverride_ContentRoot(t *testing.T) {
	dePage := &engine.Page{
		PageIdentity: engine.PageIdentity{RelPermalink: "/docs/guide/api/", Permalink: "/de/docs/guide/api/"},
		PageI18n:     engine.PageI18n{Lang: "de"},
	}
	idx := &mockPageIndex{
		pages: map[laneEntry]*engine.Page{
			{"/docs/guide/api/", "de", ""}: dePage,
		},
	}
	currentPage := &engine.Page{
		PageIdentity:      engine.PageIdentity{RelPermalink: "/docs/intro/"},
		PageI18n:          engine.PageI18n{Lang: "en", LangRelPath: "docs/intro.md"},
		PageRelationships: engine.PageRelationships{Collection: makeCollection("docs", false, "")},
	}

	dest := ClassifyDest("/guide/api.md?lang=de")
	result := ResolveInternalLink(dest, currentPage, ctxOf(idx, langVersionResolver))

	if !result.Found {
		t.Fatal("content-root ?lang=de should resolve into the de lane")
	}
	if result.URL != "/de/docs/guide/api/" {
		t.Errorf("URL = %q, want /de/docs/guide/api/", result.URL)
	}
}

func TestResolveInternalLink_VersionOverride_SameCollection(t *testing.T) {
	docs := makeCollection("docs", true, "v2")

	// v1 target (older version) registered in the v1 lane.
	target := &engine.Page{
		PageIdentity:      engine.PageIdentity{RelPermalink: "/docs/guide/api/", Permalink: "/docs/v1/guide/api/"},
		PageI18n:          engine.PageI18n{Lang: "en"},
		PageVersioning:    engine.PageVersioning{Version: "v1"},
		PageRelationships: engine.PageRelationships{Collection: docs},
	}
	idx := &mockPageIndex{
		pages: map[laneEntry]*engine.Page{
			{"/docs/guide/api/", "en", "v1"}: target,
		},
	}
	// Source is on the latest version (v2) and explicitly links into v1.
	currentPage := &engine.Page{
		PageIdentity:      engine.PageIdentity{RelPermalink: "/docs/guide/auth/"},
		PageI18n:          engine.PageI18n{Lang: "en", LangRelPath: "docs/guide/auth.md"},
		PageVersioning:    engine.PageVersioning{Version: "v2", VersionRelPath: "guide/auth"},
		PageRelationships: engine.PageRelationships{Collection: docs},
	}
	ctx := ResolveContext{
		PageIndex:   idx,
		URLResolver: versionInPath,
		Collections: map[string]*engine.Collection{"docs": docs},
	}

	dest := ClassifyDest("./api.md?version=v1")
	result := ResolveInternalLink(dest, currentPage, ctx)

	if !result.Found {
		t.Fatal("?version=v1 should resolve into the v1 lane")
	}
	if result.URL != "/docs/v1/guide/api/" {
		t.Errorf("URL = %q, want /docs/v1/guide/api/", result.URL)
	}
}

func TestResolveInternalLink_BothOverrides(t *testing.T) {
	docs := makeCollection("docs", true, "v2")

	target := &engine.Page{
		PageIdentity:      engine.PageIdentity{RelPermalink: "/docs/guide/api/", Permalink: "/de/docs/v1/guide/api/"},
		PageI18n:          engine.PageI18n{Lang: "de"},
		PageVersioning:    engine.PageVersioning{Version: "v1"},
		PageRelationships: engine.PageRelationships{Collection: docs},
	}
	idx := &mockPageIndex{
		pages: map[laneEntry]*engine.Page{
			{"/docs/guide/api/", "de", "v1"}: target,
		},
	}
	currentPage := &engine.Page{
		PageIdentity:      engine.PageIdentity{RelPermalink: "/docs/guide/auth/"},
		PageI18n:          engine.PageI18n{Lang: "en", LangRelPath: "docs/guide/auth.md"},
		PageVersioning:    engine.PageVersioning{Version: "v2", VersionRelPath: "guide/auth"},
		PageRelationships: engine.PageRelationships{Collection: docs},
	}
	ctx := ResolveContext{
		PageIndex:   idx,
		URLResolver: langVersionResolver,
		Collections: map[string]*engine.Collection{"docs": docs},
	}

	dest := ClassifyDest("./api.md?lang=de&version=v1")
	result := ResolveInternalLink(dest, currentPage, ctx)

	if !result.Found {
		t.Fatal("?lang=de&version=v1 should resolve into the (de, v1) lane")
	}
	if result.URL != "/de/docs/v1/guide/api/" {
		t.Errorf("URL = %q, want /de/docs/v1/guide/api/", result.URL)
	}
}

func TestResolveInternalLink_OverridePreservesOtherQuery(t *testing.T) {
	dePage := &engine.Page{
		PageIdentity: engine.PageIdentity{RelPermalink: "/docs/auth/", Permalink: "/de/docs/auth/"},
		PageI18n:     engine.PageI18n{Lang: "de"},
	}
	idx := &mockPageIndex{
		pages: map[laneEntry]*engine.Page{
			{"/docs/auth/", "de", ""}: dePage,
		},
	}
	currentPage := &engine.Page{
		PageIdentity: engine.PageIdentity{RelPermalink: "/docs/intro/"},
		PageI18n:     engine.PageI18n{Lang: "en", LangRelPath: "docs/intro.md"},
	}

	dest := ClassifyDest("./auth.md?lang=de&highlight=x")
	result := ResolveInternalLink(dest, currentPage, ctxOf(idx, langVersionResolver))

	if !result.Found {
		t.Fatal("override link should resolve")
	}
	if result.URL != "/de/docs/auth/?highlight=x" {
		t.Errorf("URL = %q, want /de/docs/auth/?highlight=x (lang stripped, highlight preserved)", result.URL)
	}
}

func TestResolveInternalLink_LangOverride_NotFound(t *testing.T) {
	// Only an en page exists; ?lang=ja targets a lane that has nothing.
	enPage := &engine.Page{
		PageIdentity: engine.PageIdentity{RelPermalink: "/docs/auth/", Permalink: "/docs/auth/"},
		PageI18n:     engine.PageI18n{Lang: "en"},
	}
	idx := &mockPageIndex{
		pages: map[laneEntry]*engine.Page{
			{"/docs/auth/", "en", ""}: enPage,
		},
	}
	currentPage := &engine.Page{
		PageIdentity: engine.PageIdentity{RelPermalink: "/docs/intro/"},
		PageI18n:     engine.PageI18n{Lang: "en", LangRelPath: "docs/intro.md"},
	}

	dest := ClassifyDest("./auth.md?lang=ja")
	result := ResolveInternalLink(dest, currentPage, ctxOf(idx, langVersionResolver))

	if result.Found {
		t.Error("?lang=ja with no ja lane must not resolve (broken target)")
	}
}

func TestResolveInternalLink_VersionOverride_NotFound(t *testing.T) {
	docs := makeCollection("docs", true, "v2")
	target := &engine.Page{
		PageIdentity:      engine.PageIdentity{RelPermalink: "/docs/guide/api/", Permalink: "/docs/v1/guide/api/"},
		PageI18n:          engine.PageI18n{Lang: "en"},
		PageVersioning:    engine.PageVersioning{Version: "v1"},
		PageRelationships: engine.PageRelationships{Collection: docs},
	}
	idx := &mockPageIndex{
		pages: map[laneEntry]*engine.Page{
			{"/docs/guide/api/", "en", "v1"}: target,
		},
	}
	currentPage := &engine.Page{
		PageIdentity:      engine.PageIdentity{RelPermalink: "/docs/guide/auth/"},
		PageI18n:          engine.PageI18n{Lang: "en", LangRelPath: "docs/guide/auth.md"},
		PageVersioning:    engine.PageVersioning{Version: "v1", VersionRelPath: "guide/auth"},
		PageRelationships: engine.PageRelationships{Collection: docs},
	}
	ctx := ResolveContext{
		PageIndex:   idx,
		URLResolver: versionInPath,
		Collections: map[string]*engine.Collection{"docs": docs},
	}

	dest := ClassifyDest("./api.md?version=v99")
	result := ResolveInternalLink(dest, currentPage, ctx)

	if result.Found {
		t.Error("?version=v99 with no such lane must not resolve (broken target)")
	}
}

func TestResolveInternalLink_EmptyVersionKey(t *testing.T) {
	docs := makeCollection("docs", true, "v2")
	target := &engine.Page{
		PageIdentity:      engine.PageIdentity{RelPermalink: "/docs/guide/api/", Permalink: "/docs/v1/guide/api/"},
		PageI18n:          engine.PageI18n{Lang: "en"},
		PageVersioning:    engine.PageVersioning{Version: "v1"},
		PageRelationships: engine.PageRelationships{Collection: docs},
	}
	idx := &mockPageIndex{
		pages: map[laneEntry]*engine.Page{
			{"/docs/guide/api/", "en", "v1"}: target,
		},
	}
	currentPage := &engine.Page{
		PageIdentity:      engine.PageIdentity{RelPermalink: "/docs/guide/auth/"},
		PageI18n:          engine.PageI18n{Lang: "en", LangRelPath: "docs/guide/auth.md"},
		PageVersioning:    engine.PageVersioning{Version: "v1", VersionRelPath: "guide/auth"},
		PageRelationships: engine.PageRelationships{Collection: docs},
	}
	ctx := ResolveContext{
		PageIndex:   idx,
		URLResolver: versionInPath,
		Collections: map[string]*engine.Collection{"docs": docs},
	}

	// Empty value is a no-op: behaves exactly like ./api.md (same v1 lane).
	dest := ClassifyDest("./api.md?version=")
	result := ResolveInternalLink(dest, currentPage, ctx)

	if !result.Found {
		t.Fatal("empty ?version= should be a no-op and resolve in the current (v1) lane")
	}
	if result.URL != "/docs/v1/guide/api/" {
		t.Errorf("URL = %q, want /docs/v1/guide/api/", result.URL)
	}
}

func TestResolveInternalLink_OrdinaryQuery_Passthrough(t *testing.T) {
	// "clang=c" contains the "lang=" substring but is not the reserved key.
	target := &engine.Page{
		PageIdentity: engine.PageIdentity{RelPermalink: "/docs/auth/", Permalink: "/docs/auth/"},
		PageI18n:     engine.PageI18n{Lang: "en"},
	}
	idx := &mockPageIndex{
		pages: map[laneEntry]*engine.Page{
			{"/docs/auth/", "en", ""}: target,
		},
	}
	currentPage := &engine.Page{
		PageIdentity: engine.PageIdentity{RelPermalink: "/docs/intro/"},
		PageI18n:     engine.PageI18n{Lang: "en", LangRelPath: "docs/intro.md"},
	}

	dest := ClassifyDest("./auth.md?clang=c")
	result := ResolveInternalLink(dest, currentPage, ctxOf(idx, identityResolver))

	if !result.Found {
		t.Fatal("ordinary query link should resolve in the same lane")
	}
	if result.URL != "/docs/auth/?clang=c" {
		t.Errorf("URL = %q, want /docs/auth/?clang=c (query preserved verbatim)", result.URL)
	}
}
