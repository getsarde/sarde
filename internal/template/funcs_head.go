package template

import (
	htmltemplate "html/template"

	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/engine"
	"github.com/getsarde/sarde/internal/webfonts"
)

// siteConfigOf returns the resolved site config a route was built with, or
// nil (bootstrap passes and tests may render without one).
func siteConfigOf(rd *engine.RouteData) *config.SiteConfig {
	if rd == nil || rd.Site == nil {
		return nil
	}
	cfg, _ := rd.Site.Config.(*config.SiteConfig)
	return cfg
}

// webFontLinks is the <head> markup that loads the route's theme fonts from
// the service named by theme.web_fonts: preconnects plus one stylesheet.
// Empty when the setting is off or every font is bundled or not a web font.
// themeStyles writes it first, so starter and ejected themes, which all
// call themeStyles, get it without a template change.
func webFontLinks(rd *engine.RouteData) string {
	cfg := siteConfigOf(rd)
	if cfg == nil || cfg.Theme.WebFonts == "" || rd.Theme == nil {
		return ""
	}
	families := webfonts.Families(rd.Theme.Tokens, rd.Theme.DarkTokens)
	return webfonts.HeadLinks(cfg.Theme.WebFonts, families)
}

// fnSiteHeadTags renders the site-wide head.tags from sarde.yaml. Head.html
// calls it before the page's own tags, so a page can add to them.
func fnSiteHeadTags(data any) htmltemplate.HTML {
	rd, _ := data.(*engine.RouteData)
	cfg := siteConfigOf(rd)
	if cfg == nil {
		return ""
	}
	return fnRenderHeadTags(cfg.Head.Tags)
}
