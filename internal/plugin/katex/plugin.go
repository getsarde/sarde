// Package katex provides a built-in plugin that ships KaTeX runtime assets
// and wires them into any page whose rendered HTML contains math markup.
//
// The plugin copies the embedded vendor files (JS, CSS, fonts, init script)
// under dist/assets/vendor/katex/ during BuildDone, and appends the relevant
// <script>/<link> entries to RouteData.Scripts and RouteData.Styles during
// BeforeRender — gated on a per-page detection that scans for class="math".
// Config key `plugins.config.katex.always: true` forces the assets onto every
// page regardless of detection.
//
// The `assets/` directory is expected to contain the KaTeX distribution
// (katex.min.js, katex.min.css, auto-render.min.js, init.js, fonts/*.woff2).
// Files are placeholders by default — replace them with the real KaTeX
// release before shipping.
package katex

import (
	"embed"
	"strings"

	"github.com/getsarde/sarde/internal/plugin"
	"github.com/getsarde/sarde/internal/plugin/cfgutil"
)

//go:embed all:assets
var assetsFS embed.FS

const (
	vendorPrefix  = "assets/vendor/katex/"
	stylesheetURL = "/assets/vendor/katex/katex.min.css"
)

var runtimeScripts = []string{
	"/assets/vendor/katex/katex.min.js",
	"/assets/vendor/katex/auto-render.min.js",
	"/assets/vendor/katex/init.js",
}

// New constructs the KaTeX plugin from its config map.
func New(cfg map[string]any) *plugin.Plugin {
	return &plugin.Plugin{
		Name: "katex",
		Hooks: plugin.PluginHooks{
			BeforeRender: func(ctx *plugin.BeforeRenderContext) error {
				return beforeRender(ctx, cfg)
			},
			BuildDone: func(ctx *plugin.BuildDoneContext) error {
				return buildDone(ctx)
			},
		},
	}
}

func beforeRender(ctx *plugin.BeforeRenderContext, cfg map[string]any) error {
	always := cfgutil.Bool(cfg, "always", false)
	if !always && !needsKatex(string(ctx.Page.Content)) {
		return nil
	}
	rd := ctx.RouteData
	rd.Styles = cfgutil.AppendUnique(rd.Styles, stylesheetURL)
	for _, s := range runtimeScripts {
		rd.Scripts = cfgutil.AppendUnique(rd.Scripts, s)
	}
	return nil
}

func buildDone(ctx *plugin.BuildDoneContext) error {
	// The embedded vendor tree never changes between rebuilds of one builder;
	// the first build of a session is always a full Build() and incremental
	// rebuilds never prune output, so the files are already on disk.
	if ctx.Incremental {
		return nil
	}
	return plugin.WriteFSTree(ctx, assetsFS, "assets", vendorPrefix)
}

func needsKatex(content string) bool {
	return strings.Contains(content, `class="sarde-math`)
}
