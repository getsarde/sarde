---
title: Environment Variables
description: "SARDE_* environment variables that override sarde.yaml settings"
sidebar:
  order: 8
---

Override configuration with `SARDE_`-prefixed environment variables. These form the highest-priority cascade layer.

## String variables

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

## Boolean variables

Boolean values accept: `true`, `1`, `yes`, `on` (truthy) and `false`, `0`, `no`, `off` (falsy). Case-insensitive. Invalid values are ignored with a warning.

| Variable | Config key | Description |
|----------|------------|-------------|
| `SARDE_BUILD_DRAFTS` | `build.drafts` | Include draft pages. |
| `SARDE_BUILD_MINIFY` | `build.minify` | Minify HTML output. |
| `SARDE_BUILD_CLEAN` | `build.clean` | Clean output directory before build. |
| `SARDE_BUILD_PARALLEL` | `build.parallel` | Enable parallel rendering. |
| `SARDE_SEARCH_ENABLED` | `search.enabled` | Enable site search. |
| `SARDE_THEME_DARK` | `theme.dark` | Enable dark mode toggle. |

## Integer variables

Invalid integer values are ignored with a warning.

| Variable | Config key | Description |
|----------|------------|-------------|
| `SARDE_SERVER_PORT` | `server.port` | Dev server port. |
| `SARDE_TOC_MIN_LEVEL` | `toc.min_level` | TOC minimum heading level. |
| `SARDE_TOC_MAX_LEVEL` | `toc.max_level` | TOC maximum heading level. |

Only the most commonly overridden fields support environment variables. For other fields, use `sarde.yaml` or CLI flags.
