package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/frostybee/sarde/internal/config"
	"github.com/frostybee/sarde/internal/content"
	"github.com/frostybee/sarde/internal/engine"
)

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

// BeforeRenderContext is available per-page before template rendering.
type BeforeRenderContext struct {
	Page         *engine.Page
	RouteData    *engine.RouteData
	Site         *engine.SiteContext
	PluginConfig map[string]any
	store        *SharedStore
}

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
	PageIndex      *content.PageIndex                // page index for link validation
	ValidationData map[string]engine.ValidationEntry // permalink -> collected links per page
	DevMode        bool
	mu             sync.Mutex
	warnings       *[]engine.ValidationWarning
}

// WriteFile writes a file to the output directory. Thread-safe.
func (c *BuildDoneContext) WriteFile(relPath string, data []byte) error {
	absPath := filepath.Join(c.OutputDir, filepath.FromSlash(relPath))
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(absPath, data, 0o644)
}

// SetWarnings sets the warnings slice pointer. Must be called before RunBuildDone.
func (c *BuildDoneContext) SetWarnings(w *[]engine.ValidationWarning) {
	c.warnings = w
}

// AddWarning appends a validation warning. Thread-safe.
func (c *BuildDoneContext) AddWarning(w engine.ValidationWarning) {
	c.mu.Lock()
	defer c.mu.Unlock()
	*c.warnings = append(*c.warnings, w)
}

// ---------------------------------------------------------------------------
// Manager
// ---------------------------------------------------------------------------

// Manager orchestrates plugin registration and lifecycle hook execution.
type Manager struct {
	plugins       []*Plugin
	store         *SharedStore
	templateFuncs map[string]any
}

// NewManager creates an empty plugin manager.
func NewManager() *Manager {
	return &Manager{
		store:         NewStore(),
		templateFuncs: make(map[string]any),
	}
}

// Register adds a plugin to the manager.
func (m *Manager) Register(p *Plugin) {
	m.plugins = append(m.plugins, p)
}

// RegisterBuiltins registers all enabled built-in plugins.
func (m *Manager) RegisterBuiltins(enabled []string, configs map[string]map[string]any) {
	for _, name := range enabled {
		constructor, ok := builtinRegistry[name]
		if !ok {
			continue
		}
		cfg := configs[name]
		m.Register(constructor(cfg))
	}
}

// TemplateFuncs returns template functions collected from ConfigSetup hooks.
func (m *Manager) TemplateFuncs() map[string]any {
	return m.templateFuncs
}

// RunConfigSetup executes ConfigSetup hooks serially for all plugins.
func (m *Manager) RunConfigSetup(cfg *config.SiteConfig) error {
	for _, p := range m.plugins {
		if p.Hooks.ConfigSetup == nil {
			continue
		}
		ctx := &ConfigSetupContext{
			Config:       cfg,
			PluginConfig: m.pluginConfig(p.Name, cfg),
			store:        m.store,
		}
		if err := p.Hooks.ConfigSetup(ctx); err != nil {
			return fmt.Errorf("plugin %q ConfigSetup: %w", p.Name, err)
		}
		// Merge any template funcs registered by this plugin.
		for k, v := range ctx.TemplateFuncs {
			m.templateFuncs[k] = v
		}
	}
	return nil
}

// RunContentLoaded executes ContentLoaded hooks serially for all plugins.
func (m *Manager) RunContentLoaded(cfg *config.SiteConfig, collections map[string]*engine.Collection, pages *[]*engine.Page) error {
	for _, p := range m.plugins {
		if p.Hooks.ContentLoaded == nil {
			continue
		}
		ctx := &ContentLoadedContext{
			Config:       cfg,
			PluginConfig: m.pluginConfig(p.Name, cfg),
			Collections:  collections,
			Pages:        pages,
			store:        m.store,
		}
		if err := p.Hooks.ContentLoaded(ctx); err != nil {
			return fmt.Errorf("plugin %q ContentLoaded: %w", p.Name, err)
		}
	}
	return nil
}

// RunBeforeRender executes BeforeRender hooks serially for a single page.
func (m *Manager) RunBeforeRender(cfg *config.SiteConfig, page *engine.Page, rd *engine.RouteData, site *engine.SiteContext) error {
	for _, p := range m.plugins {
		if p.Hooks.BeforeRender == nil {
			continue
		}
		ctx := &BeforeRenderContext{
			Page:         page,
			RouteData:    rd,
			Site:         site,
			PluginConfig: m.pluginConfig(p.Name, cfg),
			store:        m.store,
		}
		if err := p.Hooks.BeforeRender(ctx); err != nil {
			return fmt.Errorf("plugin %q BeforeRender: %w", p.Name, err)
		}
	}
	return nil
}

// RunBuildDone executes BuildDone hooks in parallel for all plugins.
func (m *Manager) RunBuildDone(ctx *BuildDoneContext) error {
	var wg sync.WaitGroup
	errCh := make(chan error, len(m.plugins))

	for _, p := range m.plugins {
		if p.Hooks.BuildDone == nil {
			continue
		}
		wg.Add(1)
		go func(plug *Plugin) {
			defer wg.Done()
			// Each plugin gets a context with its own PluginConfig.
			pCtx := &BuildDoneContext{
				Config:       ctx.Config,
				PluginConfig: m.pluginConfig(plug.Name, ctx.Config),
				OutputDir:    ctx.OutputDir,
				Pages:        ctx.Pages,
				Collections:  ctx.Collections,
				Site:         ctx.Site,
				warnings:     ctx.warnings,
			}
			if err := plug.Hooks.BuildDone(pCtx); err != nil {
				errCh <- fmt.Errorf("plugin %q BuildDone: %w", plug.Name, err)
			}
		}(p)
	}

	wg.Wait()
	close(errCh)

	// Return first error if any.
	for err := range errCh {
		return err
	}
	return nil
}

// Warnings returns all warnings collected during BuildDone.
func (m *Manager) Warnings() []engine.ValidationWarning {
	return nil // warnings are stored on BuildDoneContext, accessed by caller
}

func (m *Manager) pluginConfig(name string, cfg *config.SiteConfig) map[string]any {
	if cfg.Plugins.Config == nil {
		return nil
	}
	return cfg.Plugins.Config[name]
}
