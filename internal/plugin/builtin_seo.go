package plugin

import (
	"encoding/json"
	"strings"

	"github.com/coderoo-dev/coderoo/internal/engine"
)

func newSEOPlugin(cfg map[string]any) *Plugin {
	return &Plugin{
		Name: "seo",
		Hooks: PluginHooks{
			BeforeRender: func(ctx *BeforeRenderContext) error {
				return seoBeforeRender(ctx, cfg)
			},
		},
	}
}

func seoBeforeRender(ctx *BeforeRenderContext, cfg map[string]any) error {
	page := ctx.Page
	if page == nil {
		return nil
	}

	// Initialize Params if nil.
	if page.Params == nil {
		page.Params = make(map[string]any)
	}

	twitterHandle := cfgString(cfg, "twitter_handle", "")
	defaultImage := cfgString(cfg, "default_image", "")
	enableJSONLD := cfgBool(cfg, "json_ld", true)
	autoDesc := cfgBool(cfg, "auto_description", true)

	baseURL := ""
	siteTitle := ""
	if ctx.Site != nil {
		baseURL = strings.TrimRight(ctx.Site.BaseURL, "/")
		siteTitle = ctx.Site.Title
	}

	// Build description.
	description := page.Description
	if description == "" && autoDesc {
		description = string(page.Summary)
	}

	// Build image URL.
	ogImage := page.Image
	if ogImage == "" {
		ogImage = defaultImage
	}
	if ogImage != "" && !strings.HasPrefix(ogImage, "http") {
		ogImage = baseURL + ogImage
	}

	// Determine OG type.
	ogType := "website"
	if page.Collection != nil {
		ogType = "article"
	}

	canonical := baseURL + page.RelPermalink

	seo := map[string]any{
		"og_title":       page.Title,
		"og_description": description,
		"og_url":         canonical,
		"og_type":        ogType,
		"og_image":       ogImage,
		"og_site_name":   siteTitle,
		"canonical":      canonical,
	}

	// Twitter card.
	twitterCard := cfgString(cfg, "twitter_card", "summary_large_image")
	seo["twitter_card"] = twitterCard
	seo["twitter_title"] = page.Title
	seo["twitter_description"] = description
	seo["twitter_image"] = ogImage
	if twitterHandle != "" {
		seo["twitter_site"] = twitterHandle
	}

	// JSON-LD structured data.
	if enableJSONLD {
		ld := buildJSONLD(page, baseURL, description)
		if ld != "" {
			seo["json_ld"] = ld
		}
	}

	page.Params["seo"] = seo
	return nil
}

func buildJSONLD(page *engine.Page, baseURL, description string) string {
	ld := map[string]any{
		"@context": "https://schema.org",
	}

	if page.Collection != nil {
		ld["@type"] = "Article"
		ld["headline"] = page.Title
		ld["url"] = baseURL + page.RelPermalink
		if description != "" {
			ld["description"] = description
		}
		if !page.Date.IsZero() {
			ld["datePublished"] = page.Date.Format("2006-01-02")
		}
		if !page.Updated.IsZero() {
			ld["dateModified"] = page.Updated.Format("2006-01-02")
		}
	} else {
		ld["@type"] = "WebPage"
		ld["name"] = page.Title
		ld["url"] = baseURL + page.RelPermalink
	}

	data, err := json.Marshal(ld)
	if err != nil {
		return ""
	}
	return string(data)
}

