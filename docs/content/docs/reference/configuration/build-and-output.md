---
title: Build and Output
description: "Build, image, search, analytics, llms.txt, security, and dev server settings in sarde.yaml"
sidebar:
  order: 5
---

Control the build output, image processing, the search index, analytics, `llms.txt`, security headers, and the dev server.

## `build`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `output` | string | `"dist"` | Output directory for the built site. **Required.** |
| `base_path` | string | `""` | URL base path for sites hosted in a subdirectory (e.g., `/docs`). |
| `clean` | bool | `true` | Delete the output directory before each build. |
| `sitemap` | bool | `true` | Generate `sitemap.xml`. |
| `minify` | bool | `false` | Minify HTML output. |
| `last_updated` | string | `"git"` | Strategy for page "last updated" timestamps. `git` uses each file's last commit date, falling back to file modification time outside a git repository. `mtime` always uses file modification time. `false`, `off`, or `none` disables it. See [the caveats below](#last-updated-strategy). |
| `feed` | bool | `true` | Generate RSS and Atom feeds for blog collections. |
| `drafts` | bool | `false` | Include draft pages in the build output. |
| `future` | bool | `false` | Include pages with future `publish_date` values. |
| `expired` | bool | `false` | Include pages past their `expiry_date`. Pages are otherwise dropped from the build once that date passes. |
| `parallel` | bool | `true` | Enable parallel rendering. |
| `cache` | bool | `true` | Reuse rendered pages and processed images across builds. See [Build caching](#build-caching). |

### Build caching

Rendered pages are cached under `.cache/pages` and processed image variants under `.cache/images`, both relative to the project root. Add `.cache/` to `.gitignore`.

Page cache entries are keyed by content hash plus a renderer fingerprint, so editing a page or changing a setting that affects rendering invalidates only the affected entries. Upgrading Sarde can invalidate the whole cache when the renderer changes shape.

Set `cache: false` to render everything on every build. Deleting `.cache/` by hand has the same one-time effect. Caching does not affect link validation: targets and anchors are re-checked on every full build even for cached pages.

### Last-updated strategy

`git` is the default because `mtime` is unreliable wherever sites are usually published. A CI job clones the repository fresh, which writes every file at the same instant, so `mtime` reports the checkout time and every page claims to have changed on every deploy.

Sarde resolves commit times with a single pass over git history for the whole content directory, so the cost is a few tens of milliseconds regardless of page count.

Three caveats are worth knowing:

- **Shallow clones lose history:** with `fetch-depth: 1`, pages whose last change predates the shallow boundary fall back to `mtime`. Use `fetch-depth: 0`. Sarde emits a build warning when it detects a shallow clone.
- **Commit time is a proxy for content change:** a formatting sweep or lint pass bumps the date on every file it touches, even when nothing reader-visible changed. Set `updated` in [frontmatter](/reference/frontmatter#core-fields) to state the real date explicitly.
- **Uncommitted edits keep their last committed date:** during `sarde dev` a page's timestamp does not move until you commit, which is correct but can look like a bug while writing.

Outside a git repository, or when git is unavailable, Sarde falls back to `mtime` and emits one build warning explaining the consequence.

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
| `enabled` | bool | `true` | Enable site search. `false` removes the header button, the modal, the runtime script and the index. |
| `provider` | string | `"orama"` | Search engine provider. Only `orama` is accepted. |

## `analytics`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `provider` | string | `""` | Analytics provider name. |
| `site_id` | string | `""` | Site/property ID for the analytics provider. |
| `script` | string | `""` | Custom analytics script URL or path. |

## `llms_txt`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `enabled` | bool | `true` | Generate `llms.txt` file for LLM consumption. |
| `include_blog` | bool | `true` | Include blog posts in `llms.txt`. |

## `security`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `blocked_href_schemes` | list of string | `["javascript:", "data:", "vbscript:"]` | URL schemes blocked in rendered links. Links using these schemes are stripped. |

## `server`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `host` | string | `""` | Dev server bind address. |
| `port` | int | `4727` | Dev server port. Range: 1-65535. |
| `live_reload` | bool | `true` | Enable WebSocket-based live reload during development. |
