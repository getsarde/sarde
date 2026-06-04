package template

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	htmltemplate "html/template"
	"math"
	"math/rand"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/frostybee/sarde/internal/asset"
	"github.com/frostybee/sarde/internal/component"
	"github.com/frostybee/sarde/internal/content"
	"github.com/frostybee/sarde/internal/content/markdown/icons"
	"github.com/frostybee/sarde/internal/engine"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
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

// ── String function implementations ──

func fnTitle(s string) string {
	words := strings.Fields(s)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
		}
	}
	return strings.Join(words, " ")
}

func fnTruncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	if n <= 3 {
		return string(runes[:n])
	}
	return string(runes[:n-3]) + "..."
}

var (
	inlineMDOnce sync.Once
	inlineMD     goldmark.Markdown
)

func getInlineMD() goldmark.Markdown {
	inlineMDOnce.Do(func() {
		inlineMD = goldmark.New(
			goldmark.WithExtensions(extension.GFM),
		)
	})
	return inlineMD
}

func fnMarkdownify(s string) htmltemplate.HTML {
	var buf bytes.Buffer
	if err := getInlineMD().Convert([]byte(s), &buf); err != nil {
		return htmltemplate.HTML(htmltemplate.HTMLEscapeString(s))
	}
	return htmltemplate.HTML(buf.String())
}

func fnPlainify(s string) string {
	// Strip HTML tags
	var result strings.Builder
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			result.WriteRune(r)
		}
	}
	return result.String()
}

func lookupPage(pi *content.PageIndex, site *engine.SiteContext, slug string) *engine.Page {
	if pi != nil {
		if p := pi.LookupByPermalink(slug); p != nil {
			return p
		}
		if p := pi.LookupBySlug(slug); p != nil {
			return p
		}
	}
	if site != nil {
		for _, p := range site.Pages {
			if p.Slug == slug || p.RelPermalink == slug {
				return p
			}
		}
	}
	return nil
}

func fnSafeHTML(s string) htmltemplate.HTML {
	return htmltemplate.HTML(s)
}

// fnIcon renders an inline SVG icon via icons.Render. The first variadic arg is
// a CSS class; the remaining args are key/value attribute pairs (a trailing odd
// arg is ignored):
//
//	{{ icon "rocket" }}
//	{{ icon "rocket" "sarde-icon-lg" }}
//	{{ icon "arrow-up" "sarde-icon" "rotate" "90" "title" "Up" }}
//
// The SVG body is trusted build-time icon data and icons.Render escapes all
// attribute values, so wrapping the result in template.HTML is safe.
func fnIcon(name string, args ...string) htmltemplate.HTML {
	var class string
	attrs := make(map[string]string)
	if len(args) > 0 {
		class = args[0]
		pairs := args[1:]
		for i := 0; i+1 < len(pairs); i += 2 {
			attrs[pairs[i]] = pairs[i+1]
		}
	}
	return htmltemplate.HTML(icons.Render(name, class, attrs))
}

func fnHighlight(code, lang string) htmltemplate.HTML {
	// Stub: wraps code in <pre><code>. Full Chroma syntax highlighting is not yet implemented.
	escaped := htmltemplate.HTMLEscapeString(code)
	return htmltemplate.HTML(fmt.Sprintf(`<pre><code class="language-%s">%s</code></pre>`, htmltemplate.HTMLEscapeString(lang), escaped))
}

// ── Date function implementations ──

func fnDateFormat(t time.Time, layout string) string {
	return t.Format(layout)
}

// ── Math function implementations ──

func toFloat64(v any) (float64, bool) {
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(rv.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(rv.Uint()), true
	case reflect.Float32, reflect.Float64:
		return rv.Float(), true
	default:
		return 0, false
	}
}

func fnAdd(a, b any) any {
	af, aOk := toFloat64(a)
	bf, bOk := toFloat64(b)
	if !aOk || !bOk {
		return 0
	}
	result := af + bf
	if isInt(a) && isInt(b) {
		return int(result)
	}
	return result
}

func fnSub(a, b any) any {
	af, aOk := toFloat64(a)
	bf, bOk := toFloat64(b)
	if !aOk || !bOk {
		return 0
	}
	result := af - bf
	if isInt(a) && isInt(b) {
		return int(result)
	}
	return result
}

func fnMul(a, b any) any {
	af, aOk := toFloat64(a)
	bf, bOk := toFloat64(b)
	if !aOk || !bOk {
		return 0
	}
	result := af * bf
	if isInt(a) && isInt(b) {
		return int(result)
	}
	return result
}

func fnDiv(a, b any) any {
	af, aOk := toFloat64(a)
	bf, bOk := toFloat64(b)
	if !aOk || !bOk || bf == 0 {
		return 0
	}
	result := af / bf
	if isInt(a) && isInt(b) {
		return int(math.Trunc(result))
	}
	return result
}

func fnMod(a, b any) int {
	af, aOk := toFloat64(a)
	bf, bOk := toFloat64(b)
	if !aOk || !bOk || bf == 0 {
		return 0
	}
	return int(af) % int(bf)
}

func isInt(v any) bool {
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return true
	default:
		return false
	}
}

// ���─ Logic function implementations ──

func fnCond(condition bool, trueVal, falseVal any) any {
	if condition {
		return trueVal
	}
	return falseVal
}

func fnDefault(value, fallback any) any {
	if value == nil {
		return fallback
	}
	rv := reflect.ValueOf(value)
	if rv.IsZero() {
		return fallback
	}
	return value
}

