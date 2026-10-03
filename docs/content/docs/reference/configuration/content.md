---
title: Content
description: "Content, collections, taxonomies, permalinks, Markdown, and i18n settings in sarde.yaml"
sidebar:
  order: 4
---

Control how Sarde finds content, groups it into collections and taxonomies, builds URLs, renders Markdown, and serves multiple languages.

## `content`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `dir` | string | `"content"` | Content source directory. **Required.** |
| `summary_length` | int | `70` | Auto-generated summary length in words. Min: 1. |

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
| `collapse_level` | int | - | Depth at which sidebar groups start collapsed. Sections at this depth or deeper are collapsed by default. Requires `collapsible: true`. Also settable per collection in [`sidebar.yaml`](/guides/navigation-and-sidebar#overrides-with-sidebar-yaml), which takes precedence. |
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

### `collections.<name>.labs`

Applies to [labs collections](/teaching/labs/).

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `label` | string | `"Lab"` | Word used in the per-page badge, rendered as `<label> <number>`. Set to `"Exercise"` or `"Activity"` to relabel. The progress bar always reads "Step". |

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
| `id` | string | - | Unique version identifier (e.g., `"v2"`). **Required:** must be unique across all versions. |
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

## `permalinks`

A map of collection name to permalink pattern. Patterns support placeholders like `:slug`, `:year`, `:month`, `:day`, `:title`.

```yaml
permalinks:
  blog: /blog/:year/:month/:slug
  docs: /docs/:slug
```

## `markdown`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `hard_wraps` | bool | `false` | Render every single newline inside a paragraph as a line break. Off by default, so prose wrapped in the source reflows as one paragraph. Explicit breaks (two trailing spaces, or a trailing backslash) work either way. |

Math and diagram rendering are controlled by the [KaTeX](/plugins/katex/) and
[Mermaid](/plugins/mermaid/) plugins, not by this section. Add or remove them in
[`plugins.enabled`](/reference/configuration/plugins-and-checks/#plugins).

### `markdown.toc`

Controls which heading levels are *extracted* during the build: ID injection, anchor links, TOC entries, search index anchors, and link validation targets. Headings outside this range are left untouched. This is separate from the display-level [`toc`](/reference/configuration/theme-and-appearance/#toc) setting, which filters which extracted headings appear in the sidebar widget.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `min_heading_level` | int | `2` | Minimum heading level to extract. Range: 1-6. Must be &le; `max_heading_level`. |
| `max_heading_level` | int | `4` | Maximum heading level to extract. Range: 1-6. |

### `markdown.asides`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `style` | string | `"classic"` | Visual style for `:::note` / `:::tip` aside blocks. `classic` or `galaxy`. The galaxy style renders asides as rounded cards with an uppercase title and a trailing gradient rule, and swaps the note, tip, and danger icons to match. GitHub-style `gh-*` asides are unaffected. See [Aside](/extensions/aside/#aside-styles). |

### `markdown.codeblocks`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `engine` | string | `""` | Syntax highlighting engine. `nuri` or `chroma`. When empty, defaults to `nuri` (tree-sitter-based). |
| `style` | string | `"class"` | Highlighting output style. Only `class` is supported. |
| `light_theme` | string | `"github-light"` | Light mode highlighting theme name. |
| `dark_theme` | string | `"github-dark"` | Dark mode highlighting theme name. |
| `theme` | string | `""` | Single theme override (applies to both modes). |
| `dark_mode_selector` | string | `"[data-theme=\"dark\"]"` | CSS selector for dark mode scoping. The default matches Sarde's theme toggle. Always applies: a `darkMode` entry in `kazari.config.yaml` is ignored, with a build warning. |

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
