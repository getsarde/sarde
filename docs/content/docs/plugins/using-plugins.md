---
title: Using Plugins
description: "Enable, disable, and configure Sarde's built-in server-side and client-side plugins"
sidebar:
  order: 1
---

Plugins add behavior to the build process or the generated site. Sarde ships with 26 built-in plugins split into two categories: server-side plugins that run during `sarde build`, and client-side plugins that inject JavaScript and CSS into the generated pages. Plugins installed into a project's `plugins/` directory are covered separately in [External Plugins](/plugins/external-plugins/).

## Enabling and disabling plugins

Configure the active built-in plugin list in `sarde.yaml` under `plugins.enabled`:

`sarde.yaml`
```yaml
plugins:
  enabled:
    - search
    - seo
    - sitemap
    - robots
    - rss
    - atom
    - content_lint
    - link_validator
    - redirects
    - llms_txt
    - katex
    - mermaid
    - social_cards
    - reading-progress
    - scroll-to-top
```

:::warning
Setting `plugins.enabled` replaces the default list entirely. If only `reading-progress` is listed, every other plugin (search, seo, sitemap, etc.) is disabled. Always include the default plugins alongside any additions. This applies to built-in and client-side plugins only; [external plugins](/plugins/external-plugins/) are never governed by `plugins.enabled`.
:::

### Disabling individual plugins

To turn off a single plugin without touching the enabled list, add its name to `plugins.disabled`:

`sarde.yaml`
```yaml
plugins:
  disabled:
    - social_cards
    - slideviewer
```

`plugins.disabled` accepts any plugin name: built-in, client-side, or external. It only removes the named entries and never replaces anything, which makes it the simplest way to opt out of a default plugin. A name that matches no known plugin fails config validation, so typos are caught at build time.

