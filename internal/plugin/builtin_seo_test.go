package plugin

import (
	"html/template"
	"strings"
	"testing"
	"time"

	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/engine"
)

func TestSEO_PopulatesParams(t *testing.T) {
	page := &engine.Page{
		PageIdentity:      engine.PageIdentity{Title: "My Post", RelPermalink: "/blog/my-post/", Date: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)},
		PageMeta:          engine.PageMeta{Description: "A great post"},
		PageRelationships: engine.PageRelationships{Collection: &engine.Collection{Name: "blog"}},
	}

	ctx := &BeforeRenderContext{
		Page:  page,
		Site:  &engine.SiteContext{BaseURL: "https://example.com", Title: "My Site"},
		store: NewStore(),
	}

	cfg := map[string]any{
		"twitter_handle": "@mysite",
		"default_image":  "/img/default.png",
	}
	err := seoBeforeRender(ctx, cfg)
	if err != nil {
		t.Fatalf("seoBeforeRender failed: %v", err)
	}

	seo, ok := page.Params["seo"].(map[string]any)
	if !ok {
		t.Fatal("expected page.Params[seo] to be a map")
	}

	if seo["og_title"] != "My Post" {
		t.Errorf("og_title = %v, want My Post", seo["og_title"])
	}
	if seo["og_description"] != "A great post" {
		t.Errorf("og_description = %v, want A great post", seo["og_description"])
	}
	if seo["og_url"] != "https://example.com/blog/my-post/" {
		t.Errorf("og_url = %v", seo["og_url"])
	}
	if seo["og_type"] != "article" {
		t.Errorf("og_type = %v, want article", seo["og_type"])
	}
	if seo["og_image"] != "https://example.com/img/default.png" {
		t.Errorf("og_image = %v", seo["og_image"])
	}
	if seo["twitter_site"] != "@mysite" {
		t.Errorf("twitter_site = %v", seo["twitter_site"])
	}
	if seo["canonical"] != "https://example.com/blog/my-post/" {
		t.Errorf("canonical = %v", seo["canonical"])
	}

	// JSON-LD should be present as template.JS so html/template does not
	// re-encode it inside the <script> block.
	ld, ok := seo["json_ld"].(template.JS)
	jsonLD := string(ld)
	if !ok || jsonLD == "" {
		t.Error("expected json_ld to be non-empty template.JS")
	}
	if !strings.Contains(jsonLD, "Article") {
		t.Error("expected Article type in JSON-LD for collection pages")
	}
}

func TestSEO_StandalonePageType(t *testing.T) {
	page := &engine.Page{
		PageIdentity: engine.PageIdentity{Title: "About", RelPermalink: "/about/"},
	}

	ctx := &BeforeRenderContext{
		Page:  page,
		Site:  &engine.SiteContext{BaseURL: "https://example.com"},
		store: NewStore(),
	}

	seoBeforeRender(ctx, nil)

	seo := page.Params["seo"].(map[string]any)
	if seo["og_type"] != "website" {
		t.Errorf("og_type = %v, want website for standalone pages", seo["og_type"])
	}
}

func TestSEO_CollectionPageType(t *testing.T) {
	page := &engine.Page{
		PageIdentity: engine.PageIdentity{Title: "Blog", RelPermalink: "/blog/", Kind: engine.KindSection},
	}
	ctx := &BeforeRenderContext{
		Page:  page,
		Site:  &engine.SiteContext{BaseURL: "https://example.com", Title: "Site"},
		store: NewStore(),
	}
	seoBeforeRender(ctx, nil)

	jsonLD := string(page.Params["seo"].(map[string]any)["json_ld"].(template.JS))
	if !strings.Contains(jsonLD, "CollectionPage") {
		t.Errorf("expected CollectionPage in JSON-LD, got: %s", jsonLD)
	}
}

