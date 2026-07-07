// Package mermaid provides a built-in plugin that ships Mermaid runtime
// assets and wires them into pages whose rendered HTML contains a mermaid
// diagram (class="sarde-mermaid"). Config key `plugins.config.mermaid.always: true`
// forces load on every page.
//
// The `assets/` directory is expected to contain the Mermaid distribution
// (mermaid.min.js, init.js). Files are placeholders by default — replace
// mermaid.min.js with the real Mermaid release before shipping.
package mermaid

import (
	"embed"
	"strings"

	"github.com/getsarde/sarde/internal/plugin"
	"github.com/getsarde/sarde/internal/plugin/cfgutil"
)

//go:embed all:assets
var assetsFS embed.FS

const vendorPrefix = "assets/vendor/mermaid/"

var runtimeScripts = []string{
	"/assets/vendor/mermaid/mermaid.min.js",
	"/assets/vendor/mermaid/init.js",
}

// New constructs the Mermaid plugin from its config map.
func New(cfg map[string]any) *plugin.Plugin {
	return &plugin.Plugin{
		Name: "mermaid",
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
	if !always && !needsMermaid(string(ctx.Page.Content)) {
		return nil
	}
	rd := ctx.RouteData
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

func needsMermaid(content string) bool {
	return strings.Contains(content, `class="sarde-mermaid`)
}
