package template

import (
	"fmt"
	"html"
	htmltemplate "html/template"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/frostybee/sarde/internal/asset"
	"github.com/frostybee/sarde/internal/component"
	"github.com/frostybee/sarde/internal/content"
	"github.com/frostybee/sarde/internal/engine"
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

		// ── URLs ──
		"absURL": func(relPath string) string {
			if r := *urlResolverPtr; r != nil {
				return r.AbsURL(relPath, "", "")
			}
			s := *sitePtr
			if s == nil || s.BaseURL == "" {
				return relPath
			}
			base := strings.TrimRight(s.BaseURL, "/")
			if strings.HasPrefix(relPath, "/") {
				return base + relPath
			}
			return base + "/" + relPath
		},
		"relURL": func(relPath string) string {
			if strings.Contains(relPath, "://") {
				return relPath
			}
			if r := *urlResolverPtr; r != nil {
				return r.URL(relPath, "", "")
			}
			return relPath
		},
		"editURL": func(base, relPath string) string {
			if base == "" || relPath == "" {
				return ""
			}
			return strings.TrimRight(base, "/") + "/" + strings.TrimLeft(filepath.ToSlash(relPath), "/")
		},
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
		"renderHeadTags": func(v any) htmltemplate.HTML {
			tags, ok := v.([]engine.HeadTag)
			if !ok || len(tags) == 0 {
				return ""
			}
			var sb strings.Builder
			for _, h := range tags {
				if !engine.AllowedHeadTags[h.Tag] {
					continue
				}
				sb.WriteString("<")
				sb.WriteString(h.Tag)
				keys := make([]string, 0, len(h.Attrs))
				for k := range h.Attrs {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				for _, k := range keys {
					sb.WriteString(" ")
					sb.WriteString(html.EscapeString(k))
					sb.WriteString(`="`)
					sb.WriteString(html.EscapeString(h.Attrs[k]))
					sb.WriteString(`"`)
				}
				if h.Content != "" {
					sb.WriteString(">")
					sb.WriteString(html.EscapeString(h.Content))
					sb.WriteString("</")
					sb.WriteString(h.Tag)
					sb.WriteString(">\n")
				} else {
					sb.WriteString(">\n")
				}
			}
			return htmltemplate.HTML(sb.String())
		},
		"renderAttrs": func(attrs map[string]string) htmltemplate.HTML {
			if len(attrs) == 0 {
				return ""
			}
			keys := make([]string, 0, len(attrs))
			for k := range attrs {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			var sb strings.Builder
			for _, k := range keys {
				sb.WriteString(" ")
				sb.WriteString(html.EscapeString(k))
				sb.WriteString(`="`)
				sb.WriteString(html.EscapeString(attrs[k]))
				sb.WriteString(`"`)
			}
			return htmltemplate.HTML(sb.String())
		},
		"urlize": content.Slugify,
		"ref": func(slug string) string {
			if p := lookupPage(*pageIndexPtr, *sitePtr, slug); p != nil {
				return p.Permalink
			}
			return slug
		},
		"relref": func(slug string) string {
			if p := lookupPage(*pageIndexPtr, *sitePtr, slug); p != nil {
				return p.RelPermalink
			}
			return slug
		},

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

		// ── Assets ──
		"asset": func(path string) string {
			assetResolver := currentAssetResolver(assetResolverPtr)
			if assetResolver == nil {
				return path
			}
			resolved, err := assetResolver.Resolve(path)
			if err != nil {
				return path
			}
			url := "/assets/" + path
			_ = resolved
			if r := *urlResolverPtr; r != nil {
				url = r.URL(url, "", "")
			}
			return url
		},
		"fingerprint": func(path string) string {
			assetManifest := currentAssetManifest(assetManifestPtr)
			if assetManifest == nil {
				return path
			}
			entry, ok := assetManifest.Lookup(path)
			if !ok {
				return path
			}
			url := entry.OutputURL
			if r := *urlResolverPtr; r != nil {
				url = r.URL(url, "", "")
			}
			return url
		},
		"inline": func(path string) htmltemplate.HTML {
			assetResolver := currentAssetResolver(assetResolverPtr)
			if assetResolver == nil {
				return ""
			}
			data, err := assetResolver.ResolveContent(path)
			if err != nil {
				return ""
			}
			ext := strings.ToLower(filepath.Ext(path))
			switch ext {
			case ".css":
				return htmltemplate.HTML("<style>" + string(data) + "</style>")
			case ".js":
				return htmltemplate.HTML("<script>" + string(data) + "</script>")
			default:
				return htmltemplate.HTML(string(data))
			}
		},

		// ── Resource queries ──
		"getResource": func(resources []engine.Resource, name string) *engine.Resource {
			return asset.GetResource(resources, name)
		},
		"matchResources": func(resources []engine.Resource, pattern string) []engine.Resource {
			return asset.MatchResources(resources, pattern)
		},
		"resourcesByType": func(resources []engine.Resource, mediaType string) []engine.Resource {
			return asset.ResourcesByType(resources, mediaType)
		},

		// ── Image rendering ──
		"image": func(res engine.Resource) htmltemplate.HTML {
			return htmltemplate.HTML(asset.RenderPicture(
				res.RelPermalink, res.Title,
				res.Width, res.Height,
				nil, "", true,
			))
		},
		"resize_image": func(res engine.Resource, params string) htmltemplate.HTML {
			imageProcessor := currentImageProcessor(imageProcessorPtr)
			if imageProcessor == nil || res.SrcPath == "" {
				return htmltemplate.HTML(asset.RenderPicture(
					res.RelPermalink, res.Title,
					res.Width, res.Height,
					nil, "", true,
				))
			}
			opts := asset.ParseImageOptionsFromQuery(params)
			variants, lqip, err := imageProcessor.ProcessImage(res.SrcPath, opts)
			if err != nil {
				return htmltemplate.HTML(asset.RenderPicture(
					res.RelPermalink, res.Title,
					res.Width, res.Height,
					nil, "", true,
				))
			}
			return htmltemplate.HTML(asset.RenderPicture(
				res.RelPermalink, res.Title,
				res.Width, res.Height,
				variants, lqip, true,
			))
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

		// ── i18n ──
		"t": func(key string) string {
			if i18nStrings == nil || currentLang == nil {
				return key
			}
			return i18nStrings.Resolve(currentLang(), key)
		},
		"tWithData": func(key string, data any) string {
			if i18nStrings == nil || currentLang == nil {
				return key
			}
			return i18nStrings.Resolve(currentLang(), key, data)
		},

		// ── Navigation helpers ──
		"navFor": func(colName string) *engine.NavTree {
			s := *sitePtr
			if s == nil {
				return nil
			}
			col, ok := s.Collections[colName]
			if !ok || col == nil {
				return nil
			}
			return col.NavTree
		},
		"breadcrumbs": func(data any) []engine.BreadcrumbItem {
			rd, ok := data.(*engine.RouteData)
			if !ok || rd == nil {
				return nil
			}
			return rd.Breadcrumbs
		},
		"siblings": func(page *engine.Page) []*engine.Page {
			if page == nil || page.Section == nil {
				return nil
			}
			return page.Section.Pages
		},
		"translations": func(data any) []engine.TranslationLink {
			rd, ok := data.(*engine.RouteData)
			if !ok || rd == nil {
				return nil
			}
			return rd.Translations
		},

		// ── Versioning ──
		"versionOf": func(page *engine.Page, versionID string) *engine.Page {
			if page == nil {
				return nil
			}
			for _, peer := range page.VersionPeers {
				if peer.Version == versionID {
					return peer
				}
			}
			return nil
		},

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

		// ── Cross-collection ──
		"recentEntries": func(colName string, n int) []*engine.Page {
			s := *sitePtr
			if s == nil {
				return nil
			}
			col, ok := s.Collections[colName]
			if !ok || col == nil {
				return nil
			}
			pages := col.Pages
			if n > len(pages) {
				n = len(pages)
			}
			return pages[:n]
		},
		"findEntry": func(colName, slug string) *engine.Page {
			s := *sitePtr
			if s == nil {
				return nil
			}
			col, ok := s.Collections[colName]
			if !ok || col == nil {
				return nil
			}
			for _, p := range col.Pages {
				if p.Slug == slug {
					return p
				}
			}
			return nil
		},
		"allCollections": func() map[string]*engine.Collection {
			s := *sitePtr
			if s == nil {
				return nil
			}
			return s.Collections
		},

		// ── Taxonomy helpers ──
		// termURL is overridden in funcMapForLang to inject the rendering page's language.

		"topTerms": func(taxonomyName string, n int) []*engine.TaxonomyTerm {
			s := *sitePtr
			if s == nil {
				return nil
			}
			tax, ok := s.Taxonomies[taxonomyName]
			if !ok || tax == nil {
				return nil
			}
			terms := make([]*engine.TaxonomyTerm, 0, len(tax.Terms))
			for _, t := range tax.Terms {
				if !t.Hidden {
					terms = append(terms, t)
				}
			}
			sort.Slice(terms, func(i, j int) bool {
				if len(terms[i].Pages) != len(terms[j].Pages) {
					return len(terms[i].Pages) > len(terms[j].Pages)
				}
				return terms[i].Slug < terms[j].Slug
			})
			if n > 0 && n < len(terms) {
				terms = terms[:n]
			}
			return terms
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

	// Merge plugin-provided template functions.
	for k, v := range pluginFuncs {
		fm[k] = v
	}

	return fm
}

// ShortcodeFuncMapConfig holds the dependencies needed to build a FuncMap
// for shortcode templates. Component registry, plugin funcs, and i18n are
// intentionally excluded to avoid concurrency issues during parallel
// markdown rendering.
type ShortcodeFuncMapConfig struct {
	Site           **engine.SiteContext
	Resolver       *engine.ThemeResolver
	AssetResolver  *asset.Resolver
	AssetManifest  *asset.Manifest
	ImageProcessor *asset.ImageProcessor
	PageIndex      **content.PageIndex
}

// BuildShortcodeFuncMap constructs a FuncMap suitable for shortcode templates.
// It creates a temporary Engine wired to the caller's pointers so closures
// see mutations made through those pointers during the build.
func BuildShortcodeFuncMap(cfg ShortcodeFuncMapConfig) htmltemplate.FuncMap {
	e := &Engine{
		resolver:       cfg.Resolver,
		site:           *cfg.Site,
		pageIndex:      *cfg.PageIndex,
		assetResolver:  cfg.AssetResolver,
		assetManifest:  cfg.AssetManifest,
		imageProcessor: cfg.ImageProcessor,
	}
	return e.buildFuncMap(nil, nil)
}

func currentAssetResolver(ptr **asset.Resolver) *asset.Resolver {
	if ptr == nil {
		return nil
	}
	return *ptr
}

func currentAssetManifest(ptr **asset.Manifest) *asset.Manifest {
	if ptr == nil {
		return nil
	}
	return *ptr
}

func currentImageProcessor(ptr **asset.ImageProcessor) *asset.ImageProcessor {
	if ptr == nil {
		return nil
	}
	return *ptr
}

// ── Date function implementations ──

func fnDateFormat(t time.Time, layout string) string {
	return t.Format(layout)
}
