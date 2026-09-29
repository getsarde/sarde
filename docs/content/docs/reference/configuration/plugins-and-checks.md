---
title: Plugins and Checks
description: "Plugin, link validation, and content lint settings in sarde.yaml"
sidebar:
  order: 6
---

Enable and configure plugins, and set how strictly the build checks links and content.

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
    - reading_progress
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
| `tabs_marker_syntax` | bool | `true` | Flag `:::tabs` blocks whose panels will silently collapse into one, such as `=== Label` instead of `== Label`, or a `::tab[...]` directive that does not exist. |
