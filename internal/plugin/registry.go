package plugin

// builtinRegistry maps plugin names to their constructor functions.
// Each constructor receives the plugin's config from sarde.yaml plugins.config.<name>.
//
// Plugins that live in subpackages (to keep their embedded vendor assets
// isolated) cannot appear here because it would create an import cycle.
// They are registered from internal/build/builder.go via Manager.Register.
// BuiltinNames returns the names of all plugins registered in the built-in
// registry. It does not include subpackage plugins (katex, mermaid, etc.)
// or client-side plugins from the manifest.
func BuiltinNames() []string {
	names := make([]string, 0, len(builtinRegistry))
	for name := range builtinRegistry {
		names = append(names, name)
	}
	return names
}

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