func TestSEO_BreadcrumbList(t *testing.T) {
	page := &engine.Page{
		PageIdentity:      engine.PageIdentity{Title: "Post", RelPermalink: "/blog/post/"},
		PageRelationships: engine.PageRelationships{Collection: &engine.Collection{Name: "blog"}},
	}
	ctx := &BeforeRenderContext{
		Page: page,
		Site: &engine.SiteContext{BaseURL: "https://example.com"},
		RouteData: &engine.RouteData{
			RouteNav: engine.RouteNav{
				Breadcrumbs: []engine.BreadcrumbItem{
					{Label: "Home", URL: "/"},
					{Label: "Blog", URL: "/blog/"},
					{Label: "Post", URL: "/blog/post/", Current: true},
				},
			},
		},
		store: NewStore(),
	}
	seoBeforeRender(ctx, nil)

	jsonLD := string(page.Params["seo"].(map[string]any)["json_ld"].(template.JS))
	if !strings.Contains(jsonLD, "BreadcrumbList") {
		t.Error("expected BreadcrumbList in JSON-LD")
	}
	if !strings.Contains(jsonLD, `"position":1`) {
		t.Error("expected breadcrumb position 1")
	}
}

func TestSEO_CourseNode(t *testing.T) {
	page := &engine.Page{
		PageIdentity:      engine.PageIdentity{Title: "Intro to Go", RelPermalink: "/courses/intro-go/"},
		PageRelationships: engine.PageRelationships{Collection: &engine.Collection{Name: "courses"}},
		Params:            map[string]any{"schema_type": "Course", "provider": "Acme"},
	}
	ctx := &BeforeRenderContext{
		Page:  page,
		Site:  &engine.SiteContext{BaseURL: "https://example.com"},
		store: NewStore(),
	}
	seoBeforeRender(ctx, nil)

	jsonLD := string(page.Params["seo"].(map[string]any)["json_ld"].(template.JS))
	if !strings.Contains(jsonLD, `"@type":"Course"`) {
		t.Errorf("expected Course node, got: %s", jsonLD)
	}
	if !strings.Contains(jsonLD, "Acme") {
		t.Error("expected provider name")
	}
}

func TestSEO_PageImageUsedForOgImage(t *testing.T) {
	page := &engine.Page{
		PageIdentity:      engine.PageIdentity{Title: "Post With Image", RelPermalink: "/blog/post/"},
		PageMeta:          engine.PageMeta{Description: "A post", Image: "/img/custom-cover.png"},
		PageRelationships: engine.PageRelationships{Collection: &engine.Collection{Name: "blog"}},
	}

	ctx := &BeforeRenderContext{
		Page:  page,
		Site:  &engine.SiteContext{BaseURL: "https://example.com", Title: "My Site"},
		store: NewStore(),
	}

	err := seoBeforeRender(ctx, nil)
	if err != nil {
		t.Fatalf("seoBeforeRender failed: %v", err)
	}

	seo := page.Params["seo"].(map[string]any)
	if seo["og_image"] != "https://example.com/img/custom-cover.png" {
		t.Errorf("og_image = %v, want page.Image-based URL", seo["og_image"])
	}
}

func TestSEO_DefaultImageFallback(t *testing.T) {
	page := &engine.Page{
		PageIdentity:      engine.PageIdentity{Title: "Post No Image", RelPermalink: "/blog/post/"},
		PageMeta:          engine.PageMeta{Description: "A post"},
		PageRelationships: engine.PageRelationships{Collection: &engine.Collection{Name: "blog"}},
	}

	ctx := &BeforeRenderContext{
		Page:  page,
		Site:  &engine.SiteContext{BaseURL: "https://example.com", Title: "My Site"},
		store: NewStore(),
	}

	cfg := map[string]any{"default_image": "/img/fallback.png"}
	err := seoBeforeRender(ctx, cfg)
	if err != nil {
		t.Fatalf("seoBeforeRender failed: %v", err)
	}

	seo := page.Params["seo"].(map[string]any)
	if seo["og_image"] != "https://example.com/img/fallback.png" {
		t.Errorf("og_image = %v, want default_image fallback", seo["og_image"])
	}
}

