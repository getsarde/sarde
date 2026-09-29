---
title: Configuration
description: "Every sarde.yaml setting, grouped by topic, and how the five configuration layers override each other"
sidebar:
  order: 1
  attrs:
    open: "true"
---

Sarde reads its configuration from a `sarde.yaml` file at the project root. Every key is optional: a key you leave out uses its default. Start from the [Examples](/reference/configuration/examples/), then look up individual keys on the topic pages.

## Settings by topic

Each topic page documents the `sarde.yaml` keys listed next to it:

| Page | Keys |
| --- | --- |
| [Examples](/reference/configuration/examples/) | Complete `sarde.yaml` files for common kinds of site, and every key with its default |
| [Site and Branding](/reference/configuration/site-and-branding/) | `site`, `social`, `header`, `footer`, `head`, `homepage` |
| [Theme and Appearance](/reference/configuration/theme-and-appearance/) | `theme`, `toc`, `icons`, `prefetch` |
| [Content](/reference/configuration/content/) | `content`, `collections`, `taxonomies`, `permalinks`, `markdown`, `i18n` |
| [Build and Output](/reference/configuration/build-and-output/) | `build`, `images`, `search`, `analytics`, `llms_txt`, `security`, `server` |
| [Plugins and Checks](/reference/configuration/plugins-and-checks/) | `plugins`, `link_validation`, `content_lint` |
| [Deploy and Redirects](/reference/configuration/deploy-and-redirects/) | `deploy`, `redirects` |
| [Environment Variables](/reference/configuration/environment-variables/) | `SARDE_*` variables that override any of the above |

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

To see how Sarde resolved each content collection (its sort order, layout, and sidebar settings, and whether each value was inferred from the directory name or set in `sarde.yaml`), run [`sarde effective-config`](/reference/cli-commands#effective-config).
