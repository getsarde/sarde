package template

import (
	"bytes"
	"fmt"
	htmltemplate "html/template"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/frostybee/sarde/internal/asset"
	"github.com/frostybee/sarde/internal/component"
	"github.com/frostybee/sarde/internal/consts"
	"github.com/frostybee/sarde/internal/content"
	"github.com/frostybee/sarde/internal/engine"
	"github.com/frostybee/sarde/internal/i18n"
)

// Engine implements engine.TemplateEngine using Go's html/template.
type Engine struct {
	resolver       *engine.ThemeResolver
	components     *component.Registry
	baseCache      map[string]*htmltemplate.Template // layout key → baseof+partials
	templates      map[string]*htmltemplate.Template // "default:blog/single" → cloned template
	funcMap        htmltemplate.FuncMap
	dataCache      sync.Map
	site           *engine.SiteContext
	cachedCSS      string // embedded CSS, loaded once
	cssURL         string // external CSS bundle URL (set during Load)
	assetResolver  *asset.Resolver
	assetManifest  *asset.Manifest
	imageProcessor *asset.ImageProcessor
	pluginFuncs    map[string]any
	i18nStrings    *i18n.StringTable
	pageIndex      *content.PageIndex
	currentLang    string // set per-page before render, captured by t() closure
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

// SetCurrentLang sets the current language for the t() template function.
// Must be called before each Render() call.
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

// Load implements engine.TemplateEngine. It initializes the template system:
// loads base templates for each layout, sets up the component registry,
// and builds the FuncMap.
func (e *Engine) Load(resolver *engine.ThemeResolver) error {
	e.resolver = resolver

	// Load and cache embedded CSS files.
	e.cachedCSS = loadEmbeddedCSS(resolver.EmbeddedFS)
	e.cssURL = "/assets/css/sarde.css"

	// Build a bootstrap FuncMap (without component support) for initial parsing.
	bootstrapFM := buildFuncMap(e.site, resolver, nil, &e.dataCache, e.cachedCSS, &e.cssURL, e.assetResolver, e.assetManifest, e.imageProcessor, e.pluginFuncs, &e.currentLang, e.i18nStrings, e.pageIndex)

	// Create the component registry with embedded defaults.
	registry, err := component.NewRegistry(resolver.EmbeddedFS, bootstrapFM)
	if err != nil {
		return fmt.Errorf("creating component registry: %w", err)
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

	// Rebuild the final FuncMap with the real component registry.
	e.funcMap = buildFuncMap(e.site, resolver, registry, &e.dataCache, e.cachedCSS, &e.cssURL, e.assetResolver, e.assetManifest, e.imageProcessor, e.pluginFuncs, &e.currentLang, e.i18nStrings, e.pageIndex)

	// Re-register all components with the final FuncMap so they can call
	// template functions like `component`, `now`, etc.
	if err := e.reregisterComponents(); err != nil {
		return err
	}

	// Parse base templates (baseof + partials) for each layout type.
	for _, layout := range []engine.LayoutType{engine.LayoutDefault, engine.LayoutDocs} {
		if err := e.loadBase(layout); err != nil {
			return fmt.Errorf("loading base for %q layout: %w", layout, err)
		}
	}

	return nil
}

// Render implements engine.TemplateEngine. It renders a page using its
// resolved template and RouteData context.
func (e *Engine) Render(templateName string, data *engine.RouteData) ([]byte, error) {
	tmpl, err := e.getOrParseTemplate(templateName, data.Layout, data.Collection)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("executing template %q: %w", templateName, err)
	}

	return buf.Bytes(), nil
}

// getOrParseTemplate returns a cached template or resolves, parses, and caches it.
func (e *Engine) getOrParseTemplate(name string, layout engine.LayoutType, col *engine.Collection) (*htmltemplate.Template, error) {
	cacheKey := string(layout) + ":" + name

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
func (e *Engine) loadBase(layout engine.LayoutType) error {
	content, resolvedPath, err := resolveTemplate(e.resolver, "", layout, consts.TemplateBaseOf)
	if err != nil {
		return fmt.Errorf("resolving baseof.html: %w", err)
	}

	base, err := htmltemplate.New(resolvedPath).Funcs(e.funcMap).Parse(string(content))
	if err != nil {
		return fmt.Errorf("parsing baseof.html from %s: %w", resolvedPath, err)
	}

	// Load all partials into the base template.
	partials := resolveAllPartials(e.resolver)
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
		if err := e.reregisterFromOSDir(dir); err != nil {
			return err
		}
	}

	// Re-register user overrides (highest priority).
	dir := filepath.Join(e.resolver.ProjectDir, consts.DirLayouts, consts.DirComponents)
	return e.reregisterFromOSDir(dir)
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

func (e *Engine) reregisterFromOSDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil // directory may not exist
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".html") {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".html")
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		tmpl, err := htmltemplate.New(name).Funcs(e.funcMap).Parse(string(data))
		if err != nil {
			return fmt.Errorf("re-parsing component %q from %s: %w", name, dir, err)
		}
		e.components.RegisterTemplate(name, tmpl)
	}
	return nil
}

// loadEmbeddedCSS reads and concatenates all CSS files from the embedded FS.
func loadEmbeddedCSS(efs fs.FS) string {
	if efs == nil {
		return ""
	}

	cssOrder := []string{
		"css/tokens.css",
		"css/base.css",
		"css/layout.css",
		"css/content.css",
		"css/components.css",
		"css/extensions.css",
		"css/style.css",
		"css/blog.css",
		"css/taxonomy.css",
		"css/search.css",
		"css/homepage.css",
		"css/utilities.css",
		"css/print.css",
		"css/dark.css",
	}

	var sb strings.Builder
	sb.WriteString("@layer sarde.base, sarde.reset, sarde.core, sarde.content, sarde.components, sarde.variants, sarde.utils;\n")
	for _, name := range cssOrder {
		data, err := fs.ReadFile(efs, name)
		if err != nil {
			continue
		}
		sb.Write(data)
		sb.WriteByte('\n')
	}
	return sb.String()
}
