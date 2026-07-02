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

	bundleMu      sync.RWMutex
	bundleCSS     []byte
	bundleJS      []byte
	bundleCSSURL  string
	bundleJSURL   string
	bundleCSSPath string
	bundleJSPath  string
	pkgDevMode    bool
)

var initOnce sync.Once
var initErr error

// Initialize loads the embedded plugin manifest and pre-computes bundles.
// Safe to call multiple times; only the first call does work.
func Initialize() error {
	initOnce.Do(func() {
		if err := yaml.Unmarshal(manifestData, &manifest); err != nil {
			initErr = fmt.Errorf("clientplugins: bad manifest.yaml: %w", err)
			return
		}
		computeBundles(assetsFS, "assets/")
		precomputeDefaults()
	})
	return initErr
}

// SetDevMode controls whether bundle filenames use content hashes.
// Must be called before Initialize or RecomputeFromDir.
func SetDevMode(dev bool) {
	pkgDevMode = dev
}

// RecomputeFromDir reloads and rebundles all plugin client assets from the
// given filesystem directory. For use by 'sarde dev --theme-dev' only.
func RecomputeFromDir(dir string) error {
	if _, err := os.Stat(dir); err != nil {
		return fmt.Errorf("clientplugins: cannot access asset dir %s: %w", dir, err)
	}
	computeBundles(os.DirFS(dir), "")
	return nil
}

func computeBundles(fsys fs.FS, pathPrefix string) {
	slugs := sortedSlugs()

	var cssBuilder, jsBuilder bytes.Buffer

	for _, slug := range slugs {
		entry := manifest.Plugins[slug]
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

	bundleMu.Lock()
	defer bundleMu.Unlock()

	bundleCSS = nil
	bundleJS = nil
	bundleCSSURL = ""
	bundleJSURL = ""
	bundleCSSPath = ""
	bundleJSPath = ""

	if rawCSS := cssBuilder.Bytes(); len(rawCSS) > 0 {
		bundleCSS = minifyCSS(rawCSS)
		bundleCSSPath = "assets/plugins/" + asset.FingerprintedName("plugins.css", asset.Fingerprint(bundleCSS))
		bundleCSSURL = "/" + bundleCSSPath
	}

	if rawJS := jsBuilder.Bytes(); len(rawJS) > 0 {
		bundleJS = minifyJS(rawJS)
		bundleJSPath = "assets/plugins/" + asset.FingerprintedName("plugins.js", asset.Fingerprint(bundleJS))
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

// RegisterAll registers a single "clientplugins" meta-plugin that injects the
// bundled CSS/JS on every page and per-plugin config scripts gated by inject_when.
func RegisterAll(mgr *plugin.Manager, enabled []string, configs map[string]map[string]any) {
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

	for slug, entry := range manifest.Plugins {
		if !enabledSet[slug] {
			continue
		}
		cfg := mergeConfig(pluginDefaults[slug], configs[slug])
		activePlugins = append(activePlugins, pluginCfg{slug, entry, cfg})
	}

	if len(activePlugins) == 0 {
		return
	}

	p := &plugin.Plugin{
		Name: "clientplugins",
		Hooks: plugin.PluginHooks{
			BeforeRender: func(ctx *plugin.BeforeRenderContext) error {
				rd := ctx.RouteData

				bundleMu.RLock()
				cssURL := bundleCSSURL
				jsURL := bundleJSURL
				bundleMu.RUnlock()

				if cssURL != "" {
					rd.Styles = cfgutil.AppendUnique(rd.Styles, cssURL)
				}
				if jsURL != "" {
					rd.ModuleScripts = cfgutil.AppendUnique(rd.ModuleScripts, jsURL)
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
				bundleMu.RLock()
				css, cssPath := bundleCSS, bundleCSSPath
				js, jsPath := bundleJS, bundleJSPath
				bundleMu.RUnlock()

				if len(css) > 0 {
					if err := ctx.WriteFile(cssPath, css); err != nil {
						return err
					}
				}
				if len(js) > 0 {
					if err := ctx.WriteFile(jsPath, js); err != nil {
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
