---
title: Writing Plugins
description: "Author custom Sarde plugins using the four lifecycle hooks: ConfigSetup, ContentLoaded, BeforeRender, BuildDone."
sidebar:
  order: 3
---

Plugins extend the build with custom logic: generating files, injecting scripts, adding template functions, or reacting to content. A plugin is a Go struct with a name and a set of optional lifecycle hooks.

```go
type Plugin struct {
    Name  string
    Hooks PluginHooks
}

type PluginHooks struct {
    ConfigSetup   func(ctx *ConfigSetupContext) error
    ContentLoaded func(ctx *ContentLoadedContext) error
    BeforeRender  func(ctx *BeforeRenderContext) error
    BuildDone     func(ctx *BuildDoneContext) error
}
```

Hooks are optional. Implement only the ones a plugin needs; the rest are left as `nil`.

## Lifecycle hooks

The four hooks run in a fixed order during a build. Each receives a typed context object exposing the capabilities available at that stage.

### `ConfigSetup`

Runs once after config load, before content discovery. Executes serially.

Context provides:

- `Config` -- the site configuration
- `PluginConfig` -- this plugin's config block from `plugins.config.<name>` in `sarde.yaml`
- `Set` / `Get` -- a shared key-value store, persisted across hooks within a build
- `AddTemplateFunc(name, fn)` -- register a custom template function

### `ContentLoaded`

Runs after content parsing and collection assembly, before rendering. Executes serially.

Context provides:

- `Config`, `PluginConfig`, `Set` / `Get`
- `Collections` -- all built collections
- `Pages` -- a pointer to the page list, appendable
- `InjectPage(page)` -- add a virtual page to the build

### `BeforeRender`

Runs once per page, immediately before template rendering. Executes serially, one page at a time.

Context provides:

- `Page`, `Site`, `Resolver`
- `RouteData` -- inject styles, scripts, and inline scripts for this page
- `PluginConfig`, `Set` / `Get`
- `ResolveURL(relPath, lang, version)`, `AbsURL(...)`, `BaseURL()`

### `BuildDone`

Runs once after all output has been written. Executes in parallel, one goroutine per plugin.

Context is thread-safe and provides:

- `Config`, `PluginConfig`, `Set` / `Get`
- `OutputDir`, `Pages`, `Collections`, `Site`, `Resolver`, `PageIndex`
- `DevMode`, `Incremental`, `ChangedPages`
- `WriteFile(relPath, data)` -- write a file under the output directory, thread-safe and tracked for orphan pruning
- `Log(message)` -- log a message prefixed with the plugin name
- `AddWarning(w)` -- add a build warning

## Example: a minimal plugin

A plugin implementing only `BuildDone`, generating `robots.txt`:

```go
func newRobotsPlugin(cfg map[string]any) *Plugin {
    return &Plugin{
        Name: "robots",
        Hooks: PluginHooks{
            BuildDone: func(ctx *BuildDoneContext) error {
                body := "User-agent: *\nAllow: /\n"
                if ctx.Site.BaseURL != "" {
                    sitemapURL := ctx.AbsURL("/sitemap.xml", "", "")
                    body += "Sitemap: " + sitemapURL + "\n"
                }
                ctx.WriteFile("robots.txt", []byte(body))
                ctx.Log("Generated robots.txt")
                return nil
            },
        },
    }
}
```

## Registration

Built-in plugins are registered as factory functions keyed by name. Plugins that need embedded vendor assets (KaTeX, Mermaid, announcements, social cards) live in their own subpackages and are wired in separately to avoid import cycles.

Enable a plugin by adding its name to `plugins.enabled` in `sarde.yaml`. Per-plugin configuration lives under `plugins.config.<name>`, and is available in every hook's context as `PluginConfig`.

## Client-side plugins

Client-side plugins ship browser JS/CSS instead of Go code, and use a declarative manifest rather than lifecycle hooks. Each plugin has a `manifest.yaml`:

- `description` -- what the plugin does
- `inject_when` -- a named rule controlling which pages include the plugin
- `assets.css` / `assets.js` -- paths to the plugin's CSS and JS files

All enabled client-side plugins are concatenated into a single fingerprinted CSS/JS bundle per build. Their merged configs are serialized into a `window.__SARDE__.pluginConfig` JSON blob available to the client scripts.

### `inject_when` rules

| Rule | Injects on |
|------|-----------|
| `always` | Every page |
| `has_sidebar` | Pages with sidebar navigation |
| `has_toc` | Pages with a table of contents and headings |
| `has_headings` | Pages with any headings |
| `has_code_blocks` | Pages with code blocks |
| `has_images` | Pages with images |
| `has_prev_next` | Pages with prev/next navigation |
| `is_content_page` | Regular content pages, not section indexes |
| `has_updated` | Pages with a last-updated date |
