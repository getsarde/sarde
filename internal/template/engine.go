package template

import (
	"bytes"
	"fmt"
	htmltemplate "html/template"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/getsarde/sarde/internal/asset"
	"github.com/getsarde/sarde/internal/component"
	"github.com/getsarde/sarde/internal/consts"
	"github.com/getsarde/sarde/internal/content"
	"github.com/getsarde/sarde/internal/engine"
	"github.com/getsarde/sarde/internal/i18n"
	sardeplugin "github.com/getsarde/sarde/internal/plugin"
)

// Engine loads templates from the resolved lookup chain and renders pages to HTML.
type Engine struct {
	resolver       *engine.ThemeResolver
	components     *component.Registry
	baseCache      map[string]*htmltemplate.Template // layout key → baseof+partials
	templates      map[string]*htmltemplate.Template // "default:blog/single" → cloned template
	partialCache   map[string]*htmltemplate.Template // partial name → pre-parsed template
	funcMap        htmltemplate.FuncMap
	dataCache      sync.Map
	site           *engine.SiteContext
	cachedCSS      string // embedded CSS, loaded once
	cssURL         string // external CSS bundle URL (set during Load)
	tokenCSSURL    string // external token CSS URL (set by builder)
	codeBlockCSS   string // dynamically generated code block syntax theme CSS
	directiveCSS   string // concatenated generic directive CSS sidecars
	assetResolver  *asset.Resolver
	assetManifest  *asset.Manifest
	imageProcessor *asset.ImageProcessor
	pluginFuncs    map[string]any
	i18nStrings    *i18n.StringTable
	pageIndex      *content.PageIndex
	urlResolver    *engine.URLResolver
	currentLang    string // fallback lang for base-funcMap closures; per-render language is resolved via funcMapForLang(lang), so render correctness does not depend on this field
	loaded         bool   // true after first Load(); subsequent calls skip template re-parsing
	mu             sync.RWMutex
}

// NewEngine creates a new template Engine.
func NewEngine() *Engine {
	return &Engine{
		baseCache: make(map[string]*htmltemplate.Template),
		templates: make(map[string]*htmltemplate.Template),
	}
}

// SetSiteContext sets the site context used by template functions.
// Must be called before the first Render() call.
func (e *Engine) SetSiteContext(site *engine.SiteContext) {
	e.site = site
}

// SetAssetPipeline sets the asset resolver and manifest used by asset template functions.
// Must be called before Load().
func (e *Engine) SetAssetPipeline(resolver *asset.Resolver, manifest *asset.Manifest) {
	e.assetResolver = resolver
	e.assetManifest = manifest
}

// SetImageProcessor sets the image processor used by the resize_image template function.
// Must be called before Load().
func (e *Engine) SetImageProcessor(p *asset.ImageProcessor) {
	e.imageProcessor = p
}

// SetCodeBlockCSS sets dynamically generated code block syntax theme CSS.
// Must be called before Load().
func (e *Engine) SetCodeBlockCSS(css string) { e.codeBlockCSS = css }

// SetDirectiveCSS sets the concatenated CSS sidecars of all generic
// directives (internal/directive Registry.CSS). Appended unlayered to the
// main CSS bundle, same treatment as code block CSS. Must be called before
// Load().
func (e *Engine) SetDirectiveCSS(css string) { e.directiveCSS = css }

// SetTokenCSSURL sets the external URL for the theme token CSS file.
func (e *Engine) SetTokenCSSURL(url string) { e.tokenCSSURL = url }

// SetPluginFuncs sets additional template functions provided by plugins.
// Must be called before Load().
func (e *Engine) SetPluginFuncs(funcs map[string]any) {
	e.pluginFuncs = funcs
}

// SetI18nStrings sets the translation string table for the t() template function.
// Must be called before Load().
func (e *Engine) SetI18nStrings(st *i18n.StringTable) {
	e.i18nStrings = st
}

// SetPageIndex sets the page index for O(1) ref/relref lookups.
// Must be called before Load().
func (e *Engine) SetPageIndex(idx *content.PageIndex) {
	e.pageIndex = idx
}

// SetURLResolver sets the URL resolver for relURL/absURL template functions.
// Must be called before Load().
func (e *Engine) SetURLResolver(r *engine.URLResolver) {
	e.urlResolver = r
}

// SetCurrentLang sets a fallback language for the base-funcMap t() closure.
// This is NOT required for correct rendering: Render resolves the language
// per-page from RouteData and installs a language-specific t() via
// funcMapForLang on the cached template. Provided for API completeness.
func (e *Engine) SetCurrentLang(lang string) {
	e.currentLang = lang
}

// CurrentLangPtr returns a pointer to the current language field,
// suitable for capture by plugin closures that resolve i18n strings at render time.
func (e *Engine) CurrentLangPtr() *string {
	return &e.currentLang
}

