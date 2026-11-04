package plugin

import (
	"encoding/json"
	"html"
	"html/template"
	"strings"
	"time"

	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/engine"
	"github.com/getsarde/sarde/internal/plugin/cfgutil"
	"golang.org/x/text/language"
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

	twitterHandle := cfgutil.String(cfg, "twitter_handle", "")
	defaultImage := cfgutil.String(cfg, "default_image", "")
	enableJSONLD := cfgutil.Bool(cfg, "json_ld", true)
	autoDesc := cfgutil.Bool(cfg, "auto_description", true)

	baseURL := ctx.BaseURL()
	siteTitle := ""
	if ctx.Site != nil {
		siteTitle = ctx.Site.Title
	}

	// Build description: frontmatter, then the raw-markdown auto summary,
	// then plain text from the rendered HTML. The last tier covers bodies
	// that are entirely directive blocks, which the summary extractor
	// rightly skips. The rendered tier carries HTML entities, so the final
	// value is decoded exactly once; html/template re-escapes on output.
	description := page.Description
	if description == "" && autoDesc {
		description = string(page.Summary)
	}
	if description == "" && autoDesc {
		description = TrimTitlePrefix(RenderedTextFallback(page.Content, RenderedFallbackMaxChars), page.Title)
	}
	description = html.UnescapeString(description)

	// Write the computed fallback back onto the page so Head.html's
	// <meta name="description"> tag sees it too. Without this, pages
	// lacking a frontmatter description got og:description but no classic
	// meta description at all. Guarded so an explicit value is never
	// overwritten. The value is inferred, so PageTitle does not show it
	// under the title.
	if page.Description == "" && description != "" {
		page.Description = description
		page.DescriptionInferred = true
	}

	// Build image URL: the page's own image, else the plugin default.
	pageImage := seoImageURL(ctx, page, page.Image)
	ogImage := pageImage
	if ogImage == "" {
		ogImage = seoImageURL(ctx, nil, defaultImage)
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
	// A single-language site leaves page.Lang empty; the route language is
	// then the site language, the same value <html lang> uses. A fallback
	// copy sits under a translated folder but holds the default language's
	// text, so its locale names the language it is written in.
	lang := page.Lang
	if page.IsFallback && ctx.Site != nil && ctx.Site.DefaultLang != "" {
		lang = ctx.Site.DefaultLang
	}
	if lang == "" && ctx.RouteData != nil {
		lang = ctx.RouteData.Lang
	}
	locale := normalizeLocale(lang)
	seo["og_locale"] = locale
	seo["canonical"] = canonical

	// Translations lists the page itself first; the alternates name only the
	// other locales. The key is removed when there are none, since the seo
	// map outlives a single render.
	var altLocales []string
	if ctx.RouteData != nil {
		seen := map[string]bool{locale: true}
		for _, t := range ctx.RouteData.Translations {
			if l := normalizeLocale(t.Lang); !seen[l] {
				seen[l] = true
				altLocales = append(altLocales, l)
			}
		}
	}
	if len(altLocales) > 0 {
		seo["og_locale_alternate"] = altLocales
	} else {
		delete(seo, "og_locale_alternate")
	}

	// An inferred date is the file's mtime, which says nothing about when
	// the page was published and resets on every checkout, so only an
	// explicit date becomes a publish date. The key is removed otherwise,
	// since the seo map outlives a single render.
	if ogType == "article" {
		if page.DateExplicit && !page.Date.IsZero() {
			seo["article_published_time"] = page.Date.Format(time.RFC3339)
		} else {
			delete(seo, "article_published_time")
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
	twitterCard := cfgutil.String(cfg, "twitter_card", "summary_large_image")
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

	// JSON-LD structured data. Stored as template.JS, not string: inside a
	// <script> block html/template JSON-encodes a plain string a second time,
	// which wraps the whole payload in a quoted literal and voids it for
	// structured-data consumers. template.JS marks it as pre-built script
	// content, and buildJSONLD's output is json.Marshal-generated with no
	// user-controlled raw markup.
	if enableJSONLD {
		ld := buildJSONLD(ctx, page, baseURL, description, pageImage)
		if ld != "" {
			seo["json_ld"] = template.JS(ld)
		}
	}

	page.Params["seo"] = seo
	return nil
}

// buildJSONLD emits a schema.org @graph containing up to three entities:
// the primary node for the page (Article / CollectionPage / WebPage), a
// BreadcrumbList derived from route data, and a Course when the collection
// is tagged as a course (via Params["schema_type"] = "Course").
func buildJSONLD(ctx *BeforeRenderContext, page *engine.Page, baseURL, description, image string) string {
	pageURL := baseURL + page.URL()
	siteName := ""
	if ctx.Site != nil {
		siteName = ctx.Site.Title
	}

	primary := primaryNode(page, pageURL, description, siteName, image, siteAuthor(ctx.Site))
	graph := []map[string]any{primary}

	if crumbs := breadcrumbListNode(ctx, page); crumbs != nil {
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

// siteAuthor returns site.author, the author of pages that name none.
func siteAuthor(site *engine.SiteContext) string {
	if site == nil {
		return ""
	}
	if cfg, ok := site.Config.(*config.SiteConfig); ok && cfg != nil {
		return cfg.Site.Author
	}
	return ""
}

func primaryNode(page *engine.Page, pageURL, description, siteName, image, defaultAuthor string) map[string]any {
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
		if page.DateExplicit && !page.Date.IsZero() {
			node["datePublished"] = page.Date.Format("2006-01-02")
		}
		if !page.Updated.IsZero() {
			node["dateModified"] = page.Updated.Format("2006-01-02")
		}
		author, _ := page.Params["author"].(string)
		if author == "" {
			author = defaultAuthor
		}
		if author != "" {
			node["author"] = map[string]any{"@type": "Person", "name": author}
		}
		if image != "" {
			node["image"] = image
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

// breadcrumbListNode runs before resolveRouteAssets, so the crumbs still hold
// prefix-free paths. Each is resolved the way resolveRouteAssets resolves the
// visible breadcrumbs, with the page's language and version, then made
// absolute. URLResolver.URL is idempotent, so an already resolved crumb is
// safe too.
func breadcrumbListNode(ctx *BeforeRenderContext, page *engine.Page) map[string]any {
	route := ctx.RouteData
	if route == nil || len(route.Breadcrumbs) < 2 {
		return nil
	}
	lang, version := page.Lang, engine.ResolvePageVersion(page)
	items := make([]map[string]any, 0, len(route.Breadcrumbs))
	for i, c := range route.Breadcrumbs {
		item := map[string]any{
			"@type":    "ListItem",
			"position": i + 1,
			"name":     c.Label,
		}
		switch {
		case c.URL == "":
		case engine.IsAbsoluteURL(c.URL):
			item["item"] = c.URL
		default:
			item["item"] = ctx.AbsURL(c.URL, lang, version)
		}
		items = append(items, item)
	}
	return map[string]any{
		"@type":           "BreadcrumbList",
		"itemListElement": items,
	}
}

// seoImageURL returns the absolute URL of an image. A relative value is read
// from the page's folder, as a browser on the page would read it; with no
// page (the plugin's default_image) it is read from the site root. Like every
// image it gets the base path but never a language prefix: a page bundle's
// files are written once, at the language-free path.
func seoImageURL(ctx *BeforeRenderContext, page *engine.Page, v string) string {
	switch {
	case v == "" || engine.IsAbsoluteURL(v):
		return v
	case page == nil:
		return ctx.AbsURL("/"+strings.TrimPrefix(v, "/"), "", "")
	default:
		return ctx.AbsURL(engine.PageRelativeTarget(page, v))
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

// normalizeLocale converts a language tag ("en", "pt-BR", "zh-Hant") to the
// OGP locale format language_TERRITORY ("en_US", "pt_BR", "zh_TW"). A tag
// without a region gets its most likely one (CLDR likely subtags), so a bare
// code never becomes a made-up locale such as "en_EN". An empty tag gives
// "en_US", the OGP default.
func normalizeLocale(lang string) string {
	if lang == "" {
		return "en_US"
	}
	tag, err := language.Parse(lang)
	if err != nil {
		return strings.ReplaceAll(lang, "-", "_")
	}
	base, _ := tag.Base()
	region, conf := tag.Region()
	if conf == language.No {
		return base.String()
	}
	return base.String() + "_" + region.String()
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