func TestSEO_PreservesExistingOgImage(t *testing.T) {
	// Simulate social_cards having already run in BeforeRender and injected a
	// generated card URL. seoBeforeRender must merge, not clobber it.
	page := &engine.Page{
		PageIdentity:      engine.PageIdentity{Title: "Post", RelPermalink: "/blog/post/"},
		PageMeta:          engine.PageMeta{Description: "A post", Image: "/img/page-cover.png"},
		PageRelationships: engine.PageRelationships{Collection: &engine.Collection{Name: "blog"}},
		Params: map[string]any{
			"seo": map[string]any{
				"og_image":      "/og/generated-card.png",
				"twitter_image": "/og/generated-card.png",
			},
		},
	}

	ctx := &BeforeRenderContext{
		Page:  page,
		Site:  &engine.SiteContext{BaseURL: "https://example.com", Title: "My Site"},
		store: NewStore(),
	}

	cfg := map[string]any{"default_image": "/img/fallback.png"}
	if err := seoBeforeRender(ctx, cfg); err != nil {
		t.Fatalf("seoBeforeRender failed: %v", err)
	}

	seo := page.Params["seo"].(map[string]any)
	if seo["og_image"] != "/og/generated-card.png" {
		t.Errorf("og_image = %v, want preserved card URL /og/generated-card.png", seo["og_image"])
	}
	if seo["twitter_image"] != "/og/generated-card.png" {
		t.Errorf("twitter_image = %v, want preserved card URL", seo["twitter_image"])
	}
	// Other SEO fields should still be populated.
	if seo["og_title"] != "Post" {
		t.Errorf("og_title = %v, want Post", seo["og_title"])
	}
}

func TestSEO_DisableJSONLD(t *testing.T) {
	page := &engine.Page{
		PageIdentity: engine.PageIdentity{Title: "Test", RelPermalink: "/test/"},
	}

	ctx := &BeforeRenderContext{
		Page:  page,
		Site:  &engine.SiteContext{BaseURL: "https://example.com"},
		store: NewStore(),
	}

	cfg := map[string]any{"json_ld": false}
	seoBeforeRender(ctx, cfg)

	seo := page.Params["seo"].(map[string]any)
	if seo["json_ld"] != nil {
		t.Error("json_ld should be nil when disabled")
	}
}

func TestSEO_RenderedFallbackWhenSummaryEmpty(t *testing.T) {
	// A directive-only body leaves Description and Summary empty; the
	// description then falls back to plain text from the rendered HTML,
	// entity-decoded and with the leading title heading trimmed.
	content := `<h1>Welcome</h1><div class="card-grid"><a href="/docs/"><h3>Getting Started</h3><p>Install Sarde &amp; build</p></a></div>`
	page := &engine.Page{
		PageIdentity: engine.PageIdentity{Title: "Welcome", RelPermalink: "/"},
		PageContent:  engine.PageContent{Content: template.HTML(content)},
	}
	ctx := &BeforeRenderContext{
		Page:  page,
		Site:  &engine.SiteContext{BaseURL: "https://example.com", Title: "My Site"},
		store: NewStore(),
	}
	if err := seoBeforeRender(ctx, map[string]any{}); err != nil {
		t.Fatalf("seoBeforeRender failed: %v", err)
	}

	seo := page.Params["seo"].(map[string]any)
	desc, _ := seo["og_description"].(string)
	if desc != "Getting Started Install Sarde & build" {
		t.Errorf("og_description = %q", desc)
	}
	if seo["twitter_description"] != desc {
		t.Errorf("twitter_description = %v, want %q", seo["twitter_description"], desc)
	}
	if strings.Contains(desc, "<") || strings.Contains(desc, "&amp;") {
		t.Errorf("description must be tag-free and entity-decoded: %q", desc)
	}
}

func TestSEO_RenderedFallbackSkipsCodeAndMarksTheDescriptionInferred(t *testing.T) {
	run := func(content string) *engine.Page {
		page := &engine.Page{
			PageIdentity: engine.PageIdentity{Title: "Example", RelPermalink: "/example/"},
			PageContent:  engine.PageContent{Content: template.HTML(content)},
		}
		ctx := &BeforeRenderContext{
			Page:  page,
			Site:  &engine.SiteContext{BaseURL: "https://example.com"},
			store: NewStore(),
		}
		if err := seoBeforeRender(ctx, map[string]any{}); err != nil {
			t.Fatalf("seoBeforeRender failed: %v", err)
		}
		return page
	}

	// Only code and a diagram: no prose, so no description at all.
	codeOnly := run(`<figure class="frame" data-lang="go"><pre data-language="go"><div class="kz-code"><div class="kz-line">fmt.Println("hi")</div></div></pre></figure>` +
		`<div class="sarde-mermaid not-content">graph TD; A--&gt;B</div>`)
	if codeOnly.Description != "" {
		t.Errorf("Description = %q, want none for a code-only page", codeOnly.Description)
	}

	// Prose around code: the prose only, inferred, so PageTitle hides it.
	mixed := run(`<pre><code>npm install</code></pre><p>Run <code>sarde build</code> to publish.</p>`)
	if mixed.Description != "Run sarde build to publish." {
		t.Errorf("Description = %q", mixed.Description)
	}
	if !mixed.DescriptionInferred {
		t.Error("a description written back from the page body must be marked inferred")
	}
}