// CachedCSS returns the concatenated embedded CSS bundle.
func (e *Engine) CachedCSS() string { return e.cachedCSS }

// CSSURL returns the root-relative URL of the embedded CSS bundle.
func (e *Engine) CSSURL() string { return e.cssURL }

// Load initializes the template system:
// loads base templates for each layout, sets up the component registry,
// and builds the FuncMap. In production mode (!devMode), the embedded CSS
// bundle is minified via esbuild. The output is always fingerprinted.
func (e *Engine) Load(resolver *engine.ThemeResolver, devMode bool) error {
	e.resolver = resolver

	// On subsequent loads (same engine reused across rebuilds), skip the
	// expensive template/component re-parsing. Only clear render caches so
	// new pages get fresh template clones with the updated SiteContext.
	if e.loaded {
		e.mu.Lock()
		e.templates = make(map[string]*htmltemplate.Template)
		e.mu.Unlock()
		e.dataCache = sync.Map{}
		return nil
	}

	// Load CSS bundle: prefer external theme CSS over embedded.
	var raw string
	if resolver.ThemeName != "" {
		raw = loadThemeCSS(resolver.ProjectDir, resolver.ThemeName)
	}
	if raw == "" {
		raw = loadEmbeddedCSS(resolver.EmbeddedFS)
	}
	if e.codeBlockCSS != "" {
		raw += "\n" + e.codeBlockCSS
	}
	if e.directiveCSS != "" {
		raw += "\n" + e.directiveCSS
	}
	processed, err := asset.TransformCSS(raw, !devMode)
	if err != nil {
		return fmt.Errorf("minifying embedded CSS: %w", err)
	}
	e.cachedCSS = processed
	hash := asset.Fingerprint([]byte(processed))
	e.cssURL = "/assets/css/" + asset.FingerprintedName("sarde.css", hash)

	// Build a bootstrap FuncMap (without component support or partial cache) for initial parsing.
	// Uses &e.site / &e.pageIndex so closures always see the latest values across rebuilds.
	bootstrapFM := e.buildFuncMap(nil, nil)

	// Create the component registry with embedded defaults.
	registry, err := component.NewRegistry(resolver.EmbeddedFS, bootstrapFM)
	if err != nil {
		return fmt.Errorf("creating component registry: %w", err)
	}

	// Load external plugin components (above embedded, below theme and user).
	for _, pluginDir := range resolver.PluginDirs {
		if err := registry.LoadOverridesFromDir(filepath.Join(pluginDir, consts.DirComponents)); err != nil {
			return fmt.Errorf("loading plugin component overrides: %w", err)
		}
	}

	// Load theme component overrides.
	if resolver.ThemeName != "" {
		themeCompDir := filepath.Join(resolver.ProjectDir, consts.DirThemes, resolver.ThemeName, consts.DirLayouts, consts.DirComponents)
		if err := registry.LoadOverridesFromDir(themeCompDir); err != nil {
			return fmt.Errorf("loading theme component overrides: %w", err)
		}
	}

	// Load user component overrides (highest priority).
	userCompDir := filepath.Join(resolver.ProjectDir, consts.DirLayouts, consts.DirComponents)
	if err := registry.LoadOverridesFromDir(userCompDir); err != nil {
		return fmt.Errorf("loading user component overrides: %w", err)
	}

	e.components = registry

	// Pre-parse all partials into a cache (read once, parsed once, reused on every render).
	e.partialCache = make(map[string]*htmltemplate.Template)
	partialData := resolveAllPartials(resolver)

	// Rebuild the final FuncMap with the real component registry and partial cache.
	e.funcMap = e.buildFuncMap(registry, e.partialCache)

	for name, data := range partialData {
		tmpl, err := htmltemplate.New(name).Funcs(e.funcMap).Parse(string(data))
		if err != nil {
			return fmt.Errorf("parsing partial %q: %w", name, err)
		}
		e.partialCache[name] = tmpl
	}

	// Re-register all components with the final FuncMap so they can call
	// template functions like `component`, `now`, etc.
	if err := e.reregisterComponents(); err != nil {
		return err
	}

	// Parse base templates (baseof + partials) for each layout type.
	for _, layout := range []engine.LayoutType{engine.LayoutDefault, engine.LayoutDocs, engine.LayoutPresentation, engine.LayoutLabs} {
		if err := e.loadBase(layout, partialData); err != nil {
			return fmt.Errorf("loading base for %q layout: %w", layout, err)
		}
	}

	e.loaded = true
	return nil
}

// ForceReload resets the loaded flag so the next Load() call
// re-parses all templates, components, and partials from disk.
func (e *Engine) ForceReload() {
	e.loaded = false
}

