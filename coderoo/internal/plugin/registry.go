package plugin

// builtinRegistry maps plugin names to their constructor functions.
// Each constructor receives the plugin's config from site.yaml plugins.config.<name>.
//
// Plugins that live in subpackages (to keep their embedded vendor assets
// isolated) cannot appear here because it would create an import cycle.
// They are registered from internal/build/builder.go via Manager.Register.
var builtinRegistry = map[string]func(cfg map[string]any) *Plugin{
	"sitemap":      newSitemapPlugin,
	"robots":       newRobotsPlugin,
	"rss":          newRSSPlugin,
	"atom":         newAtomPlugin,
	"seo":          newSEOPlugin,
	"search":       newSearchPlugin,
	"link_validator": newLinkValidatorPlugin,
	"redirects":    newRedirectsPlugin,
	"content_lint": newContentLintPlugin,
	"llms_txt":     newLlmsTxtPlugin,
}
