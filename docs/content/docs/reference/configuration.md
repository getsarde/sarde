---
title: Configuration
description: "Complete reference for every sarde.yaml setting, resolved through the five-layer config cascade"
sidebar:
  order: 1
---

Sarde reads its configuration from a `sarde.yaml` file at the project root. Every key documented on this page can appear in that file. Keys not present fall back to their defaults.

## Config cascade

Configuration is resolved through five layers. Each layer overrides the one before it.

| Priority | Layer | Source |
|----------|-------|--------|
| 1 (lowest) | Embedded defaults | Compiled into the Sarde binary |
| 2 | Theme config | `theme.yaml` inside the active theme directory |
| 3 | Project config | `sarde.yaml` at the project root |
| 4 | CLI flags | `--drafts`, `--baseURL`, `--future` |
| 5 (highest) | Environment variables | `SARDE_*` prefixed variables |

Boolean fields use a three-state model internally: unset (nil), explicitly `true`, or explicitly `false`. An unset boolean in a higher layer does not override a value set by a lower layer. Setting a boolean to `false` in `sarde.yaml` explicitly disables it, even if a lower layer enabled it.

Slices (lists) are replaced wholesale, not merged. A non-empty list in a higher layer replaces the entire list from a lower layer. Maps (`redirects`, `permalinks`) are merged per-key: higher layers add or overwrite individual entries without removing others.

## `site`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `title` | string | `"My Site"` | Site title. Appears in the browser tab and header. **Required.** |
| `description` | string | `""` | Site description. Used in meta tags and feeds. Recommended. |
| `url` | string | `""` | Production URL (e.g., `https://example.com`). Used for canonical links, sitemaps, and feeds. Recommended. |
| `language` | string | `"en"` | Default language code (BCP 47). **Required.** |
| `logo` | string or object | - | Site logo. Accepts a single path string (used for both themes) or an object with `light`, `dark`, and `alt` fields. |
| `favicon` | string | `""` | Path to the favicon file relative to `public/`. |
| `edit_url` | string | `""` | Base URL for "Edit this page" links. Append the content file path to this URL. Example: `https://github.com/user/repo/edit/main/content`. |
| `title_delimiter` | string | `"\|"` | Separator between page title and site title in the browser tab. |
| `heading_links` | bool | `true` | Add anchor links to headings. |
| `custom_404` | string | `""` | Path to a custom 404 page template. |

### `site.logo` (object form)

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `light` | string | `""` | Logo path for light mode. |
| `dark` | string | `""` | Logo path for dark mode. |
| `alt` | string | `""` | Alt text for the logo image. |

## `social`

A list of social links displayed in the header and footer.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `label` | string | - | Display label (e.g., `"GitHub"`). |
| `url` | string | - | Full URL to the social profile. |
| `icon` | string | - | Icon name (e.g., `"github"`, `"twitter"`). |

```yaml
social:
  - label: GitHub
    url: https://github.com/user/repo
    icon: github
```

## `theme`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `name` | string | `"default"` | Theme name. |
| `preset` | string | `""` | Color preset. Valid values: `ocean`, `forest`, `rose`, `clean`, `minimal`, `docs`, `academic`. |
| `dark` | bool | `true` | Enable dark mode toggle. |
| `overrides` | map | `{}` | Token overrides for light mode. Map of token name to CSS value. See [Theme Tokens](/reference/theme-tokens). |
| `dark_overrides` | map | `{}` | Token overrides for dark mode only. Same format as `overrides`. |
| `primary_color` | string | `""` | Primary brand color (hex). Shorthand for setting the `--sd-accent` token. |
| `accent_color` | string | `""` | Accent color (hex). Sets `--sd-accent` and auto-derives hover/high/low variants. |
| `font_family` | string | `""` | Base font family CSS value. |
| `font_mono` | string | `""` | Monospace font family CSS value. |
| `code_light` | string | `""` | Syntax highlighting theme for light mode. |
| `code_dark` | string | `""` | Syntax highlighting theme for dark mode. |

## `toc`

