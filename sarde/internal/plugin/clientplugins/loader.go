// Package clientplugins provides a declarative manifest-driven loader for
// client-side plugins. Instead of separate Go subpackages, all pure JS/CSS
// plugins are defined in manifest.yaml with inject_when rules, and their
// assets are embedded via go:embed.
package clientplugins

import (
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"sort"

	"github.com/evanw/esbuild/pkg/api"
	"gopkg.in/yaml.v3"

	"github.com/frostybee/sarde/internal/engine"
	"github.com/frostybee/sarde/internal/plugin"
)

//go:embed manifest.yaml
var manifestData []byte

//go:embed defaults
var defaultsFS embed.FS

//go:embed all:assets
var assetsFS embed.FS

// ManifestEntry defines a single client-side plugin in the manifest.
type ManifestEntry struct {
	Description string `yaml:"description"`
	InjectWhen  string `yaml:"inject_when"`
	Assets      struct {
		CSS string `yaml:"css"`
		JS  string `yaml:"js"`
	} `yaml:"assets"`
	Module bool `yaml:"module"`
}

// Manifest holds all client plugin definitions.
type Manifest struct {
	Plugins map[string]ManifestEntry `yaml:"plugins"`
}

var (
	manifest       Manifest
	pluginDefaults map[string]map[string]any

	bundleCSS     []byte
	bundleJS      []byte
	bundleCSSURL  string
	bundleJSURL   string
	bundleCSSPath string
	bundleJSPath  string
)

func init() {
	if err := yaml.Unmarshal(manifestData, &manifest); err != nil {
		panic("clientplugins: bad manifest.yaml: " + err.Error())
	}
	precomputeBundles()
	precomputeDefaults()
}

func precomputeBundles() {
	slugs := sortedSlugs()

	var cssBuilder, jsBuilder bytes.Buffer

	for _, slug := range slugs {
		entry := manifest.Plugins[slug]
		if entry.Assets.CSS != "" {
			data, err := fs.ReadFile(assetsFS, "assets/"+entry.Assets.CSS)
			if err == nil && len(data) > 0 {
				cssBuilder.WriteString("/* " + slug + " */\n")
				cssBuilder.Write(data)
				cssBuilder.WriteByte('\n')
			}
		}
		if entry.Assets.JS != "" {
			data, err := fs.ReadFile(assetsFS, "assets/"+entry.Assets.JS)
			if err == nil && len(data) > 0 {
				jsBuilder.WriteString("/* " + slug + " */\n;(function(){\n")
				jsBuilder.Write(data)
				jsBuilder.WriteString("\n})();\n")
			}
		}
	}

	if rawCSS := cssBuilder.Bytes(); len(rawCSS) > 0 {
		bundleCSS = minifyCSS(rawCSS)
		hash := contentHash(bundleCSS)
		bundleCSSPath = "assets/plugins/plugins." + hash + ".css"
		bundleCSSURL = "/" + bundleCSSPath
	}

	if rawJS := jsBuilder.Bytes(); len(rawJS) > 0 {
		bundleJS = minifyJS(rawJS)
		hash := contentHash(bundleJS)
		bundleJSPath = "assets/plugins/plugins." + hash + ".js"
		bundleJSURL = "/" + bundleJSPath
	}
}

func sortedSlugs() []string {
	slugs := make([]string, 0, len(manifest.Plugins))
	for slug := range manifest.Plugins {
		slugs = append(slugs, slug)
	}
	sort.Strings(slugs)
	return slugs
}

func minifyCSS(data []byte) []byte {
	result := api.Transform(string(data), api.TransformOptions{
		Loader:            api.LoaderCSS,
		MinifyWhitespace:  true,
		MinifySyntax:      true,
		MinifyIdentifiers: true,
	})
	if len(result.Errors) > 0 {
		return data
	}
	return result.Code
}

func minifyJS(data []byte) []byte {
	result := api.Transform(string(data), api.TransformOptions{
		Loader:            api.LoaderJS,
		MinifyWhitespace:  true,
		MinifySyntax:      true,
		MinifyIdentifiers: true,
	})
	if len(result.Errors) > 0 {
		return data
	}
	return result.Code
}

