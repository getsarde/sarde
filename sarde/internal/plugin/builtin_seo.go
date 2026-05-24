package plugin

import (
	"encoding/json"
	"strings"

	"github.com/frostybee/sarde/internal/engine"
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
		ld := buildJSONLD(page, ctx.RouteData, ctx.Site, baseURL, description)
		if ld != "" {
			seo["json_ld"] = ld
		}
	}

	page.Params["seo"] = seo
	return nil
}

// buildJSONLD emits a schema.org @graph containing up to three entities:
// the primary node for the page (Article / CollectionPage / WebPage), a
// BreadcrumbList derived from route data, and a Course when the collection
// is tagged as a course (via Params["schema_type"] = "Course").
func buildJSONLD(page *engine.Page, route *engine.RouteData, site *engine.SiteContext, baseURL, description string) string {
	pageURL := baseURL + page.RelPermalink
	siteName := ""
	if site != nil {
		siteName = site.Title
	}

	primary := primaryNode(page, pageURL, description, siteName)
	graph := []map[string]any{primary}

	if crumbs := breadcrumbListNode(route, baseURL); crumbs != nil {
		graph = append(graph, crumbs)
	}
	if course := courseNode(page, pageURL, description); course != nil {
		graph = append(graph, course)
	}

	out := map[string]any{
		"@context": "https://schema.org",
		"@graph":   graph,
	}
	data, err := json.Marshal(out)
	if err != nil {
		return ""
	}
	return string(data)
}

func primaryNode(page *engine.Page, pageURL, description, siteName string) map[string]any {
	node := map[string]any{}
	switch {
	case page.Kind == engine.KindSection || page.Kind == engine.KindHome:
		node["@type"] = "CollectionPage"
		node["name"] = page.Title
		node["url"] = pageURL
		if siteName != "" {
			node["isPartOf"] = map[string]any{"@type": "WebSite", "name": siteName}
		}
	case page.Collection != nil:
		node["@type"] = "Article"
		node["headline"] = page.Title
		node["url"] = pageURL
		node["mainEntityOfPage"] = pageURL
		if !page.Date.IsZero() {
			node["datePublished"] = page.Date.Format("2006-01-02")
		}
		if !page.Updated.IsZero() {
			node["dateModified"] = page.Updated.Format("2006-01-02")
		}
		if author, ok := page.Params["author"].(string); ok && author != "" {
			node["author"] = map[string]any{"@type": "Person", "name": author}
		}
		if page.Image != "" {
			node["image"] = page.Image
		}
	default:
		node["@type"] = "WebPage"
		node["name"] = page.Title
		node["url"] = pageURL
	}
	if description != "" {
		node["description"] = description
	}
	return node
}

func breadcrumbListNode(route *engine.RouteData, baseURL string) map[string]any {
	if route == nil || len(route.Breadcrumbs) < 2 {
		return nil
	}
	items := make([]map[string]any, 0, len(route.Breadcrumbs))
	for i, c := range route.Breadcrumbs {
		item := map[string]any{
			"@type":    "ListItem",
			"position": i + 1,
			"name":     c.Label,
		}
		if c.URL != "" {
			item["item"] = baseURL + c.URL
		}
		items = append(items, item)
	}
	return map[string]any{
		"@type":           "BreadcrumbList",
		"itemListElement": items,
	}
}

func courseNode(page *engine.Page, pageURL, description string) map[string]any {
	// Only emit Course when the page (or its frontmatter params) opts in.
	t, _ := page.Params["schema_type"].(string)
	if !strings.EqualFold(t, "Course") {
		return nil
	}
	node := map[string]any{
		"@type": "Course",
		"name":  page.Title,
		"url":   pageURL,
	}
	if description != "" {
		node["description"] = description
	}
	if provider, ok := page.Params["provider"].(string); ok && provider != "" {
		node["provider"] = map[string]any{"@type": "Organization", "name": provider}
	}
	return node
}

