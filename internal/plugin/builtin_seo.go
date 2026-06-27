package plugin

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/getsarde/sarde/internal/engine"
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

	canonical := baseURL + page.URL()

	// Version-aware canonical: non-latest versioned pages point canonical
	// at the latest version's URL to consolidate SEO signals on current docs.
	if page.Collection != nil && page.Collection.Config != nil {
		if vc := page.Collection.Config.Versioning; vc != nil && vc.Enabled {
			if page.Version != "" && page.Version != vc.LastVersion {
				if latestPeer := findLatestPeer(page, vc.LastVersion); latestPeer != nil {
					canonical = baseURL + latestPeer.URL()
				}
			}
		}
	}

	// Merge into any existing seo map rather than replacing it wholesale.
	// The social_cards plugin also runs in BeforeRender and may have already
	// injected og_image/twitter_image (a generated card URL); replacing the
	// map here would discard it. Merging makes the two plugins order-independent.
	seo, _ := page.Params["seo"].(map[string]any)
	if seo == nil {
		seo = make(map[string]any)
	}
	seo["og_title"] = page.Title
	seo["og_description"] = description
	seo["og_url"] = canonical
	seo["og_type"] = ogType
	seo["og_site_name"] = siteTitle
	seo["og_locale"] = normalizeLocale(page.Lang)
	seo["canonical"] = canonical

	if ctx.RouteData != nil && len(ctx.RouteData.Translations) > 0 {
		altLocales := make([]string, 0, len(ctx.RouteData.Translations))
		for _, t := range ctx.RouteData.Translations {
			altLocales = append(altLocales, normalizeLocale(t.Lang))
		}
		seo["og_locale_alternate"] = altLocales
	}

	if ogType == "article" {
		if !page.Date.IsZero() {
			seo["article_published_time"] = page.Date.Format(time.RFC3339)
		}
		if !page.Updated.IsZero() {
			seo["article_modified_time"] = page.Updated.Format(time.RFC3339)
		}
	}

	// Preserve an image already set by an earlier plugin (e.g. a generated
	// social card); only fall back to the page/default image otherwise.
	if existing, _ := seo["og_image"].(string); existing == "" {
		seo["og_image"] = ogImage
	}
	if _, ok := seo["og_image_alt"]; !ok {
		seo["og_image_alt"] = page.Title
	}

	// Twitter card.
	twitterCard := cfgString(cfg, "twitter_card", "summary_large_image")
	seo["twitter_card"] = twitterCard
	seo["twitter_title"] = page.Title
	seo["twitter_description"] = description
	if existing, _ := seo["twitter_image"].(string); existing == "" {
		seo["twitter_image"] = ogImage
	}
	if _, ok := seo["twitter_image_alt"]; !ok {
		seo["twitter_image_alt"] = page.Title
	}
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
	pageURL := baseURL + page.URL()
	siteName := ""
	if site != nil {
		siteName = site.Title
	}

	primary := primaryNode(page, pageURL, description, siteName)
	graph := []map[string]any{primary}

	if crumbs := breadcrumbListNode(route, site, baseURL); crumbs != nil {
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

func breadcrumbListNode(route *engine.RouteData, site *engine.SiteContext, baseURL string) map[string]any {
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
			// c.URL is already fully resolved (basePath + lang + version)
			// by resolveRouteAssets — just prepend the origin.
			item["item"] = baseURL + c.URL
		}
		items = append(items, item)
	}
	return map[string]any{
		"@type":           "BreadcrumbList",
		"itemListElement": items,
	}
}

func findLatestPeer(page *engine.Page, lastVersion string) *engine.Page {
	for _, peer := range page.VersionPeers {
		if peer.Version == lastVersion {
			return peer
		}
	}
	return nil
}

// normalizeLocale converts a BCP 47 language tag (e.g. "en", "pt-BR") to
// the OGP locale format (e.g. "en_US", "pt_BR"). For bare language codes
// without a region, the language is repeated as region in uppercase.
func normalizeLocale(lang string) string {
	if lang == "" {
		return "en_US"
	}
	parts := strings.SplitN(lang, "-", 2)
	if len(parts) == 2 {
		return parts[0] + "_" + strings.ToUpper(parts[1])
	}
	return parts[0] + "_" + strings.ToUpper(parts[0])
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