func precomputeDefaults() {
	pluginDefaults = make(map[string]map[string]any, len(manifest.Plugins))
	for slug := range manifest.Plugins {
		data, err := fs.ReadFile(defaultsFS, "defaults/"+slug+".yaml")
		if err != nil {
			continue
		}
		var raw struct {
			Fields map[string]map[string]any `yaml:"fields"`
		}
		if err := yaml.Unmarshal(data, &raw); err != nil {
			continue
		}
		defaults := make(map[string]any, len(raw.Fields))
		for name, field := range raw.Fields {
			if def, ok := field["default"]; ok {
				defaults[name] = def
			}
		}
		pluginDefaults[slug] = defaults
	}
}

func contentHash(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h[:4])
}

// RegisterAll registers a single "clientplugins" meta-plugin that injects the
// bundled CSS/JS on every page and per-plugin config scripts gated by inject_when.
func RegisterAll(mgr *plugin.Manager, enabled []string, configs map[string]map[string]any) {
	enabledSet := make(map[string]bool, len(enabled))
	for _, name := range enabled {
		enabledSet[name] = true
	}

	type pluginCfg struct {
		slug         string
		entry        ManifestEntry
		configScript template.HTML
	}
	var activePlugins []pluginCfg

	for slug, entry := range manifest.Plugins {
		if !enabledSet[slug] {
			continue
		}
		cfg := mergeConfig(pluginDefaults[slug], configs[slug])
		var configScript template.HTML
		if len(cfg) > 0 {
			jsonBytes, _ := json.Marshal(cfg)
			configScript = template.HTML(fmt.Sprintf(
				`window.__pluginConfig=window.__pluginConfig||{};window.__pluginConfig[%q]=%s;`,
				slug, string(jsonBytes),
			))
		}
		activePlugins = append(activePlugins, pluginCfg{slug, entry, configScript})
	}

	if len(activePlugins) == 0 {
		return
	}

	p := &plugin.Plugin{
		Name: "clientplugins",
		Hooks: plugin.PluginHooks{
			BeforeRender: func(ctx *plugin.BeforeRenderContext) error {
				rd := ctx.RouteData

				if bundleCSSURL != "" {
					rd.Styles = appendUnique(rd.Styles, bundleCSSURL)
				}
				if bundleJSURL != "" {
					rd.ModuleScripts = appendUnique(rd.ModuleScripts, bundleJSURL)
				}

				for _, pc := range activePlugins {
					if pc.configScript != "" && shouldInject(pc.entry.InjectWhen, ctx.Page, ctx.RouteData) {
						rd.InlineScripts = append(rd.InlineScripts, pc.configScript)
					}
				}
				return nil
			},
			BuildDone: func(ctx *plugin.BuildDoneContext) error {
				if len(bundleCSS) > 0 {
					if err := ctx.WriteFile(bundleCSSPath, bundleCSS); err != nil {
						return err
					}
				}
				if len(bundleJS) > 0 {
					if err := ctx.WriteFile(bundleJSPath, bundleJS); err != nil {
						return err
					}
				}
				return nil
			},
		},
	}
	mgr.Register(p)
}

// PluginSlugs returns the slugs of all plugins defined in the manifest.
func PluginSlugs() []string {
	slugs := make([]string, 0, len(manifest.Plugins))
	for slug := range manifest.Plugins {
		slugs = append(slugs, slug)
	}
	return slugs
}

// Defaults returns the default config values for a plugin, or nil if unknown.
func Defaults(slug string) map[string]any {
	return pluginDefaults[slug]
}

func appendUnique(list []string, item string) []string {
	for _, s := range list {
		if s == item {
			return list
		}
	}
	return append(list, item)
}

func shouldInject(rule string, page *engine.Page, rd *engine.RouteData) bool {
	switch rule {
	case "always":
		return true
	case "has_sidebar":
		return engine.LayoutHasSidebar(rd.Layout)
	case "has_toc":
		return engine.LayoutHasTOC(rd.Layout) && len(page.Headings) > 0
	case "has_headings":
		return len(page.Headings) > 0
	case "has_code_blocks":
		return page.HasCodeBlocks
	case "has_images":
		return page.HasImages
	case "has_prev_next":
		return page.PrevPage != nil || page.NextPage != nil
	case "is_content_page":
		return page.Kind == engine.KindPage || page.Kind == engine.KindBundle
	case "has_updated":
		return !page.Updated.IsZero()
	}
	return false
}

func mergeConfig(defaults, userCfg map[string]any) map[string]any {
	merged := make(map[string]any)
	for k, v := range defaults {
		merged[k] = v
	}
	for k, v := range userCfg {
		merged[k] = v
	}
	return merged
}
