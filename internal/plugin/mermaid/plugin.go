// Package mermaid provides a built-in plugin that ships Mermaid runtime
// assets and wires them into pages whose rendered HTML contains a mermaid
// diagram (class="mermaid"). Config key `plugins.config.mermaid.always: true`
// forces load on every page.
//
// The `assets/` directory is expected to contain the Mermaid distribution
// (mermaid.min.js, init.js). Files are placeholders by default — replace
// mermaid.min.js with the real Mermaid release before shipping.
package mermaid

import (
	"embed"
	"io/fs"
	"strings"

	"github.com/coderoo-dev/coderoo/internal/plugin"
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
	always := cfgBool(cfg, "always", false)
	if !always && !needsMermaid(string(ctx.Page.Content)) {
		return nil
	}
	rd := ctx.RouteData
	for _, s := range runtimeScripts {
		rd.Scripts = appendUnique(rd.Scripts, s)
	}
	return nil
}

func buildDone(ctx *plugin.BuildDoneContext) error {
	return fs.WalkDir(assetsFS, "assets", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, err := fs.ReadFile(assetsFS, path)
		if err != nil {
			return err
		}
		rel := strings.TrimPrefix(path, "assets/")
		return ctx.WriteFile(vendorPrefix+rel, data)
	})
}

func needsMermaid(content string) bool {
	return strings.Contains(content, `class="mermaid`)
}

func appendUnique(list []string, item string) []string {
	for _, existing := range list {
		if existing == item {
			return list
		}
	}
	return append(list, item)
}

func cfgBool(cfg map[string]any, key string, fallback bool) bool {
	if cfg == nil {
		return fallback
	}
	v, ok := cfg[key]
	if !ok {
		return fallback
	}
	b, ok := v.(bool)
	if !ok {
		return fallback
	}
	return b
}
