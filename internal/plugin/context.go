package plugin

import (
	"strings"
	"sync"

	"github.com/getsarde/sarde/internal/atomicwrite"
	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/content"
	"github.com/getsarde/sarde/internal/engine"
	"github.com/getsarde/sarde/internal/outputpath"
)

// LangAwareAnnouncementFunc is an internal template func factory key used by the
// announcements plugin so concurrent multilingual renders do not share mutable language state.
const LangAwareAnnouncementFunc = "__sarde_announcementBannerForLang"

// Plugin is a named bundle of lifecycle hooks.
type Plugin struct {
	Name  string
	Hooks PluginHooks
}

// PluginHooks holds optional hook functions. Nil hooks are not called.
type PluginHooks struct {
	ConfigSetup   func(ctx *ConfigSetupContext) error
	ContentLoaded func(ctx *ContentLoadedContext) error
	BeforeRender  func(ctx *BeforeRenderContext) error
	BuildDone     func(ctx *BuildDoneContext) error
}

// ---------------------------------------------------------------------------
// Context types
// ---------------------------------------------------------------------------

// ConfigSetupContext is available after config is loaded, before content discovery.
type ConfigSetupContext struct {
	Config        *config.SiteConfig
	PluginConfig  map[string]any
	TemplateFuncs map[string]any // collected via AddTemplateFunc
	store         *SharedStore
}

// Set stores a value in the shared cross-hook store.
func (c *ConfigSetupContext) Set(key string, value any) { c.store.Set(key, value) }

// Get retrieves a value from the shared cross-hook store.
func (c *ConfigSetupContext) Get(key string) any { return c.store.Get(key) }

// AddTemplateFunc registers a template function to be available in templates.
func (c *ConfigSetupContext) AddTemplateFunc(name string, fn any) {
	if c.TemplateFuncs == nil {
		c.TemplateFuncs = make(map[string]any)
	}
	c.TemplateFuncs[name] = fn
}

// ContentLoadedContext is available after content parsing, before assembly.
type ContentLoadedContext struct {
	Config       *config.SiteConfig
	PluginConfig map[string]any
	Collections  map[string]*engine.Collection
	Pages        *[]*engine.Page // pointer — plugins can append via InjectPage
	store        *SharedStore
}

// Set stores a value in the shared cross-hook store.
func (c *ContentLoadedContext) Set(key string, value any) { c.store.Set(key, value) }

// Get retrieves a value from the shared cross-hook store.
func (c *ContentLoadedContext) Get(key string) any { return c.store.Get(key) }

// InjectPage appends a virtual page to the page list.
func (c *ContentLoadedContext) InjectPage(page *engine.Page) {
	*c.Pages = append(*c.Pages, page)
}

// resolveURL applies resolver URL resolution, falling back to relPath.
func resolveURL(resolver *engine.URLResolver, relPath, lang, version string) string {
	if resolver != nil {
		return resolver.URL(relPath, lang, version)
	}
	return relPath
}

// absURL builds a fully-qualified URL via the resolver, falling back to the
// site origin, then to relPath itself.
func absURL(resolver *engine.URLResolver, site *engine.SiteContext, relPath, lang, version string) string {
	if resolver != nil {
		return resolver.AbsURL(relPath, lang, version)
	}
	if site != nil {
		return strings.TrimRight(site.BaseURL, "/") + relPath
	}
	return relPath
}

// siteBaseURL returns the site base URL without its trailing slash, or "".
func siteBaseURL(site *engine.SiteContext) string {
	if site == nil {
		return ""
	}
	return strings.TrimRight(site.BaseURL, "/")
}

// BeforeRenderContext is available per-page before template rendering.
type BeforeRenderContext struct {
	Page         *engine.Page
	RouteData    *engine.RouteData
	Site         *engine.SiteContext
	Resolver     *engine.URLResolver
	PluginConfig map[string]any
	store        *SharedStore
}

// ResolveURL returns a root-relative URL with basePath, lang, and version applied.
func (c *BeforeRenderContext) ResolveURL(relPath, lang, version string) string {
	return resolveURL(c.Resolver, relPath, lang, version)
}

