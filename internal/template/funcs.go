package template

import (
	"fmt"
	htmltemplate "html/template"
	"strings"
	"time"

	"github.com/getsarde/sarde/internal/component"
	"github.com/getsarde/sarde/internal/content"
	"github.com/getsarde/sarde/internal/engine"
)

// buildFuncMap creates the template.FuncMap with all template functions.
// Closures capture Engine field addresses (e.g. &e.site) so they always see
// the current values across hot-reload rebuilds without needing re-registration.
// registry and partialCache are passed explicitly because they differ between
// the bootstrap pass (nil) and the final pass (populated).
func (e *Engine) buildFuncMap(
	registry *component.Registry,
	partialCache map[string]*htmltemplate.Template,
) htmltemplate.FuncMap {
	sitePtr := &e.site
	resolver := e.resolver
	urlResolverPtr := &e.urlResolver
	dataCache := &e.dataCache
	cachedCSS := e.cachedCSS
	cssURLPtr := &e.cssURL
	tokenCSSURLPtr := &e.tokenCSSURL
	assetResolverPtr := &e.assetResolver
	assetManifestPtr := &e.assetManifest
	imageProcessorPtr := &e.imageProcessor
	pluginFuncs := e.pluginFuncs
	currentLang := e.currentLangResolver()
	i18nStrings := e.i18nStrings
	pageIndexPtr := &e.pageIndex

	fm := htmltemplate.FuncMap{
		// ── Strings ──
		"upper":       strings.ToUpper,
		"lower":       strings.ToLower,
		"title":       fnTitle,
		"truncate":    fnTruncate,
		"slugify":     content.Slugify,
		"replace":     strings.ReplaceAll,
		"split":       strings.Split,
		"join":        fnJoin,
		"contains":    strings.Contains,
		"hasPrefix":   strings.HasPrefix,
		"hasSuffix":   strings.HasSuffix,
		"trim":        strings.TrimSpace,
		"markdownify": fnMarkdownify,
		"plainify":    fnPlainify,
		"safeHTML":    fnSafeHTML,
		"highlight":   fnHighlight,

		// ── Icons ──
		"icon": fnIcon,

		// ── Dates ──
		"dateFormat": fnDateFormat,
		"now":        func() time.Time { return time.Now() },

		// ── Math ──
		"add": fnAdd,
		"sub": fnSub,
		"mul": fnMul,
		"div": fnDiv,
		"mod": fnMod,

		// ── Logic ──
		"cond":    fnCond,
		"default": fnDefault,
		"isset":   fnIsset,

		// ── Collections ──
		"first":   fnFirst,
		"last":    fnLast,
		"after":   fnAfter,
		"shuffle": fnShuffle,
		"sortBy":  fnSort,
		"where":   fnWhere,
		"group":   fnGroup,
		"uniq":    fnUniq,
		"in":      fnIn,
		"seq":     fnSeq,

		"boolVal": func(p *bool, def bool) bool {
			if p != nil {
				return *p
			}
			return def
		},
		"boolParam": func(params map[string]any, key string, def bool) bool {
			if v, ok := params[key]; ok {
				if b, ok := v.(bool); ok {
					return b
				}
			}
			return def
		},
		"stringParam": func(params map[string]any, key string) string {
			if params == nil {
				return ""
			}
			if v, ok := params[key]; ok {
				if s, ok := v.(string); ok {
					return s
				}
			}
			return ""
		},
		"renderHeadTags": fnRenderHeadTags,
		"renderAttrs":    fnRenderAttrs,
		"urlize":         content.Slugify,

		// ── Debug ──
		"printf":  fmt.Sprintf,
		"jsonify": fnJsonify,
		"dump":    fnDump,

		// ── Data ──
		"data": func(name string) any {
			return loadDataFile(resolver.ProjectDir, name, dataCache)
		},

		// ── Theme Styles ──
		"themeStyles": func(data any) htmltemplate.HTML {
			rd, ok := data.(*engine.RouteData)
			if !ok || rd == nil {
				return ""
			}
			var sb strings.Builder
			// Token CSS: external file when URL is set, inline fallback otherwise.
			if tokenCSSURLPtr != nil && *tokenCSSURLPtr != "" {
				url := *tokenCSSURLPtr
				if r := *urlResolverPtr; r != nil {
					url = r.URL(url, "", "")
				}
				sb.WriteString(`<link rel="stylesheet" href="`)
				sb.WriteString(url)
				sb.WriteString("\">\n")
			} else if rd.Theme != nil && rd.Theme.StyleTag != "" {
				sb.WriteString(string(rd.Theme.StyleTag))
				sb.WriteByte('\n')
			}
			// Main CSS bundle: external <link> when URL is set, inline fallback otherwise.
			if cssURLPtr != nil && *cssURLPtr != "" {
				url := *cssURLPtr
				if r := *urlResolverPtr; r != nil {
					url = r.URL(url, "", "")
				}
				sb.WriteString(`<link rel="stylesheet" href="`)
				sb.WriteString(url)
				sb.WriteString(`">`)
			} else if cachedCSS != "" {
				sb.WriteString(wrapCSS(cachedCSS))
			}
			return htmltemplate.HTML(sb.String())
		},

		// ── Map construction ──
		"dict": func(pairs ...any) (map[string]any, error) {
			if len(pairs)%2 != 0 {
				return nil, fmt.Errorf("dict requires even number of args, got %d", len(pairs))
			}
			m := make(map[string]any, len(pairs)/2)
			for i := 0; i < len(pairs); i += 2 {
				k, ok := pairs[i].(string)
				if !ok {
					return nil, fmt.Errorf("dict key must be string, got %T", pairs[i])
				}
				m[k] = pairs[i+1]
			}
			return m, nil
		},

		// ── Versioning ──
		"versionOf": fnVersionOf,

		// ── Type conversion ──
		"toString": func(v any) string { return fmt.Sprint(v) },
		"toInt": func(v any) int {
			f, ok := toFloat64(v)
			if !ok {
				return 0
			}
			return int(f)
		},

		// ── Language ──
		"lang": func(data any) string {
			rd, ok := data.(*engine.RouteData)
			if !ok || rd == nil {
				return ""
			}
			return rd.Lang
		},

		// ── Special: partial & component ──
		"partial": func(name string, data any) (htmltemplate.HTML, error) {
			if partialCache == nil {
				return "", fmt.Errorf("partial %q: partial cache not initialized", name)
			}
			tmpl := partialCache[name]
			if tmpl == nil {
				return "", fmt.Errorf("partial %q not found", name)
			}
			var buf strings.Builder
			if err := tmpl.Execute(&buf, data); err != nil {
				return "", fmt.Errorf("rendering partial %q: %w", name, err)
			}
			return htmltemplate.HTML(buf.String()), nil
		},
		"component": func(name string, data any) (htmltemplate.HTML, error) {
			if registry == nil {
				return "", nil
			}
			return registry.RenderComponent(name, data)
		},
	}

	// Default no-op for optional plugin template functions.
	// Plugins override these via ConfigSetup.AddTemplateFunc.
	fm["announcementBanner"] = func() htmltemplate.HTML { return "" }

	for k, v := range buildURLFuncs(urlResolverPtr, sitePtr) {
		fm[k] = v
	}
	for k, v := range buildAssetFuncs(assetResolverPtr, assetManifestPtr, imageProcessorPtr, urlResolverPtr) {
		fm[k] = v
	}
	for k, v := range buildNavFuncs(sitePtr) {
		fm[k] = v
	}
	for k, v := range buildI18nFuncs(i18nStrings, currentLang) {
		fm[k] = v
	}
	for k, v := range buildContentFuncs(pageIndexPtr, sitePtr) {
		fm[k] = v
	}

	// Merge plugin-provided template functions.
	for k, v := range pluginFuncs {
		fm[k] = v
	}

	return fm
}

// ── Date function implementations ──

// fnDateFormat is the language-less fallback; funcMapForLang overrides it per
// render with a locale-aware closure. It still resolves the "short"/"long"/
// "iso" presets (theme.date_format is stored raw since the locale work), just
// always in English.
func fnDateFormat(t time.Time, layout string) string {
	return localizedDateFormat(t, layout, "")
}