func TestSEO_RenderedFallbackRespectsAutoDescriptionOptOut(t *testing.T) {
	page := &engine.Page{
		PageIdentity: engine.PageIdentity{Title: "Welcome", RelPermalink: "/"},
		PageContent:  engine.PageContent{Content: template.HTML("<p>Some rendered prose.</p>")},
	}
	ctx := &BeforeRenderContext{
		Page:  page,
		Site:  &engine.SiteContext{BaseURL: "https://example.com"},
		store: NewStore(),
	}
	if err := seoBeforeRender(ctx, map[string]any{"auto_description": false}); err != nil {
		t.Fatalf("seoBeforeRender failed: %v", err)
	}

	seo := page.Params["seo"].(map[string]any)
	if desc, _ := seo["og_description"].(string); desc != "" {
		t.Errorf("auto_description: false must skip the rendered fallback, got %q", desc)
	}
}

// A site-absolute og:image gets the base path, never the page's language; a
// protocol-relative URL is used as written.
func TestSEO_OGImage_BasePath(t *testing.T) {
	resolver := &engine.URLResolver{
		BaseURL: "https://example.com", BasePath: "/sub/",
		I18nEnabled: true, DefaultLang: "en", Languages: map[string]bool{"en": true, "fr": true},
	}
	for image, want := range map[string]string{
		"/images/cover.png":       "https://example.com/sub/images/cover.png",
		"/sub/images/cover.png":   "https://example.com/sub/images/cover.png",
		"//cdn.example.com/a.png": "//cdn.example.com/a.png",
		"https://cdn.x/a.png":     "https://cdn.x/a.png",
		// A relative image of the page's bundle sits beside the French page;
		// one outside the bundle exists once, with no language prefix.
		"cover.jpg":     "https://example.com/sub/fr/blog/post/cover.jpg",
		"../shared.png": "https://example.com/sub/blog/shared.png",
	} {
		page := &engine.Page{
			PageIdentity:      engine.PageIdentity{Title: "Post", RelPermalink: "/blog/post/"},
			PageMeta:          engine.PageMeta{Description: "A post", Image: image},
			PageI18n:          engine.PageI18n{Lang: "fr"},
			PageRelationships: engine.PageRelationships{Collection: &engine.Collection{Name: "blog"}},
			Resources:         []engine.Resource{{Name: "cover.jpg", RelPermalink: "/blog/post/cover.jpg"}},
		}
		ctx := &BeforeRenderContext{
			Page:     page,
			Site:     &engine.SiteContext{BaseURL: "https://example.com", Title: "My Site"},
			Resolver: resolver,
			store:    NewStore(),
		}
		if err := seoBeforeRender(ctx, nil); err != nil {
			t.Fatalf("seoBeforeRender(%q): %v", image, err)
		}
		seo := page.Params["seo"].(map[string]any)
		if seo["og_image"] != want || seo["twitter_image"] != want {
			t.Errorf("image %q: og_image = %v, twitter_image = %v, want %q", image, seo["og_image"], seo["twitter_image"], want)
		}
		if ld := string(seo["json_ld"].(template.JS)); !strings.Contains(ld, `"image":"`+want+`"`) {
			t.Errorf("image %q: JSON-LD image is not %q: %s", image, want, ld)
		}
	}
}

// A relative default_image has no page to sit beside, so it is read from the
// site root.
func TestSEO_RelativeDefaultImageResolvesAgainstSiteRoot(t *testing.T) {
	page := &engine.Page{PageIdentity: engine.PageIdentity{Title: "Post", RelPermalink: "/blog/post/"}}
	ctx := &BeforeRenderContext{
		Page:     page,
		Site:     &engine.SiteContext{BaseURL: "https://example.com"},
		Resolver: &engine.URLResolver{BaseURL: "https://example.com", BasePath: "/sub/"},
		store:    NewStore(),
	}
	if err := seoBeforeRender(ctx, map[string]any{"default_image": "img/og.png"}); err != nil {
		t.Fatal(err)
	}
	if got := page.Params["seo"].(map[string]any)["og_image"]; got != "https://example.com/sub/img/og.png" {
		t.Errorf("og_image = %v, want https://example.com/sub/img/og.png", got)
	}
}