// AbsURL returns a fully-qualified URL (origin + resolved path).
func (c *BeforeRenderContext) AbsURL(relPath, lang, version string) string {
	return absURL(c.Resolver, c.Site, relPath, lang, version)
}

// BaseURL returns the site base URL without its trailing slash, or "".
func (c *BeforeRenderContext) BaseURL() string { return siteBaseURL(c.Site) }

// Set stores a value in the shared cross-hook store.
func (c *BeforeRenderContext) Set(key string, value any) { c.store.Set(key, value) }

// Get retrieves a value from the shared cross-hook store.
func (c *BeforeRenderContext) Get(key string) any { return c.store.Get(key) }

// BuildDoneContext is available after all files are written. Thread-safe for parallel use.
type BuildDoneContext struct {
	Config         *config.SiteConfig
	PluginConfig   map[string]any
	OutputDir      string
	Pages          []*engine.Page
	Collections    map[string]*engine.Collection
	Site           *engine.SiteContext
	Resolver       *engine.URLResolver
	PageIndex      *content.PageIndex                // page index for link validation
	ValidationData map[string]engine.ValidationEntry // permalink -> collected links per page
	DevMode        bool
	Incremental    bool // true after an incremental rebuild rather than a full Build()

	// ChangedPages holds the pages whose parsed content actually changed
	// during an incremental rebuild. Nil on full builds, non-empty whenever
	// Incremental is true (the incremental path only reaches BuildDone when
	// at least one page changed). Plugins that want per-page work
	// proportional to a save's actual diff should branch on Incremental and
	// use ChangedPages instead of Pages.
	ChangedPages []*engine.Page

	// RemovedPermalinks lists permalinks removed by this rebuild. Always nil
	// today: new and deleted content files force a full rebuild, so the
	// incremental path never has removals to report. Reserved for a future
	// incremental path that handles deletions without falling back.
	RemovedPermalinks []string

	TrackFn    func(string)
	mu         *sync.Mutex
	warnings   *[]engine.ValidationWarning
	logger     *engine.BuildLogger
	pluginName string
}

// ResolveURL returns a root-relative URL with basePath, lang, and version applied.
func (c *BuildDoneContext) ResolveURL(relPath, lang, version string) string {
	return resolveURL(c.Resolver, relPath, lang, version)
}

// AbsURL returns a fully-qualified URL (origin + resolved path).
func (c *BuildDoneContext) AbsURL(relPath, lang, version string) string {
	return absURL(c.Resolver, c.Site, relPath, lang, version)
}

// BaseURL returns the site base URL without its trailing slash, or "".
func (c *BuildDoneContext) BaseURL() string { return siteBaseURL(c.Site) }

func (c *BuildDoneContext) initMu() {
	if c.mu == nil {
		c.mu = &sync.Mutex{}
	}
}

// Log emits a build log message prefixed with the plugin's name.
func (c *BuildDoneContext) Log(message string) {
	if c.logger != nil {
		c.logger.Log(c.pluginName, message)
	}
}

// SetLogger sets the build logger. Must be called before RunBuildDone.
func (c *BuildDoneContext) SetLogger(l *engine.BuildLogger) {
	c.logger = l
}

// WriteFile writes a file to the output directory. Thread-safe.
func (c *BuildDoneContext) WriteFile(relPath string, data []byte) error {
	absPath, err := outputpath.SafeJoin(c.OutputDir, relPath)
	if err != nil {
		return err
	}
	c.initMu()
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.TrackFn != nil {
		c.TrackFn(absPath)
	}
	return atomicwrite.WriteFile(absPath, data, 0o644)
}

// SetWarnings sets the warnings slice pointer. Must be called before RunBuildDone.
func (c *BuildDoneContext) SetWarnings(w *[]engine.ValidationWarning) {
	c.warnings = w
}

// AddWarning appends a validation warning. Thread-safe.
func (c *BuildDoneContext) AddWarning(w engine.ValidationWarning) {
	c.initMu()
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.warnings == nil {
		return
	}
	*c.warnings = append(*c.warnings, w)
}
