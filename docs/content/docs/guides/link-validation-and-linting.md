---
title: Link Validation and Linting
description: "Configure internal link checking, external URL probing, and content linting policies"
sidebar:
  order: 20
---

Sarde validates internal links during every build and optionally probes external URLs. A separate content linter checks heading structure, image alt text, and other Markdown quality rules. Both produce warnings or errors depending on the configured policy.

## Link checker overview

The link checker validates all internal links, anchor references, and image sources during the build. Every link is recorded in a link graph, resolved within its content dimension (collection, language, version), and classified as OK, broken target, broken anchor, unverified, or external.

The checker runs automatically. No configuration is needed to validate internal links.

## Configurable policies

Each link issue type has its own policy: `"error"` (fails the build), `"warn"` (logs a warning), or `"ignore"` (silent skip).

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
```

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `enabled` | bool | `true` | Enable or disable link validation entirely. |
| `on_broken` | string | `"error"` | Policy for broken internal link targets. |
| `on_broken_anchor` | string | `"error"` | Policy for links to missing heading anchors. |
| `on_relative_links` | string | `"warn"` | Policy for relative links (`./` or `../`). |
| `on_local_links` | string | `"warn"` | Policy for `localhost` / `127.0.0.1` URLs. |
| `on_unverified_internal` | string | `"warn"` | Policy for extension-less internal links that did not resolve in the current lane. |
| `check_anchors` | bool | `true` | Verify that `#fragment` targets exist as heading IDs on the target page. |
| `check_images` | bool | `true` | Validate image `src` paths. |
| `same_site_policy` | string | `"ignore"` | Policy for links to the site's own absolute URL. |
| `site_root_escape_prefix` | string | `"site:"` | Prefix that routes a link to the site root, bypassing lane logic (e.g., `site:/pricing`). Set to `""` to disable. |
| `exclude` | string[] | `[]` | Glob patterns for link destinations to skip. |
| `fail_build` | bool | `false` | When `true`, any link issue (regardless of policy) fails the build. |

## External URL probing

External link checking is opt-in. Enable it to probe `http://` and `https://` URLs for reachability:

```yaml
link_validation:
  external:
    check: true
    concurrency: 8
    timeout: "10s"
    cache: ".sarde/linkcache.json"
    cache_ttl: "72h"
    on_broken: "warn"
    ignore: []
    method: "head-then-get"
```

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `external.check` | bool | `false` | Enable external URL probing. |
| `external.concurrency` | int | `8` | Maximum concurrent HTTP requests. |
| `external.timeout` | string | `"10s"` | Per-request timeout (Go duration format). |
| `external.cache` | string | `".sarde/linkcache.json"` | Path to the cache file for external results. |
| `external.cache_ttl` | string | `"72h"` | How long cached results are considered fresh. |
| `external.on_broken` | string | `"warn"` | Policy for unreachable external URLs. |
| `external.ignore` | string[] | `[]` | URL glob patterns to skip. |
| `external.method` | string | `"head-then-get"` | HTTP method: `"head-then-get"` tries HEAD first, falls back to GET on 403/405. `"head"` or `"get"` use a single method. |

Results are cached in `.sarde/linkcache.json`. URLs that returned a successful status within the TTL window are not re-probed.

## `sarde check-links` CLI

Run link validation standalone without a full build:

```bash
sarde check-links
```

| Flag | Default | Description |
|------|---------|-------------|
| `--strict` | `false` | Treat all link issues as errors (exit code 1). |
| `--external` | `false` | Also probe external URLs. |
| `--report` | `""` | Report format: `pretty`, `json`, `github-actions`. Uses config default if empty. |
| `--base-path` | `""` | Override URL base path (e.g., `/docs/`). |
| `--content` | `""` | Override content directory path. |

The command is aliased as `sarde check` for backward compatibility.

Result: A summary line reports checked links, lanes, broken targets, broken anchors, and warnings.

## Content lint rules

The content linter checks Markdown structure independently of link validation. It runs as a built-in plugin during the build.

```yaml
content_lint:
  enabled: true
  rules:
    heading_max_length: 60
    heading_increment: true
    image_alt_required: true
    no_empty_links: true
    frontmatter_required: []
    tabs_marker_syntax: true
```

| Rule | Type | Default | Description |
|------|------|---------|-------------|
| `heading_max_length` | int | `60` | Maximum heading text length. Set to `0` to disable. |
| `heading_increment` | bool | `true` | Warn when heading levels skip (e.g., `##` followed by `####`). |
| `image_alt_required` | bool | `true` | Warn on images with empty alt text (`![](...)`). |
| `no_empty_links` | bool | `true` | Warn on links with empty text (`[](...)`). |
| `frontmatter_required` | string[] | `[]` | List of frontmatter fields that must be present (e.g., `["title", "description"]`). |
| `tabs_marker_syntax` | bool | `true` | Warn on `:::tabs` blocks whose `== Label` markers are malformed or missing. |

Draft pages are excluded from linting. On incremental rebuilds, only changed pages are re-linted.

## Report formats

Three report formats are available, configurable via `link_validation.report` or the `--report` CLI flag:

| Format | Description |
|--------|-------------|
| `pretty` | Human-readable terminal output with color. Findings grouped by source file. Default. |
| `json` | Machine-readable JSON with findings, coverage, and summary. |
| `github-actions` | GitHub Actions annotations (`::error` and `::warning`). Findings appear inline in pull request diffs. |

```yaml
link_validation:
  report: "github-actions"
```

## `fail_build` option

Set `fail_build: true` to fail the build on any link finding, regardless of individual policy settings:

```yaml
link_validation:
  fail_build: true
```

This is useful in CI pipelines where a non-zero exit code should block deployment.

## Edge cases

- Links prefixed with `site:` (or the configured `site_root_escape_prefix`) resolve at the site root, bypassing collection/language/version lane logic. These links are never validated or flagged.
- The link checker resolves internal links within content dimensions. A link in French v2 docs resolves against French v2 pages. Use `?lang=` or `?version=` query parameters for cross-lane links.
- Static assets (`/img/logo.png`) and the site root (`/`) are never flagged as unverified.
- Content lint warnings are ephemeral in dev mode. On incremental rebuilds, only changed pages are checked. A full build re-lints all pages.
- Link targets and anchors are re-validated on every full build, including for pages served from the page cache. A warm build reports the same coverage as a cold one, and renaming a linked page is caught without editing the file that links to it.
- The `sarde check-links` command runs discovery, parsing, and link resolution without rendering templates or writing output.
