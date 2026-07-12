---
title: Link Validator
description: "Check internal links, anchors, and image references during the build"
sidebar:
  order: 34
  group: Server-Side
---

Checks internal links, anchors, and image references during the build and reports broken targets. Also validates relative links, localhost links, and same-site absolute URLs. Enabled by default.

This plugin uses the global `link_validation` config section in `sarde.yaml`, not the `plugins.config` map.

## Default configuration

`sarde.yaml` (defaults shown)
```yaml
link_validation:
  enabled: true
  on_broken: "error"
  on_broken_anchor: "error"
  on_relative_links: "warn"
  on_local_links: "warn"
  on_unverified_internal: "warn"
  check_anchors: true
  check_images: true
  same_site_policy: "ignore"
  site_root_escape_prefix: "site:"
  report: "pretty"
  exclude: []
  fail_build: false
  external:
    check: false
    concurrency: 8
    timeout: "10s"
    cache: ".sarde/linkcache.json"
    cache_ttl: "72h"
    on_broken: "warn"
    ignore: []
    method: "head-then-get"
```

## Internal link checks

The plugin resolves every internal link against the built page index. A link is broken when:

- The target path does not match any known page or static asset.
- The target anchor (`#section-id`) does not match any heading ID on the target page (when `check_anchors` is `true`).
- An image `src` does not resolve to a known asset (when `check_images` is `true`).

## Policies

Each link category has a configurable policy: `"error"`, `"warn"`, or `"ignore"`.

| Option | Default | Description |
|--------|---------|-------------|
| `on_broken` | `"error"` | Policy for links to pages that do not exist. |
| `on_broken_anchor` | `"error"` | Policy for anchor links (`#id`) that do not match a heading. |
| `on_relative_links` | `"warn"` | Policy for relative links (`./page` or `../page`). |
| `on_local_links` | `"warn"` | Policy for links to `localhost` or `127.0.0.1`. |
| `on_unverified_internal` | `"warn"` | Policy for extension-less internal links that could not be resolved in the current lane. |
| `same_site_policy` | `"ignore"` | Policy for absolute URLs pointing to the site's own domain. `"ignore"` passes them through, `"warn"` or `"error"` flags them. |

## Configuration

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | Boolean | `true` | Enable or disable link validation entirely. |
| `check_anchors` | Boolean | `true` | Validate `#anchor` references against heading IDs. |
| `check_images` | Boolean | `true` | Validate image `src` attributes against known assets. |
| `same_site_policy` | String | `"ignore"` | How to handle absolute URLs to the site's own domain. |
| `site_root_escape_prefix` | String | `"site:"` | Prefix that routes a link to the site root, bypassing lane logic. Links with this prefix are never validated. Set to `""` to disable. |
| `report` | String | `"pretty"` | Output format: `"pretty"`, `"json"`, or `"github-actions"`. |
| `exclude` | List | `[]` | URL glob patterns to skip during validation. |
| `fail_build` | Boolean | `false` | Fail the build when broken links are found. |

## External link checking

External URL validation is opt-in. Enable it to probe URLs with `http://` or `https://` schemes.

```yaml
link_validation:
  external:
    check: true
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `external.check` | Boolean | `false` | Enable external URL probing. |
| `external.concurrency` | Number | `8` | Maximum concurrent HTTP requests. |
| `external.timeout` | String | `"10s"` | HTTP request timeout (Go duration format). |
| `external.cache` | String | `".sarde/linkcache.json"` | Path to the link check result cache file. |
| `external.cache_ttl` | String | `"72h"` | How long cached results remain valid. |
| `external.on_broken` | String | `"warn"` | Policy for broken external links. |
| `external.ignore` | List | `[]` | URL glob patterns to skip for external checks. |
| `external.method` | String | `"head-then-get"` | HTTP method: `"head-then-get"`, `"head"`, or `"get"`. |

## Skipped link types

The following links are never validated:

- `mailto:`, `tel:`, `data:`, `javascript:`, `vbscript:` schemes
- URLs matching `exclude` patterns
- Links with the `site_root_escape_prefix` (default: `site:`)
- External URLs (unless `external.check` is `true`)

## Full builds vs incremental rebuilds

On a full build (`sarde build`), the authoritative link validation runs through the internal link graph (the `internal/links` package). The plugin does not duplicate that work.

On incremental rebuilds (`sarde dev`), only links from changed pages are re-validated by this plugin. Links from unchanged pages pointing to removed headings are not re-flagged until the next full build or until the source file is edited.

## Edge cases

- Query parameters in link targets are stripped before resolution. A link to `/docs/config?v=2#theme` resolves as `/docs/config` with anchor `theme`.
- Same-site URLs with `same_site_policy` set to `"ignore"` (the default) are silently skipped. Setting it to `"warn"` or `"error"` helps catch accidentally absolute internal links.
- The `exclude` and `ignore` lists are aliases. If `exclude` is non-empty, it takes precedence over `ignore`.
- Static assets (images, PDFs, etc.) are resolved against the asset index, not the page index.