// Render renders a page using its
// resolved template and RouteData context.
func (e *Engine) Render(templateName string, data *engine.RouteData) ([]byte, error) {
	lang := renderLang(data, e.site)
	tmpl, err := e.getOrParseTemplate(templateName, data.Layout, data.Collection, lang)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("executing template %q: %w", templateName, err)
	}

	return buf.Bytes(), nil
}

func (e *Engine) currentLangResolver() func() string {
	return func() string {
		return e.currentLang
	}
}

func renderLang(data *engine.RouteData, site *engine.SiteContext) string {
	if data != nil {
		if data.Lang != "" {
			return data.Lang
		}
		if data.Page != nil && data.Page.Lang != "" {
			return data.Page.Lang
		}
	}
	if site != nil && site.Language != "" {
		return site.Language
	}
	return "en"
}

// lookupTaxonomies returns the taxonomy map for the given language,
// falling back to the default Taxonomies map.
func (e *Engine) lookupTaxonomies(lang string) map[string]*engine.Taxonomy {
	if e.site.TaxonomiesByLang != nil && lang != "" {
		if m, ok := e.site.TaxonomiesByLang[lang]; ok {
			return m
		}
	}
	return e.site.Taxonomies
}

func (e *Engine) funcMapForLang(lang string) htmltemplate.FuncMap {
	fm := make(htmltemplate.FuncMap, len(e.funcMap)+4)
	for k, v := range e.funcMap {
		fm[k] = v
	}

	fm["t"] = func(key string) string {
		if e.i18nStrings == nil {
			return key
		}
		return e.i18nStrings.Resolve(lang, key)
	}
	fm["tWithData"] = func(key string, data any) string {
		if e.i18nStrings == nil {
			return key
		}
		return e.i18nStrings.Resolve(lang, key, data)
	}
	// Locale-aware override of the base dateFormat func: presets resolve
	// through CLDR data for the render language, custom layouts keep plain
	// time.Format behavior.
	fm["dateFormat"] = func(t time.Time, format string) string {
		return localizedDateFormat(t, format, lang)
	}
	fm["relURL"] = func(relPath string) string {
		if strings.Contains(relPath, "://") {
			return relPath
		}
		if e.urlResolver != nil {
			return e.urlResolver.URL(relPath, lang, "")
		}
		return relPath
	}
	fm["absURL"] = func(relPath string) string {
		if e.urlResolver != nil {
			return e.urlResolver.AbsURL(relPath, lang, "")
		}
		return relPath
	}
	fm["termURL"] = func(taxonomyName, termName string) string {
		slug := content.Slugify(termName)
		url := "/" + taxonomyName + "/" + slug + "/"
		if e.site != nil {
			taxMap := e.lookupTaxonomies(lang)
			if tax, ok := taxMap[taxonomyName]; ok && tax != nil {
				if term, ok := tax.Terms[slug]; ok {
					url = term.Permalink
				}
			}
		}
		if e.urlResolver != nil {
			url = e.urlResolver.URL(url, lang, "")
		}
		return url
	}
	fm["lookupTerm"] = func(taxonomyName, termName string) *engine.TaxonomyTerm {
		if e.site == nil {
			return nil
		}
		slug := content.Slugify(termName)
		taxMap := e.lookupTaxonomies(lang)
		if tax, ok := taxMap[taxonomyName]; ok && tax != nil {
			if term, ok := tax.Terms[slug]; ok {
				return term
			}
		}
		return nil
	}
	fm["topTerms"] = func(taxonomyName string, n int) []*engine.TaxonomyTerm {
		if e.site == nil {
			return nil
		}
		taxMap := e.lookupTaxonomies(lang)
		tax, ok := taxMap[taxonomyName]
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
	}
	if e.pluginFuncs != nil {
		if fn, ok := e.pluginFuncs[sardeplugin.LangAwareAnnouncementFunc].(func(string) htmltemplate.HTML); ok {
			fm["announcementBanner"] = func() htmltemplate.HTML {
				return fn(lang)
			}
		}
	}
	if e.partialCache != nil {
		fm["partial"] = func(name string, data any) (htmltemplate.HTML, error) {
			tmpl := e.partialCache[name]
			if tmpl == nil {
				return "", fmt.Errorf("partial %q not found", name)
			}
			clone, err := tmpl.Clone()
			if err != nil {
				return "", fmt.Errorf("cloning partial %q: %w", name, err)
			}
			clone.Funcs(fm)
			var buf strings.Builder
			if err := clone.Execute(&buf, data); err != nil {
				return "", fmt.Errorf("rendering partial %q: %w", name, err)
			}
			return htmltemplate.HTML(buf.String()), nil
		}
	}
	if e.components != nil {
		fm["component"] = func(name string, data any) (htmltemplate.HTML, error) {
			return e.components.RenderComponentWithFuncs(name, data, fm)
		}
	}
	return fm
}