External plugins are enabled by their presence in the `plugins/` directory and disabled the same way, through `plugins.disabled`. See [External Plugins](/plugins/external-plugins/#enabling-and-disabling-plugins) for the full model.

## Plugin configuration

Pass per-plugin options under `plugins.config.<name>`:

`sarde.yaml`
```yaml
plugins:
  config:
    rss:
      title: "Course Updates"
      limit: 20
    scroll-to-top:
      threshold: 400
```

Each plugin documents its own configuration options. Options not specified fall back to the plugin's built-in defaults.

## Default plugins

Fourteen plugins are enabled by default. Twelve more are available but must be added to `plugins.enabled` to activate.

### Enabled by default

| Plugin | Type | Purpose |
|--------|------|---------|
| [`search`](/plugins/search) | Server | Builds the Orama offline search index. |
| [`seo`](/plugins/seo) | Server | Generates Open Graph, Twitter Card, and JSON-LD meta tags. |
| [`sitemap`](/plugins/sitemap) | Server | Writes `sitemap.xml`. |
| [`robots`](/plugins/robots) | Server | Writes `robots.txt`. |
| [`rss`](/plugins/feeds) | Server | Generates an RSS 2.0 feed. |
| [`atom`](/plugins/feeds) | Server | Generates an Atom 1.0 feed. |
| [`content_lint`](/plugins/content-lint) | Server | Checks heading structure, image alt text, and empty links. |
| [`link_validator`](/plugins/link-validator) | Server | Validates internal links, anchors, and images. |
| [`redirects`](/plugins/redirects) | Server | Generates redirect pages from aliases and config entries. |
| [`llms_txt`](/plugins/llms-txt) | Server | Writes `llms.txt` for LLM-friendly site summaries. |
| [`katex`](/plugins/katex) | Server | Injects KaTeX runtime assets on pages with math content. |
| [`mermaid`](/plugins/mermaid) | Server | Injects Mermaid runtime assets on pages with diagrams. |
| [`social_cards`](/plugins/social-cards) | Server | Auto-generates Open Graph images (1200x630). |

[SlideViewer](/plugins/slideviewer) is distributed separately as an [external plugin](/plugins/external-plugins) and activates by presence in your project's `plugins/` directory.

### Disabled by default

| Plugin | Type | Purpose |
|--------|------|---------|
| [`scroll-to-top`](/plugins/scroll-to-top) | Client | Floating button to scroll back to the top. |
| [`copy-section-link`](/plugins/copy-section-link) | Client | Click-to-copy anchor links on headings. |
| [`external-links`](/plugins/external-links) | Client | Marks external links with an icon and opens them in a new tab. |
| [`image-lightbox`](/plugins/image-lightbox) | Client | Click-to-zoom overlay for images. |
| [`keyboard-nav`](/plugins/keyboard-nav) | Client | Navigate between pages with arrow keys. |
| [`focus-mode`](/plugins/focus-mode) | Client | Hides sidebar and TOC for distraction-free reading. |
| [`reading-progress`](/plugins/reading-progress) | Client | Progress bar showing scroll position. |
| [`search-highlighter`](/plugins/search-highlighter) | Client | Highlights search terms on the destination page. |
| [`text-highlighter`](/plugins/text-highlighter) | Client | User-driven text highlighting with persistence. |
| [`reading-position-memory`](/plugins/reading-position-memory) | Client | Remembers scroll position across visits. |
| [`reading-preferences`](/plugins/reading-preferences) | Client | Font size and content width controls. |
| [`announcements`](/plugins/announcements) | Server | Displays announcement banners with scheduling and i18n. |

## Server-side vs. client-side

**Server-side plugins** run Go code during the build. They write files (feeds, sitemaps, search indexes, social card images), validate content, or inject `<meta>` tags into rendered pages. They produce no client-side JavaScript.

**Client-side plugins** ship as bundled JavaScript and CSS. All enabled client-side plugins are concatenated into a single pair of files (`plugins.<hash>.css` and `plugins.<hash>.js`) for minimal network overhead. Each plugin reads its configuration from an inline script and activates only on pages where its `inject_when` condition is met.

A plugin that is not enabled contributes nothing to the bundle, so a site never downloads code for plugins it has not turned on. Enabling a different set of plugins changes the bundle contents and therefore its hash. If no client-side plugin is enabled, no bundle is written and no page references one.

Because `plugins.config` only reaches plugins that are actually active, a config block for a plugin that is not enabled is ignored. The build warns when this happens:

```
WARN  plugin  sarde.yaml: plugins.config.keyboard-nav
      plugin "keyboard-nav" is not in plugins.enabled, configuration ignored
```

A config block naming no known plugin (usually a typo) warns as `unknown plugin`. External plugins are exempt from both warnings, since they are enabled by their presence under `plugins/` rather than by `plugins.enabled`.

## Page injection rules

Client-side plugins declare when their configuration is injected into a page. The plugin's JavaScript and CSS bundle is always loaded, but the plugin only activates on pages matching its condition:

| Rule | Activates on |
|------|-------------|
| `always` | Every page. |
| `has_code_blocks` | Pages containing fenced code blocks. |
| `has_images` | Pages containing images. |
| `has_sidebar` | Pages with a sidebar (docs layout). |
| `has_toc` | Pages with a table of contents. |
| `has_headings` | Pages with headings. |
| `has_prev_next` | Pages with previous/next navigation links. |
| `is_content_page` | All content pages (excludes list and taxonomy pages). |
| `has_updated` | Pages with an `updated` date in frontmatter. |

## Lifecycle hooks

Server-side plugins hook into four stages of the build pipeline, executed in order:

| Hook | When | Concurrency |
|------|------|-------------|
| `ConfigSetup` | After config load, before content discovery. | Serial. |
| `ContentLoaded` | After content parsing, before page assembly. | Serial. |
| `BeforeRender` | Once per page, before template rendering. | Serial per page. |
| `BuildDone` | After all files are written. | Parallel across plugins. |

Plugins run in the order they appear in `plugins.enabled` for serial hooks. `BuildDone` runs all plugins concurrently.
