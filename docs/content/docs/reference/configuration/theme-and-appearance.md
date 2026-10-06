---
title: Theme and Appearance
description: "Theme, table of contents, icon, and link prefetch settings in sarde.yaml"
sidebar:
  order: 3
---

Choose the theme and color preset, control the table of contents, register icon sets, and tune link prefetching.

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
| `font_family` | string | `""` | Base font family CSS value. A single family name is quoted and given a generic fallback, see [Shortcut fields](/guides/themes-and-styling/#shortcut-fields). |
| `font_mono` | string | `""` | Monospace font family CSS value. |
| `font_heading` | string | `""` | Font family CSS value for `h1` to `h6`. Sets the `font-heading` token. Headings use `font_family` when unset. |
| `font_scale` | number | `1` | Multiplier for the `text-xs` to `text-5xl` size scale, from `0.5` to `2`. Sets the `text-scale` token. |
| `web_fonts` | string | `""` | Loads the theme's fonts that are in the Google Fonts library and not bundled. Valid values: `google` (Google Fonts), `bunny` (Bunny Fonts). Empty loads none. The service receives each visitor's IP address. See [Web fonts](/guides/themes-and-styling/#web-fonts). |
| `code_light` | string | `""` | Syntax highlighting theme for light mode. |
| `code_dark` | string | `""` | Syntax highlighting theme for dark mode. |
| `date_format` | string | `"short"` | Display format for the "last updated" date. See below. |

### Date format

`theme.date_format` accepts three preset names or any [Go layout string](https://pkg.go.dev/time#pkg-constants):

| Value | Renders as (English) |
|-------|-----------|
| `short` (default) | Jan 2, 2006 |
| `long` | January 2, 2006 |
| `iso` | 2006-01-02 |
| Any other value | Used verbatim as a Go layout, e.g. `2006/01/02` |

The `short` and `long` presets are **locale-aware**: on a multilingual site, each page renders the date in its own language using CLDR data, so a French page shows "1 juin 2025" while the English page shows "Jun 1, 2025". Around 30 common languages are supported out of the box; a language without built-in data falls back to the English format. `iso` is locale-independent, and a custom Go layout always renders English month names.

This controls **only** the "last updated" date rendered by the [LastUpdated component](/reference/ui-components#lastupdated). Other dates in the theme, such as blog post dates and list-page dates, use formats fixed by their templates; override those templates to change them.

The `datetime` attribute on the emitted `<time>` element is always ISO 8601 regardless of this setting, so the markup stays machine-readable. To change where the timestamp comes from rather than how it looks, see [`build.last_updated`](/reference/configuration/build-and-output/#last-updated-strategy).

## `toc`

Global table of contents *display* settings. These control which extracted headings appear in the TOC sidebar. Per-collection overrides are available under [`collections.<name>.toc`](/reference/configuration/content/#collections-name-toc). For controlling which headings are *extracted* (IDs, anchor links, link validation), see [`markdown.toc`](/reference/configuration/content/#markdown-toc).

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `enabled` | bool | `true` | Show the table of contents panel. |
| `min_level` | int | `2` | Minimum heading level to display. Range: 1-6. Must be &le; `max_level`. |
| `max_level` | int | `4` | Maximum heading level to display. Range: 1-6. |

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

## `prefetch`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `enabled` | bool | `true` | Enable link prefetching for faster navigation. |
| `strategy` | string | `"hover"` | Prefetch trigger strategy. `hover` prefetches on mouse hover. `visible` prefetches links visible in the viewport. `idle` prefetches during idle time. |
| `delay` | int | `300` | Delay in milliseconds before prefetching starts (for `hover` strategy). Min: 0. |
