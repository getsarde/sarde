package plugin

import (
	"fmt"
	"sync"

	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/engine"
)

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
func (m *Manager) RunBeforeRender(cfg *config.SiteConfig, page *engine.Page, rd *engine.RouteData, site *engine.SiteContext, resolver *engine.URLResolver) error {
	for _, p := range m.plugins {
		if p.Hooks.BeforeRender == nil {
			continue
		}
		ctx := &BeforeRenderContext{
			Page:         page,
			RouteData:    rd,
			Site:         site,
			Resolver:     resolver,
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
	sharedMu := &sync.Mutex{}

	for _, p := range m.plugins {
		if p.Hooks.BuildDone == nil {
			continue
		}
		wg.Add(1)
		go func(plug *Plugin) {
			defer wg.Done()
			pCtx := &BuildDoneContext{
				Config:            ctx.Config,
				PluginConfig:      m.pluginConfig(plug.Name, ctx.Config),
				OutputDir:         ctx.OutputDir,
				Pages:             ctx.Pages,
				Collections:       ctx.Collections,
				Site:              ctx.Site,
				Resolver:          ctx.Resolver,
				PageIndex:         ctx.PageIndex,
				ValidationData:    ctx.ValidationData,
				DevMode:           ctx.DevMode,
				Incremental:       ctx.Incremental,
				ProjectDir:        ctx.ProjectDir,
				ChangedPages:      ctx.ChangedPages,
				RemovedPermalinks: ctx.RemovedPermalinks,
				TrackFn:           ctx.TrackFn,
				mu:                sharedMu,
				warnings:          ctx.warnings,
				logger:            ctx.logger,
				pluginName:        plug.Name,
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

func (m *Manager) pluginConfig(name string, cfg *config.SiteConfig) map[string]any {
	if cfg.Plugins.Config == nil {
		return nil
	}
	return cfg.Plugins.Config[name]
}
