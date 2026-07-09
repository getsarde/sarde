---
title: Using Plugins
sidebar:
  order: 1
---

Plugins add behavior to the build process or the generated site. Sarde ships with 27 built-in plugins split into two categories: server-side plugins that run during `sarde build`, and client-side plugins that inject JavaScript and CSS into the generated pages.

## Enabling and disabling plugins

Configure the active plugin list in `sarde.yaml` under `plugins.enabled`:

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
Setting `plugins.enabled` replaces the default list entirely. If only `reading-progress` is listed, every other plugin (search, seo, sitemap, etc.) is disabled. Always include the default plugins alongside any additions.
:::

To disable a single default plugin, copy the full default list and remove the unwanted entry.

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

Thirteen plugins are enabled by default. Fourteen more are available but must be added to `plugins.enabled` to activate.

### Enabled by default

| Plugin | Type | Purpose |
|--------|------|---------|
| `search` | Server | Builds the Orama offline search index. |
| `seo` | Server | Generates Open Graph, Twitter Card, and JSON-LD meta tags. |
| `sitemap` | Server | Writes `sitemap.xml`. |
| `robots` | Server | Writes `robots.txt`. |
| `rss` | Server | Generates an RSS 2.0 feed. |
| `atom` | Server | Generates an Atom 1.0 feed. |
| `content_lint` | Server | Checks heading structure, image alt text, and empty links. |
| `link_validator` | Server | Validates internal links, anchors, and images. |
| `redirects` | Server | Generates redirect pages from aliases and config entries. |
| `llms_txt` | Server | Writes `llms.txt` for LLM-friendly site summaries. |
| `katex` | Server | Injects KaTeX runtime assets on pages with math content. |
| `mermaid` | Server | Injects Mermaid runtime assets on pages with diagrams. |
| `social_cards` | Server | Auto-generates Open Graph images (1200x630). |

### Disabled by default

| Plugin | Type | Purpose |
|--------|------|---------|
| `scroll-to-top` | Client | Floating button to scroll back to the top. |
| `copy-section-link` | Client | Click-to-copy anchor links on headings. |
| `external-links` | Client | Marks external links with an icon and opens them in a new tab. |
| `image-lightbox` | Client | Click-to-zoom overlay for images. |
| `keyboard-nav` | Client | Navigate between pages with arrow keys. |
| `focus-mode` | Client | Hides sidebar and TOC for distraction-free reading. |
| `reading-progress` | Client | Progress bar showing scroll position. |
| `search-highlighter` | Client | Highlights search terms on the destination page. |
| `text-highlighter` | Client | User-driven text highlighting with persistence. |
| `last-updated` | Client | Displays the page's last-updated date. |
| `reading-position-memory` | Client | Remembers scroll position across visits. |
| `reading-preferences` | Client | Font size and content width controls. |
| `code-collapsible` | Client | Auto-collapses long code blocks with a toggle. |
| `announcements` | Server | Displays announcement banners with scheduling and i18n. |

## Server-side vs. client-side

**Server-side plugins** run Go code during the build. They write files (feeds, sitemaps, search indexes, social card images), validate content, or inject `<meta>` tags into rendered pages. They produce no client-side JavaScript.

**Client-side plugins** ship as bundled JavaScript and CSS. All enabled client-side plugins are concatenated into a single pair of files (`plugins.<hash>.css` and `plugins.<hash>.js`) for minimal network overhead. Each plugin reads its configuration from an inline script and activates only on pages where its `inject_when` condition is met.

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