// getOrParseTemplate returns a cached template or resolves, parses, and caches it.
func (e *Engine) getOrParseTemplate(name string, layout engine.LayoutType, col *engine.Collection, lang string) (*htmltemplate.Template, error) {
	cacheKey := string(layout) + ":" + name + ":" + lang

	// Fast path: read lock.
	e.mu.RLock()
	if tmpl, ok := e.templates[cacheKey]; ok {
		e.mu.RUnlock()
		return tmpl, nil
	}
	e.mu.RUnlock()

	// Slow path: write lock.
	e.mu.Lock()
	defer e.mu.Unlock()

	// Double-check after acquiring write lock.
	if tmpl, ok := e.templates[cacheKey]; ok {
		return tmpl, nil
	}

	// Get the base template for this layout. Map non-primary layouts to their base.
	baseLayout := layout
	switch baseLayout {
	case engine.LayoutSplash, engine.LayoutFull, engine.LayoutCentered:
		baseLayout = engine.LayoutDefault
	case engine.LayoutWide:
		baseLayout = engine.LayoutDocs
	}
	base, ok := e.baseCache[string(baseLayout)]
	if !ok {
		return nil, fmt.Errorf("no base template for layout %q", baseLayout)
	}

	// Clone the base (baseof + partials).
	clone, err := base.Clone()
	if err != nil {
		return nil, fmt.Errorf("cloning base template: %w", err)
	}
	clone.Funcs(e.funcMapForLang(lang))

	// Extract collection and file from template name.
	// e.g., "blog/single" → collection="blog", file="single.html"
	collection := ""
	templateFile := name
	if parts := strings.SplitN(name, "/", 2); len(parts) == 2 {
		collection = parts[0]
		templateFile = parts[1]
	}
	if !strings.HasSuffix(templateFile, ".html") {
		templateFile += ".html"
	}

	content, resolvedPath, err := resolveTemplate(e.resolver, collection, layout, templateFile)
	if err != nil {
		return nil, err
	}

	// Parse the page template into the cloned base.
	if _, err := clone.New(resolvedPath).Parse(string(content)); err != nil {
		return nil, fmt.Errorf("parsing template %q from %s: %w", name, resolvedPath, err)
	}

	e.templates[cacheKey] = clone
	return clone, nil
}

// loadBase loads baseof.html and all partials for a layout type.
func (e *Engine) loadBase(layout engine.LayoutType, partials map[string][]byte) error {
	content, resolvedPath, err := resolveTemplate(e.resolver, "", layout, consts.TemplateBaseOf)
	if err != nil {
		return fmt.Errorf("resolving baseof.html: %w", err)
	}

	base, err := htmltemplate.New(resolvedPath).Funcs(e.funcMap).Parse(string(content))
	if err != nil {
		return fmt.Errorf("parsing baseof.html from %s: %w", resolvedPath, err)
	}

	for name, data := range partials {
		if _, err := base.New(consts.DirPartials + "/" + name).Parse(string(data)); err != nil {
			return fmt.Errorf("parsing partial %q: %w", name, err)
		}
	}

	e.baseCache[string(layout)] = base
	return nil
}

// reregisterComponents re-parses all component templates with the final funcMap
// so they can call functions like `component`, `now`, etc.
func (e *Engine) reregisterComponents() error {
	// Re-register from embedded FS.
	if e.resolver.EmbeddedFS != nil {
		if err := e.reregisterFromFS(e.resolver.EmbeddedFS, consts.DirComponents); err != nil {
			return err
		}
	}

	// Re-register theme overrides.
	if e.resolver.ThemeName != "" {
		dir := filepath.Join(e.resolver.ProjectDir, consts.DirThemes, e.resolver.ThemeName, consts.DirLayouts, consts.DirComponents)
		if err := e.reregisterFromFS(os.DirFS(dir), "."); err != nil {
			return fmt.Errorf("re-parsing component from %s: %w", dir, err)
		}
	}

	// Re-register user overrides (highest priority).
	dir := filepath.Join(e.resolver.ProjectDir, consts.DirLayouts, consts.DirComponents)
	if err := e.reregisterFromFS(os.DirFS(dir), "."); err != nil {
		return fmt.Errorf("re-parsing component from %s: %w", dir, err)
	}
	return nil
}

func (e *Engine) reregisterFromFS(efs fs.FS, dir string) error {
	entries, err := fs.ReadDir(efs, dir)
	if err != nil {
		return nil
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".html") {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".html")
		data, err := fs.ReadFile(efs, path.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		tmpl, err := htmltemplate.New(name).Funcs(e.funcMap).Parse(string(data))
		if err != nil {
			return fmt.Errorf("re-parsing component %q: %w", name, err)
		}
		e.components.RegisterTemplate(name, tmpl)
	}
	return nil
}
