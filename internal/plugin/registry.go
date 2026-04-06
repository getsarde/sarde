package plugin

// builtinRegistry maps plugin names to their constructor functions.
// Each constructor receives the plugin's config from site.yaml plugins.config.<name>.
var builtinRegistry = map[string]func(cfg map[string]any) *Plugin{
	"sitemap":      newSitemapPlugin,
	"robots":       newRobotsPlugin,
	"rss":          newRSSPlugin,
	"seo":          newSEOPlugin,
	"reading_time": newReadingTimePlugin,
	"search":       newSearchPlugin,
	"link_checker": newLinkCheckerPlugin,
	"redirects":    newRedirectsPlugin,
	"content_lint": newContentLintPlugin,
	"llms_txt":     newLlmsTxtPlugin,
}
