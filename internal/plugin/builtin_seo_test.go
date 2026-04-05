package plugin

import (
	"strings"
	"testing"
	"time"

	"github.com/coderoo-dev/coderoo/internal/engine"
)

func TestSEO_PopulatesParams(t *testing.T) {
	page := &engine.Page{
		Title:        "My Post",
		RelPermalink: "/blog/my-post/",
		Description:  "A great post",
		Date:         time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
		Collection:   &engine.Collection{Name: "blog"},
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

	// JSON-LD should be present.
	jsonLD, ok := seo["json_ld"].(string)
	if !ok || jsonLD == "" {
		t.Error("expected json_ld to be non-empty string")
	}
	if !strings.Contains(jsonLD, "Article") {
		t.Error("expected Article type in JSON-LD for collection pages")
	}
}

func TestSEO_StandalonePageType(t *testing.T) {
	page := &engine.Page{
		Title:        "About",
		RelPermalink: "/about/",
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

func TestSEO_DisableJSONLD(t *testing.T) {
	page := &engine.Page{
		Title:        "Test",
		RelPermalink: "/test/",
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