// The JSON-LD crumbs are built before the route data is resolved, so they
// get the base path, the page's language and its version here, matching the
// visible breadcrumbs.
func TestSEO_BreadcrumbList_ResolvesBasePathAndLanguage(t *testing.T) {
	page := &engine.Page{
		PageIdentity:      engine.PageIdentity{Title: "Guide", RelPermalink: "/docs/guide/"},
		PageI18n:          engine.PageI18n{Lang: "fr"},
		PageRelationships: engine.PageRelationships{Collection: &engine.Collection{Name: "docs"}},
	}
	ctx := &BeforeRenderContext{
		Page: page,
		Site: &engine.SiteContext{BaseURL: "https://example.com"},
		Resolver: &engine.URLResolver{
			BaseURL: "https://example.com", BasePath: "/sarde/",
			I18nEnabled: true, DefaultLang: "en", Languages: map[string]bool{"en": true, "fr": true},
		},
		RouteData: &engine.RouteData{RouteNav: engine.RouteNav{Breadcrumbs: []engine.BreadcrumbItem{
			{Label: "Docs", URL: "/docs/"},
			{Label: "Phantom"},
			{Label: "Guide", URL: "/docs/guide/", Current: true},
		}}},
		store: NewStore(),
	}
	if err := seoBeforeRender(ctx, nil); err != nil {
		t.Fatal(err)
	}
	ld := string(page.Params["seo"].(map[string]any)["json_ld"].(template.JS))
	for _, want := range []string{
		`"item":"https://example.com/sarde/fr/docs/"`,
		`"item":"https://example.com/sarde/fr/docs/guide/"`,
	} {
		if !strings.Contains(ld, want) {
			t.Errorf("JSON-LD missing %s: %s", want, ld)
		}
	}
	if strings.Contains(ld, `"item":"https://example.com/docs/`) || strings.Contains(ld, `"item":"https://example.com/sarde/"`) {
		t.Errorf("a crumb lost its prefix, or an empty crumb got a URL: %s", ld)
	}
}

// An inferred date is the file's mtime, so only an explicit date is
// published as the article's publish date; the modified date is unaffected.
func TestSEO_PublishDateOnlyWhenExplicit(t *testing.T) {
	for _, explicit := range []bool{false, true} {
		page := &engine.Page{
			PageIdentity: engine.PageIdentity{
				Title: "Post", RelPermalink: "/blog/post/",
				Date:    time.Date(2026, 10, 29, 0, 0, 0, 0, time.UTC),
				Updated: time.Date(2026, 10, 18, 0, 0, 0, 0, time.UTC),
			},
			PageMeta:          engine.PageMeta{DateExplicit: explicit},
			PageRelationships: engine.PageRelationships{Collection: &engine.Collection{Name: "blog"}},
		}
		ctx := &BeforeRenderContext{Page: page, Site: &engine.SiteContext{BaseURL: "https://example.com"}, store: NewStore()}
		if err := seoBeforeRender(ctx, nil); err != nil {
			t.Fatal(err)
		}
		seo := page.Params["seo"].(map[string]any)
		ld := string(seo["json_ld"].(template.JS))
		_, hasOG := seo["article_published_time"]
		hasLD := strings.Contains(ld, `"datePublished"`)
		if hasOG != explicit || hasLD != explicit {
			t.Errorf("explicit=%v: article_published_time present=%v, datePublished present=%v", explicit, hasOG, hasLD)
		}
		if !strings.Contains(ld, `"dateModified":"2026-10-18"`) || seo["article_modified_time"] == nil {
			t.Errorf("explicit=%v: modified date missing: %s", explicit, ld)
		}
	}
}

