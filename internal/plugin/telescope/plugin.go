// Package telescope provides a built-in plugin that adds a command-palette
// style quick-navigation modal, opened with Ctrl+/ (Cmd+/ on Mac). It ships a
// fuzzy page search (Fuse.js) over a build-time index of page metadata, plus
// pinned and recently visited pages persisted in localStorage.
package telescope

import (
	"embed"
	"encoding/json"
	"fmt"
	htmltemplate "html/template"
	"io/fs"
	"sync"

	"github.com/getsarde/sarde/internal/asset"
	"github.com/getsarde/sarde/internal/i18n"
	"github.com/getsarde/sarde/internal/plugin"
	"github.com/getsarde/sarde/internal/plugin/cfgutil"
)

//go:embed all:assets
var assetsFS embed.FS

//go:embed defaults.yaml
var defaultsData []byte

const (
	pluginPrefix  = "assets/plugins/telescope/"
	indexFileName = "telescope-pages.json"
)

var (
	assetsOnce sync.Once
	jsData     []byte // fuse.min.js + telescope.js concatenated
	cssData    []byte
	jsURL      string
	cssURL     string
	defaultCfg map[string]any
)

func ensureAssets() {
	assetsOnce.Do(func() {
		fuse, _ := fs.ReadFile(assetsFS, "assets/fuse.min.js")
		runtime, _ := fs.ReadFile(assetsFS, "assets/telescope.js")
		if len(fuse) > 0 && len(runtime) > 0 {
			jsData = make([]byte, 0, len(fuse)+len(runtime)+1)
			jsData = append(jsData, fuse...)
			jsData = append(jsData, '\n')
			jsData = append(jsData, runtime...)
		}
		cssData, _ = fs.ReadFile(assetsFS, "assets/telescope.css")

		if len(jsData) > 0 {
			jsURL = "/" + pluginPrefix + asset.FingerprintedName("telescope.js", asset.Fingerprint(jsData))
		}
		if len(cssData) > 0 {
			cssURL = "/" + pluginPrefix + asset.FingerprintedName("telescope.css", asset.Fingerprint(cssData))
		}
		defaultCfg, _ = cfgutil.DefaultsFromFields(defaultsData)
	})
}

// stringKeys lists the i18n keys resolved into the client config, minus the
// "telescope." prefix.
var stringKeys = []string{
	"trigger", "tab_search", "tab_recent", "placeholder", "filter_recent",
	"pin", "unpin", "clear", "clear_pinned", "clear_recent", "no_recent",
	"no_results", "close", "loading", "pinned_section", "recent_section",
	"results_section", "results_count", "showing_of", "dialog_opened",
	"recent_cleared", "pinned_cleared", "hint_navigate", "hint_select",
	"hint_pin", "hint_close",
}

// FieldNames returns the declared config field names, sorted, for config key
// validation warnings.
func FieldNames() []string {
	names, _ := cfgutil.FieldNames(defaultsData)
	return names
}

// New constructs the telescope plugin from its config map and StringTable.
// st may be nil; UI labels then degrade to their raw i18n keys and the client
// falls back to built-in English strings.
func New(cfg map[string]any, st *i18n.StringTable) *plugin.Plugin {
	ensureAssets()
	pcfg := cfgTelescopeConfig(cfg)

	// The inline config script is identical for every page of a language, so
	// it is serialized once per language. BeforeRender may run from parallel
	// render workers, hence the mutex.
	var mu sync.Mutex
	scriptByLang := make(map[string]htmltemplate.JS)

	inlineScript := func(lang string) htmltemplate.JS {
		mu.Lock()
		defer mu.Unlock()
		if s, ok := scriptByLang[lang]; ok {
			return s
		}
		strs := make(map[string]string, len(stringKeys))
		for _, key := range stringKeys {
			strs[key] = resolveString(st, lang, "telescope."+key)
		}
		if pcfg.placeholder != "" {
			strs["placeholder"] = pcfg.placeholder
		}
		jsonBytes, _ := json.Marshal(pcfg.clientConfig(strs))
		s := htmltemplate.JS(
			`window.__SARDE__=window.__SARDE__||{};` +
				`window.__SARDE__.pluginConfig=window.__SARDE__.pluginConfig||{};` +
				`window.__SARDE__.pluginConfig.telescope=` + string(jsonBytes) + `;`,
		)
		scriptByLang[lang] = s
		return s
	}

	return &plugin.Plugin{
		Name: "telescope",
		Hooks: plugin.PluginHooks{
			BeforeRender: func(ctx *plugin.BeforeRenderContext) error {
				rd := ctx.RouteData
				if cssURL != "" {
					rd.Styles = cfgutil.AppendUnique(rd.Styles, cssURL)
				}
				if jsURL != "" {
					rd.Scripts = cfgutil.AppendUnique(rd.Scripts, jsURL)
				}
				lang := ""
				if ctx.Page != nil {
					lang = ctx.Page.Lang
				}
				rd.InlineScripts = append(rd.InlineScripts, inlineScript(lang))
				return nil
			},
			BuildDone: func(ctx *plugin.BuildDoneContext) error {
				if len(jsData) > 0 {
					name := asset.FingerprintedName("telescope.js", asset.Fingerprint(jsData))
					if err := ctx.WriteFile(pluginPrefix+name, jsData); err != nil {
						return err
					}
				}
				if len(cssData) > 0 {
					name := asset.FingerprintedName("telescope.css", asset.Fingerprint(cssData))
					if err := ctx.WriteFile(pluginPrefix+name, cssData); err != nil {
						return err
					}
				}
				entries := buildIndex(ctx.Pages, pcfg.exclude, ctx.ResolveURL)
				data, err := json.Marshal(entries)
				if err != nil {
					return err
				}
				if err := ctx.WriteFile(indexFileName, data); err != nil {
					return err
				}
				ctx.Log(fmt.Sprintf("Built telescope index (%d pages)", len(entries)))
				return nil
			},
		},
	}
}

func resolveString(st *i18n.StringTable, lang, key string) string {
	if st == nil || lang == "" {
		return key
	}
	return st.Resolve(lang, key)
}