func fnIsset(m map[string]any, key string) bool {
	_, ok := m[key]
	return ok
}

// ── Collection function implementations ──

func fnFirst(n int, list any) any {
	v := reflect.ValueOf(list)
	if v.Kind() != reflect.Slice {
		return list
	}
	if n > v.Len() {
		n = v.Len()
	}
	return v.Slice(0, n).Interface()
}

func fnLast(n int, list any) any {
	v := reflect.ValueOf(list)
	if v.Kind() != reflect.Slice {
		return list
	}
	l := v.Len()
	if n > l {
		n = l
	}
	return v.Slice(l-n, l).Interface()
}

func fnAfter(n int, list any) any {
	v := reflect.ValueOf(list)
	if v.Kind() != reflect.Slice {
		return list
	}
	if n > v.Len() {
		n = v.Len()
	}
	return v.Slice(n, v.Len()).Interface()
}

func fnShuffle(list any) any {
	v := reflect.ValueOf(list)
	if v.Kind() != reflect.Slice {
		return list
	}
	l := v.Len()
	result := reflect.MakeSlice(v.Type(), l, l)
	reflect.Copy(result, v)
	rand.Shuffle(l, func(i, j int) {
		vi := result.Index(i).Interface()
		vj := result.Index(j).Interface()
		result.Index(i).Set(reflect.ValueOf(vj))
		result.Index(j).Set(reflect.ValueOf(vi))
	})
	return result.Interface()
}

func fnSort(list any, field string) any {
	v := reflect.ValueOf(list)
	if v.Kind() != reflect.Slice {
		return list
	}
	l := v.Len()
	result := reflect.MakeSlice(v.Type(), l, l)
	reflect.Copy(result, v)

	sort.SliceStable(result.Interface(), func(i, j int) bool {
		a := getField(result.Index(i).Interface(), field)
		b := getField(result.Index(j).Interface(), field)
		return fmt.Sprint(a) < fmt.Sprint(b)
	})
	return result.Interface()
}

func fnWhere(list any, field string, value any) any {
	v := reflect.ValueOf(list)
	if v.Kind() != reflect.Slice {
		return list
	}
	result := reflect.MakeSlice(v.Type(), 0, v.Len())
	for i := 0; i < v.Len(); i++ {
		item := v.Index(i).Interface()
		fv := getField(item, field)
		if fmt.Sprint(fv) == fmt.Sprint(value) {
			result = reflect.Append(result, v.Index(i))
		}
	}
	return result.Interface()
}

func fnGroup(list any, field string) map[string]any {
	v := reflect.ValueOf(list)
	if v.Kind() != reflect.Slice {
		return nil
	}
	groups := make(map[string][]any)
	for i := 0; i < v.Len(); i++ {
		item := v.Index(i).Interface()
		key := fmt.Sprint(getField(item, field))
		groups[key] = append(groups[key], item)
	}
	result := make(map[string]any, len(groups))
	for k, v := range groups {
		result[k] = v
	}
	return result
}

func fnUniq(list any) any {
	v := reflect.ValueOf(list)
	if v.Kind() != reflect.Slice {
		return list
	}
	seen := make(map[string]bool)
	result := reflect.MakeSlice(v.Type(), 0, v.Len())
	for i := 0; i < v.Len(); i++ {
		key := fmt.Sprint(v.Index(i).Interface())
		if !seen[key] {
			seen[key] = true
			result = reflect.Append(result, v.Index(i))
		}
	}
	return result.Interface()
}

func fnIn(list any, value any) bool {
	v := reflect.ValueOf(list)
	if v.Kind() != reflect.Slice {
		return false
	}
	target := fmt.Sprint(value)
	for i := 0; i < v.Len(); i++ {
		if fmt.Sprint(v.Index(i).Interface()) == target {
			return true
		}
	}
	return false
}

func fnSeq(args ...int) []int {
	switch len(args) {
	case 1:
		result := make([]int, args[0])
		for i := range result {
			result[i] = i + 1
		}
		return result
	case 2:
		start, end := args[0], args[1]
		if start > end {
			return nil
		}
		result := make([]int, end-start+1)
		for i := range result {
			result[i] = start + i
		}
		return result
	default:
		return nil
	}
}

// ── Debug function implementations ──

func fnJsonify(value any) htmltemplate.HTML {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return htmltemplate.HTML(fmt.Sprintf("<!-- jsonify error: %s -->", err))
	}
	return htmltemplate.HTML(b)
}

func fnDump(value any) htmltemplate.HTML {
	return htmltemplate.HTML(fmt.Sprintf("<pre>%s</pre>", htmltemplate.HTMLEscapeString(fmt.Sprintf("%+v", value))))
}

// ── Join helper (argument order matches template piping: join .List ", ") ─��

func fnJoin(list []string, sep string) string {
	return strings.Join(list, sep)
}

// ── Reflection helper ──

func getField(item any, field string) any {
	v := reflect.ValueOf(item)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() == reflect.Struct {
		f := v.FieldByName(field)
		if f.IsValid() {
			return f.Interface()
		}
	}
	if v.Kind() == reflect.Map {
		f := v.MapIndex(reflect.ValueOf(field))
		if f.IsValid() {
			return f.Interface()
		}
	}
	return nil
}

// wrapCSS wraps raw CSS in a <style> tag.
func wrapCSS(css string) string {
	if css == "" {
		return ""
	}
	return "<style>\n" + css + "</style>"
}
