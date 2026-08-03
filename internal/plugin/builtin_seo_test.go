package plugin

import (
	"html/template"
	"strings"
	"testing"
	"time"

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
