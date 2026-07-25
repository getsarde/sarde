// Package clientplugins provides a declarative manifest-driven loader for
// client-side plugins. Instead of separate Go subpackages, all pure JS/CSS
// plugins are defined in manifest.yaml with inject_when rules, and their
// assets are embedded via go:embed.
package clientplugins

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"sort"
	"sync"

	"github.com/evanw/esbuild/pkg/api"
	"gopkg.in/yaml.v3"

	"github.com/getsarde/sarde/internal/asset"
	"github.com/getsarde/sarde/internal/devlog"
	"github.com/getsarde/sarde/internal/engine"
	"github.com/getsarde/sarde/internal/plugin"
	"github.com/getsarde/sarde/internal/plugin/cfgutil"
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
)

var initOnce sync.Once
var initErr error

// Initialize loads the embedded plugin manifest and per-plugin defaults.
// Safe to call multiple times; only the first call does work.
//
// Bundling deliberately does not happen here: which plugins belong in the
// bundle depends on the site's plugins.enabled, and Initialize runs during
// config resolution (see build.KnownPluginNames), before any config exists.
// RegisterAll builds the bundle instead, once the enabled set is known.
func Initialize() error {
	initOnce.Do(func() {
		if err := yaml.Unmarshal(manifestData, &manifest); err != nil {
			initErr = fmt.Errorf("clientplugins: bad manifest.yaml: %w", err)
			return
		}
		precomputeDefaults()
	})
	return initErr
}

// bundle holds the concatenated client assets for one resolved set of enabled
// plugins. It is a value rather than package state so that two builds with
// different configs in the same process cannot see each other's bundle.
type bundle struct {
	css, js         []byte
	cssPath, jsPath string
	cssURL, jsURL   string
}

// assetSource returns the filesystem to read plugin assets from. A non-empty
// dir (from 'sarde dev --theme-dev') wins over the embedded assets; an
// unreadable dir falls back to embedded rather than shipping nothing.
func assetSource(dir string) (fs.FS, string) {
	if dir == "" {
		return assetsFS, "assets/"
	}
	if _, err := os.Stat(dir); err != nil {
		devlog.Warn("build", "clientplugins: cannot access asset dir %s: %v", dir, err)
		return assetsFS, "assets/"
	}
	return os.DirFS(dir), ""
}

// buildBundle concatenates and minifies the assets of the given slugs. Slugs
// are sorted first so the bundle bytes, and therefore the fingerprint, are
// deterministic regardless of map iteration order.
func buildBundle(fsys fs.FS, pathPrefix string, slugs []string) bundle {
	sorted := append([]string(nil), slugs...)
	sort.Strings(sorted)

	var cssBuilder, jsBuilder bytes.Buffer

	for _, slug := range sorted {
		entry, ok := manifest.Plugins[slug]
		if !ok {
			continue
		}
		if entry.Assets.CSS != "" {
			data, err := fs.ReadFile(fsys, pathPrefix+entry.Assets.CSS)
			if err == nil && len(data) > 0 {
				cssBuilder.WriteString("/* " + slug + " */\n")
				cssBuilder.Write(data)
				cssBuilder.WriteByte('\n')
			}
		}
		if entry.Assets.JS != "" {
			data, err := fs.ReadFile(fsys, pathPrefix+entry.Assets.JS)
			if err == nil && len(data) > 0 {
				jsBuilder.WriteString("/* " + slug + " */\n;(function(){\n")
				jsBuilder.Write(data)
				jsBuilder.WriteString("\n})();\n")
			}
		}
	}

	var b bundle

	if rawCSS := cssBuilder.Bytes(); len(rawCSS) > 0 {
		b.css = minifyCSS(rawCSS)
		b.cssPath = "assets/plugins/" + asset.FingerprintedName("plugins.css", asset.Fingerprint(b.css))
		b.cssURL = "/" + b.cssPath
	}

	if rawJS := jsBuilder.Bytes(); len(rawJS) > 0 {
		b.js = minifyJS(rawJS)
		b.jsPath = "assets/plugins/" + asset.FingerprintedName("plugins.js", asset.Fingerprint(b.js))
		b.jsURL = "/" + b.jsPath
	}

	return b
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
		defaults, err := cfgutil.DefaultsFromFields(data)
		if err != nil {
			continue
		}
		pluginDefaults[slug] = defaults
	}
}

// RegisterAll registers a single "clientplugins" meta-plugin that injects the
// bundled CSS/JS on every page and per-plugin config scripts gated by inject_when.
//
// Only plugins named in enabled contribute to the bundle, so a site ships no
// code for plugins it has not turned on. assetDir overrides the embedded assets
// for 'sarde dev --theme-dev'; empty means use the embedded ones.
func RegisterAll(mgr *plugin.Manager, enabled []string, configs map[string]map[string]any, assetDir string) {
	enabledSet := make(map[string]bool, len(enabled))
	for _, name := range enabled {
		enabledSet[name] = true
	}

	type pluginCfg struct {
		slug   string
		entry  ManifestEntry
		config map[string]any
	}
	var activePlugins []pluginCfg
	var activeSlugs []string

	for slug, entry := range manifest.Plugins {
		if !enabledSet[slug] {
			continue
		}
		cfg := mergeConfig(pluginDefaults[slug], configs[slug])
		activePlugins = append(activePlugins, pluginCfg{slug, entry, cfg})
		activeSlugs = append(activeSlugs, slug)
	}

	if len(activePlugins) == 0 {
		return
	}

	fsys, pathPrefix := assetSource(assetDir)
	b := buildBundle(fsys, pathPrefix, activeSlugs)

	p := &plugin.Plugin{
		Name: "clientplugins",
		Hooks: plugin.PluginHooks{
			BeforeRender: func(ctx *plugin.BeforeRenderContext) error {
				rd := ctx.RouteData

				if b.cssURL != "" {
					rd.Styles = cfgutil.AppendUnique(rd.Styles, b.cssURL)
				}
				if b.jsURL != "" {
					rd.ModuleScripts = cfgutil.AppendUnique(rd.ModuleScripts, b.jsURL)
				}

				merged := make(map[string]any)
				for _, pc := range activePlugins {
					if len(pc.config) > 0 && shouldInject(pc.entry.InjectWhen, ctx.Page, ctx.RouteData) {
						merged[pc.slug] = pc.config
					}
				}
				if len(merged) > 0 {
					jsonBytes, _ := json.Marshal(merged)
					rd.InlineScripts = append(rd.InlineScripts, template.JS(
						`window.__SARDE__=window.__SARDE__||{};window.__SARDE__.pluginConfig=`+string(jsonBytes)+`;`,
					))
				}
				return nil
			},
			BuildDone: func(ctx *plugin.BuildDoneContext) error {
				if len(b.css) > 0 {
					if err := ctx.WriteFile(b.cssPath, b.css); err != nil {
						return err
					}
				}
				if len(b.js) > 0 {
					if err := ctx.WriteFile(b.jsPath, b.js); err != nil {
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

func shouldInject(rule string, page *engine.Page, rd *engine.RouteData) bool {
	return plugin.MatchesInjectRule(rule, page, rd)
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