Global table of contents *display* settings. These control which extracted headings appear in the TOC sidebar. Per-collection overrides are available under [`collections.<name>.toc`](#collections-name-toc). For controlling which headings are *extracted* (IDs, anchor links, link validation), see [`markdown.toc`](#markdown-toc).

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `enabled` | bool | `true` | Show the table of contents panel. |
| `min_level` | int | `2` | Minimum heading level to display. Range: 1-6. Must be &le; `max_level`. |
| `max_level` | int | `4` | Maximum heading level to display. Range: 1-6. |

## `header`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `search` | bool | `true` | Show the search button in the header. |
| `theme_toggle` | bool | `true` | Show the light/dark mode toggle. |
| `social` | bool | `true` | Show social links in the header. |
| `links` | list | `[]` | Navigation links in the header. |

### Header/footer links

Each entry in `header.links` or `footer.links`:

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `label` | string | - | Display text. |
| `url` | string | - | Link URL. |
| `external` | bool | `false` | Open in a new tab. |

## `footer`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `text` | string | `""` | Footer text (supports Markdown). |
| `links` | list | `[]` | Footer navigation links. Same format as [header links](#header-footer-links). |
| `credits` | bool | `true` | Show "Powered by Sarde" credit line. |

## `head`

Inject custom tags, CSS, and JavaScript into every page's `<head>`.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `tags` | list | `[]` | Custom HTML tags to inject. |
| `custom_css` | list | `[]` | Paths to additional CSS files (relative to `assets/`). |
| `custom_js` | list | `[]` | Paths to additional JavaScript files (relative to `assets/`). |

### Head tags

Each entry in `head.tags`:

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `tag` | string | - | HTML tag name (e.g., `"meta"`, `"link"`). |
| `attrs` | map | - | Tag attributes as key-value pairs. |
| `content` | string | - | Tag inner content (for tags like `<script>`). |

```yaml
head:
  tags:
    - tag: meta
      attrs:
        name: google-site-verification
        content: abc123
  custom_css:
    - custom/styles.css
  custom_js:
    - custom/analytics.js
```

## `build`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `output` | string | `"dist"` | Output directory for the built site. **Required.** |
| `base_path` | string | `""` | URL base path for sites hosted in a subdirectory (e.g., `/docs`). |
| `clean` | bool | `true` | Delete the output directory before each build. |
| `sitemap` | bool | `true` | Generate `sitemap.xml`. |
| `minify` | bool | `false` | Minify HTML output. |
| `last_updated` | string | `"mtime"` | Strategy for page "last updated" timestamps. `git` uses git commit dates. `mtime` uses file modification time. `false`, `off`, or `none` disables it. |
| `feed` | bool | `true` | Generate RSS and Atom feeds for blog collections. |
| `drafts` | bool | `false` | Include draft pages in the build output. |
| `future` | bool | `false` | Include pages with future `publish_date` values. |
| `expired` | bool | - | Include pages past their `expiry_date`. |
| `parallel` | bool | `true` | Enable parallel rendering. |
| `cache` | bool | - | Enable build caching. |

## `markdown`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `katex` | bool | `true` | Enable KaTeX math rendering. Assets are injected only when math syntax is detected on a page. |
| `mermaid` | bool | `true` | Enable Mermaid diagram rendering. Assets are injected only when `sarde-mermaid` code blocks are detected. |
| `cdn` | bool | `false` | Load KaTeX/Mermaid assets from a CDN instead of bundled files. |
| `unsafe` | bool | `false` | Allow raw HTML in Markdown content. |
| `typographer` | bool | `true` | Enable typographic replacements (smart quotes, dashes). |
| `github_alerts` | bool | `true` | Parse GitHub-style alert blocks (`> [!NOTE]`, `> [!TIP]`, etc.). |
| `triple_colon_callouts` | bool | `true` | Parse `:::` container-based callouts/asides. |

### `markdown.toc`

Controls which heading levels are *extracted* during the build: ID injection, anchor links, TOC entries, search index anchors, and link validation targets. Headings outside this range are left untouched. This is separate from the display-level [`toc`](#toc) setting, which filters which extracted headings appear in the sidebar widget.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `min_heading_level` | int | `2` | Minimum heading level to extract. Range: 1-6. Must be &le; `max_heading_level`. |
| `max_heading_level` | int | `4` | Maximum heading level to extract. Range: 1-6. |

### `markdown.codeblocks`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `engine` | string | `""` | Syntax highlighting engine. `nuri` or `chroma`. When empty, defaults to `nuri` (tree-sitter-based). |
| `style` | string | `"class"` | Highlighting output style. Only `class` is supported. |
| `light_theme` | string | `"github-light"` | Light mode highlighting theme name. |
| `dark_theme` | string | `"github-dark"` | Dark mode highlighting theme name. |
| `theme` | string | `""` | Single theme override (applies to both modes). |
| `dark_mode_selector` | string | `"[data-theme=\"dark\"]"` | CSS selector for dark mode scoping. |

## `prefetch`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `enabled` | bool | `true` | Enable link prefetching for faster navigation. |
| `strategy` | string | `"hover"` | Prefetch trigger strategy. `hover` prefetches on mouse hover. `visible` prefetches links visible in the viewport. `idle` prefetches during idle time. |
| `delay` | int | `300` | Delay in milliseconds before prefetching starts (for `hover` strategy). Min: 0. |

## `images`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `widths` | list of int | `[400, 800, 1200]` | Responsive image widths to generate (in pixels). Each value must be at least 1. |
| `formats` | list of string | `["webp"]` | Output image formats. Valid values: `jpeg`, `jpg`, `png`, `webp`, `avif`. |
| `quality` | int | `80` | Compression quality for generated images. Range: 1-100. |
| `placeholder` | string | `"lqip"` | Placeholder strategy while images load. `lqip` (low-quality image placeholder), `blur`, `dominantColor`, or `none`. |
| `max_width` | int | `2400` | Maximum image width in pixels. Images wider than this are downscaled. Min: 1. |
| `lazy_loading` | bool | `true` | Add `loading="lazy"` to images. |
| `dimensions` | bool | `true` | Add `width` and `height` attributes to prevent layout shift. |

## `search`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `enabled` | bool | `true` | Enable site search. |
| `provider` | string | `"orama"` | Search engine provider. Only `orama` is supported. |

## `icons`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `default_prefix` | string | `"lucide"` | Icon set used for names without a prefix. The bundled set is `lucide`. |
| `sets` | list | `[]` | Additional Iconify JSON icon sets to load. |
| `sets_dir` | string | `""` | Directory of additional Iconify `*.json` sets, auto-discovered by filename prefix. |
| `local_dir` | string | `"icons"` | Directory of local `*.svg` files, referenced by filename. |
| `attribution` | string | `""` | Attribution line for icon sets whose license requires it. |
| `render` | string | `"inline"` | Rendering mode. `inline` outputs a full `<svg>` element per use. `sprite` renders one hidden `<symbol>` per unique icon and references it via `<use>`. |

### Icon sets

Each entry in `icons.sets`:

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `prefix` | string | - | Icon set prefix (e.g., `"mdi"`). |
| `file` | string | - | Path to the Iconify JSON file. |

```yaml
icons:
  sets:
    - prefix: mdi
      file: icons/mdi.json
```

## `analytics`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `provider` | string | `""` | Analytics provider name. |
| `site_id` | string | `""` | Site/property ID for the analytics provider. |
| `script` | string | `""` | Custom analytics script URL or path. |

## `plugins`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `enabled` | list of string | See below | List of enabled built-in plugin names. Replaces the default list entirely when set. Does not affect external plugins. |
| `disabled` | list of string | `[]` | Plugin names to turn off, of any type (built-in, client-side, or external). Only removes the named entries; never replaces the rest. |
| `config` | map | `{}` | Per-plugin configuration. Keys are plugin names, values are maps of plugin-specific options. |

Default `enabled` list:

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
```

```yaml
plugins:
  config:
    social_cards:
      skip_if_image: true
    rss:
      limit: 20
    slideviewer:
      always: false
```

```yaml
plugins:
  disabled:
    - social_cards
    - reading-progress
```

## `taxonomies`

Each key under `taxonomies` defines a taxonomy. The value can be a bare string (sets the singular form) or an object with the fields below.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `singular` | string | - | Singular form of the taxonomy name (e.g., `"tag"` for a `tags` taxonomy). |
| `paginate_by` | int | - | Number of items per taxonomy term page. Min: 1. |
| `undefined_tags` | string | - | Behavior when content uses a tag not defined in `data/`. `warn`, `error`, `ignore`, or `create`. |
| `render` | bool | `true` | Generate taxonomy listing pages. |
| `show_tags` | bool | - | Show taxonomy terms on content pages. |

```yaml
taxonomies:
  tags: tag
  categories:
    singular: category
    paginate_by: 20
```

## `collections`

Each key under `collections` defines overrides for a content collection. Collections are auto-detected from directory names under `content/` (e.g., `blog`, `docs`, `slides`). These settings overlay the auto-detected defaults. See [Content and Collections](/guides/content-and-collections/) for the full auto-detection table.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `enabled` | bool | - | Enable or disable this collection. |
| `path` | string | - | Content directory path override. |
| `url_prefix` | string | - | URL prefix override (e.g., `/docs`). |
| `sort` | string | - | Sort order. Format: `<field> <direction>` (e.g., `"date desc"`, `"weight asc"`). |
| `layout` | string | - | Default layout for pages. `default`, `docs`, `splash`, `wide`, `full`, `centered`, `split`, or `presentation`. |
| `permalink` | string | - | Permalink pattern (e.g., `/:slug`). |
| `paginate` | int | - | Items per list page. Min: 1. |
| `feed` | bool | - | Generate RSS/Atom feeds for this collection. |
| `tabs` | bool | - | Enable [tabbed navigation](/guides/tabbed-navigation) for this collection. |
| `i18n_fallback` | string | - | i18n fallback strategy for this collection. `default` (use default language page) or `omit` (hide untranslated pages). |

### `collections.<name>.sidebar`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `collapsible` | bool | - | Allow sidebar groups to collapse. |
| `collapsed_by_default` | bool | - | Start sidebar groups in collapsed state. |
| `collapse_level` | int | - | Depth at which sidebar groups start collapsed. Sections at this depth or deeper are collapsed by default. Requires `collapsible: true`. |
| `max_depth` | int | - | Maximum nesting depth to display. Range: 1-10. |
| `search` | bool | - | Enable sidebar search/filter. |

### `collections.<name>.toc`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `enabled` | bool | - | Show table of contents for this collection. |
| `depth` | int | - | Maximum heading depth. Range: 1-6. Maps to the resolved max heading level. |
| `scroll_highlight` | bool | - | Highlight the current section in the TOC while scrolling. |

### `collections.<name>.prev_next`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `enabled` | bool | - | Show previous/next navigation links. |
| `labels` | list of string | - | Custom labels for the previous and next links (two-element list). |

### `collections.<name>.versioning`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `enabled` | bool | - | Enable versioning for this collection. |
| `last_version` | string | - | ID of the latest stable version. Must match one of the `versions[].id` values. |
| `publish_latest_at_version_url` | bool | - | Publish the latest version content at the versioned URL path as well. |
| `fallback` | string | - | Fallback for pages missing in a version. `default` or `omit`. |
| `versions` | list | - | Version definitions. |

Each entry in `versions`:

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `id` | string | - | Unique version identifier (e.g., `"v2"`). **Required.** Must be unique across all versions. |
| `label` | string | - | Display label (e.g., `"Version 2.0"`). |
| `path` | string | - | Content directory path. Defaults to the `id` value. |
| `banner` | string | `"none"` | Version banner type. `none`, `unmaintained`, or `unreleased`. |
| `redirect` | string | `"same-page"` | Redirect behavior when switching versions. `same-page` (try the same page in the target version) or `root` (go to the version root). |

```yaml
collections:
  docs:
    sort: "weight asc"
    layout: docs
    sidebar:
      collapsible: true
      collapsed_by_default: false
    toc:
      enabled: true
      depth: 4
    versioning:
      enabled: true
      last_version: v2
      versions:
        - id: v2
          label: "2.x"
        - id: v1
          label: "1.x"
          banner: unmaintained
```

## `homepage`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `template` | string | `"hero"` | Homepage template. Available templates: `hero`, `catalog`, `minimal`, `dashboard`, `portfolio`, `landing`, `marketing`, `blog`. |

### `homepage.hero`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `eyebrow` | string | `""` | Small text above the title. |
| `title` | string | `""` | Hero heading text. |
| `subtitle` | string | `""` | Hero subheading text. |
| `background` | string | `"gradient"` | Background style for the hero section. |
| `cta` | object | - | Primary call-to-action button. Has `label` and `url` fields. |
| `secondary_cta` | object | - | Secondary call-to-action button. Same format as `cta`. |
| `stats` | list | - | Statistics to display. Each entry has `value` and `label` fields. |
| `code` | object | - | Code block to display in the hero. Has `title`, `language`, and `body` fields. **Mutually exclusive with `image`.** |
| `image` | object | - | Image to display in the hero. Has `src`, `light`, `dark`, `alt`, and `html` fields. **Mutually exclusive with `code`.** |

```yaml
homepage:
  template: hero
  hero:
    title: "Build fast sites"
    subtitle: "A static site generator that ships as a single binary."
    background: gradient
    cta:
      label: "Get Started"
      url: /docs/start-here/getting-started
    image:
      light: /images/hero-light.svg
      dark: /images/hero-dark.svg
      alt: "Hero illustration"
```

## `i18n`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `default_language` | string | `"en"` | Default language code. |
| `strategy` | string | `"prefix-except-default"` | URL strategy for localized pages. Only `prefix-except-default` is supported (default language has no URL prefix, other languages are prefixed). |
| `fallback` | string | `"default"` | Fallback for untranslated pages. `default` (show the default language version) or `omit` (hide the page). |
| `strict` | bool | `false` | Strict mode. Once enabled by any cascade layer, it cannot be disabled by a higher layer. |
| `languages` | map | `{}` | Language definitions. Each key is a BCP 47 language code. |

### `i18n.languages.<code>`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `name` | string | - | Language display name (e.g., `"Français"`). |
| `title` | string | - | Site title override for this language. |
| `weight` | int | - | Sort weight for language switcher ordering. |
| `dir` | string | `"ltr"` | Text direction. `ltr` or `rtl`. |

```yaml
i18n:
  default_language: en
  languages:
    en:
      name: English
    fr:
      name: "Français"
      title: "Mon Site"
      weight: 2
    ar:
      name: "العربية"
      dir: rtl
      weight: 3
```

## `link_validation`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `enabled` | bool | `true` | Enable the link validation pipeline. |
| `level` | string | `"warn"` | Default severity level. `error`, `warn`, or `ignore`. |
| `on_broken` | string | `"error"` | Policy for broken internal links. `error`, `warn`, or `ignore`. |
| `on_broken_anchor` | string | `"error"` | Policy for broken anchor references. `error`, `warn`, or `ignore`. |
| `report` | string | `"pretty"` | Report output format. `pretty`, `json`, or `github-actions`. |
| `on_relative_links` | string | `"warn"` | Policy for relative links (without leading `/`). `error`, `warn`, or `ignore`. |
| `on_local_links` | string | `"warn"` | Policy for `file://` or absolute local paths. `error`, `warn`, or `ignore`. |
| `on_unverified_internal` | string | `"warn"` | Policy for internal links that cannot be resolved to a known page. `error`, `warn`, or `ignore`. |
| `check_anchors` | bool | `true` | Validate anchor targets (`#id` references). |
| `check_images` | bool | `true` | Validate image `src` paths. |
| `same_site_policy` | string | `"ignore"` | Policy for absolute links pointing to the same site URL. `error`, `warn`, or `ignore`. |
| `site_root_escape_prefix` | string | `"site:"` | Prefix for links that bypass collection/lang/version lane logic (e.g., `site:/pricing`). Set to `""` to disable. |
| `exclude` | list of string | `[]` | Path patterns to exclude from validation. |
| `ignore` | list of string | `[]` | Additional path patterns to ignore during validation. |
| `fail_build` | bool | `false` | Fail the build on link validation errors. |

### `link_validation.external`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `check` | bool | `false` | Enable external URL checking (opt-in). |
| `concurrency` | int | `8` | Maximum concurrent HTTP requests. Min: 1. |
| `timeout` | string | `"10s"` | HTTP request timeout (Go duration format, e.g., `"10s"`, `"30s"`). |
| `cache` | string | `".sarde/linkcache.json"` | Path to the URL check result cache file. |
| `cache_ttl` | string | `"72h"` | Cache entry time-to-live (Go duration format). |
| `on_broken` | string | `"warn"` | Policy for broken external links. `error`, `warn`, or `ignore`. |
| `ignore` | list of string | `[]` | URL glob patterns to skip. |
| `method` | string | `"head-then-get"` | HTTP method for checking. `head-then-get` (try HEAD first, fall back to GET), `head`, or `get`. |

## `content_lint`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `enabled` | bool | `true` | Enable content linting. |

### `content_lint.rules`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `heading_max_length` | int | `60` | Maximum heading length in characters. Min: 1. |
| `heading_increment` | bool | `true` | Require heading levels to increment by one (no skipping from `##` to `####`). |
| `image_alt_required` | bool | `true` | Require alt text on all images. |
| `no_empty_links` | bool | `true` | Flag links with empty text. |
| `frontmatter_required` | list of string | `[]` | Frontmatter fields that must be present on every page (e.g., `["title", "description"]`). |

## `content`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `dir` | string | `"content"` | Content source directory. **Required.** |
| `summary_length` | int | `70` | Auto-generated summary length in words. Min: 1. |

## `deploy`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `provider` | string | `""` | Deployment provider. `github`, `netlify`, `cloudflare`, `vercel`, or `custom`. |
| `branch` | string | `""` | GitHub Pages deployment branch (e.g., `"gh-pages"`). |
| `site_id` | string | `""` | Netlify site ID. |
| `project_name` | string | `""` | Cloudflare Pages project name. |
| `project_id` | string | `""` | Vercel project ID. |
| `command` | string | `""` | Custom deployment command (for `custom` provider). |
| `redirect_format` | string | `""` | Redirect file format. `html`, `netlify`, `vercel`, or `all`. |

## `server`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `host` | string | `""` | Dev server bind address. |
| `port` | int | `4727` | Dev server port. Range: 1-65535. |
| `live_reload` | bool | `true` | Enable WebSocket-based live reload during development. |

## `permalinks`

A map of collection name to permalink pattern. Patterns support placeholders like `:slug`, `:year`, `:month`, `:day`, `:title`.

```yaml
permalinks:
  blog: /blog/:year/:month/:slug
  docs: /docs/:slug
```

## `redirects`

A map of source path to destination URL. Higher cascade layers add entries without removing existing ones.

```yaml
redirects:
  /old-page: /new-page
  /legacy/docs: /docs/start-here/getting-started
```

## `llms_txt`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `enabled` | bool | `true` | Generate `llms.txt` file for LLM consumption. |
| `include_blog` | bool | `true` | Include blog posts in `llms.txt`. |

## `security`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `blocked_href_schemes` | list of string | `["javascript:", "data:", "vbscript:"]` | URL schemes blocked in rendered links. Links using these schemes are stripped. |

## Environment variables

Override configuration with `SARDE_`-prefixed environment variables. These form the highest-priority cascade layer.

### String variables

| Variable | Config key | Description |
|----------|------------|-------------|
| `SARDE_SITE_TITLE` | `site.title` | Site title. |
| `SARDE_SITE_DESCRIPTION` | `site.description` | Site description. |
| `SARDE_SITE_URL` | `site.url` | Production URL. |
| `SARDE_SITE_LANGUAGE` | `site.language` | Default language code. |
| `SARDE_THEME_NAME` | `theme.name` | Theme name. |
| `SARDE_THEME_PRESET` | `theme.preset` | Color preset. |
| `SARDE_BUILD_OUTPUT` | `build.output` | Output directory. |
| `SARDE_BUILD_BASE_PATH` | `build.base_path` | URL base path. |
| `SARDE_ANALYTICS_PROVIDER` | `analytics.provider` | Analytics provider. |
| `SARDE_ANALYTICS_SITE_ID` | `analytics.site_id` | Analytics site ID. |
| `SARDE_SEARCH_PROVIDER` | `search.provider` | Search provider. |
| `SARDE_SERVER_HOST` | `server.host` | Dev server bind address. |

### Boolean variables

Boolean values accept: `true`, `1`, `yes`, `on` (truthy) and `false`, `0`, `no`, `off` (falsy). Case-insensitive. Invalid values are ignored with a warning.

| Variable | Config key | Description |
|----------|------------|-------------|
| `SARDE_BUILD_DRAFTS` | `build.drafts` | Include draft pages. |
| `SARDE_BUILD_MINIFY` | `build.minify` | Minify HTML output. |
| `SARDE_BUILD_CLEAN` | `build.clean` | Clean output directory before build. |
| `SARDE_BUILD_PARALLEL` | `build.parallel` | Enable parallel rendering. |
| `SARDE_MARKDOWN_UNSAFE` | `markdown.unsafe` | Allow raw HTML in Markdown. |
| `SARDE_SEARCH_ENABLED` | `search.enabled` | Enable site search. |
| `SARDE_THEME_DARK` | `theme.dark` | Enable dark mode toggle. |

### Integer variables

Invalid integer values are ignored with a warning.

| Variable | Config key | Description |
|----------|------------|-------------|
| `SARDE_SERVER_PORT` | `server.port` | Dev server port. |
| `SARDE_TOC_MIN_LEVEL` | `toc.min_level` | TOC minimum heading level. |
| `SARDE_TOC_MAX_LEVEL` | `toc.max_level` | TOC maximum heading level. |

Only the most commonly overridden fields support environment variables. For other fields, use `sarde.yaml` or CLI flags.