func TestNormalizeLocale(t *testing.T) {
	tests := []struct{ in, want string }{
		{"", "en_US"},
		{"en", "en_US"},
		{"fr", "fr_FR"},
		{"de", "de_DE"},
		{"ja", "ja_JP"},
		{"en-GB", "en_GB"},
		{"EN-gb", "en_GB"},
		{"pt-BR", "pt_BR"},
		{"pt_BR", "pt_BR"},
		{"zh-Hant", "zh_TW"},
	}
	for _, tt := range tests {
		if got := normalizeLocale(tt.in); got != tt.want {
			t.Errorf("normalizeLocale(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// seoLocaleParams runs the SEO plugin on page with route data rd and returns
// the page's seo params.
func seoLocaleParams(t *testing.T, page *engine.Page, rd *engine.RouteData) map[string]any {
	t.Helper()
	ctx := &BeforeRenderContext{
		Page:      page,
		RouteData: rd,
		Site:      &engine.SiteContext{BaseURL: "https://example.com", Title: "My Site"},
		store:     NewStore(),
	}
	if err := seoBeforeRender(ctx, nil); err != nil {
		t.Fatalf("seoBeforeRender: %v", err)
	}
	seo, _ := page.Params["seo"].(map[string]any)
	return seo
}

func TestSEO_OGLocaleAlternate_SkipsOwnLocale(t *testing.T) {
	page := &engine.Page{
		PageIdentity: engine.PageIdentity{Title: "Guide", RelPermalink: "/docs/guide/"},
		PageI18n:     engine.PageI18n{Lang: "fr"},
	}
	// Translations lists the page itself first, as buildTranslationLinks does.
	rd := &engine.RouteData{}
	rd.Translations = []engine.TranslationLink{{Lang: "fr"}, {Lang: "en"}}
	seo := seoLocaleParams(t, page, rd)
	if seo["og_locale"] != "fr_FR" {
		t.Errorf("og_locale = %v, want fr_FR", seo["og_locale"])
	}
	alt, _ := seo["og_locale_alternate"].([]string)
	if len(alt) != 1 || alt[0] != "en_US" {
		t.Errorf("og_locale_alternate = %v, want [en_US]", seo["og_locale_alternate"])
	}

	// Rendered again without translations, the earlier list must not linger.
	seo = seoLocaleParams(t, page, &engine.RouteData{})
	if v, ok := seo["og_locale_alternate"]; ok {
		t.Errorf("og_locale_alternate = %v, want no key without translations", v)
	}
}

func TestSEO_OGLocale_FallsBackToRouteLang(t *testing.T) {
	// A single-language site leaves page.Lang empty; the route language is
	// the site language.
	page := &engine.Page{PageIdentity: engine.PageIdentity{Title: "Accueil", RelPermalink: "/"}}
	rd := &engine.RouteData{}
	rd.Lang = "fr"
	if got := seoLocaleParams(t, page, rd)["og_locale"]; got != "fr_FR" {
		t.Errorf("og_locale = %v, want fr_FR", got)
	}
}

// A fallback copy under /fr/ holds the default language's text, so its
// og:locale names that language, not the folder's.
func TestSEO_OGLocale_FallbackCopyNamesItsContentLanguage(t *testing.T) {
	page := &engine.Page{
		PageIdentity: engine.PageIdentity{Title: "Guide", RelPermalink: "/docs/guide/"},
		PageI18n:     engine.PageI18n{Lang: "fr", IsFallback: true},
	}
	ctx := &BeforeRenderContext{
		Page:  page,
		Site:  &engine.SiteContext{BaseURL: "https://example.com", DefaultLang: "en"},
		store: NewStore(),
	}
	if err := seoBeforeRender(ctx, nil); err != nil {
		t.Fatal(err)
	}
	if got := page.Params["seo"].(map[string]any)["og_locale"]; got != "en_US" {
		t.Errorf("og_locale = %v, want en_US", got)
	}
}

// An article without its own author names site.author in its JSON-LD.
func TestSEO_SiteAuthorFallback(t *testing.T) {
	cfg := config.Defaults()
	cfg.Site.Author = "Field Notes Team"
	page := &engine.Page{
		PageIdentity:      engine.PageIdentity{Title: "Post", RelPermalink: "/blog/post/"},
		PageRelationships: engine.PageRelationships{Collection: &engine.Collection{Name: "blog"}},
	}
	ctx := &BeforeRenderContext{Page: page, Site: &engine.SiteContext{BaseURL: "https://example.com", Config: cfg}, store: NewStore()}
	if err := seoBeforeRender(ctx, nil); err != nil {
		t.Fatal(err)
	}
	ld := string(page.Params["seo"].(map[string]any)["json_ld"].(template.JS))
	if !strings.Contains(ld, `"author":{"@type":"Person","name":"Field Notes Team"}`) {
		t.Errorf("JSON-LD author missing: %s", ld)
	}
}
