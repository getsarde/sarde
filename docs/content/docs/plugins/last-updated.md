---
title: Last Updated
description: "Show a badge with the page's last-modified time, relative or absolute"
sidebar:
  order: 16
  group: Client-Side
---

Displays a "Last updated" badge showing when the page was last modified, using either a relative timestamp ("3 days ago") or an absolute date. Disabled by default.

## Enable the plugin

`sarde.yaml`
```yaml
plugins:
  enabled:
    - search
    - seo
    - sitemap
    # ... other default plugins
    - last-updated
```

## How it works

The badge reads the page's last-modified timestamp, which Sarde resolves through a fallback chain:

1. The frontmatter `updated` field, if set.
2. The file's last git commit time (when `build.last_updated` is `"git"`).
3. The file's filesystem modification time (when `build.last_updated` is `"mtime"`, the default).

Pages without a resolvable timestamp receive no badge.

By default, the badge appears at the bottom of the page above the previous/next navigation. It shows relative time ("3 days ago") with the full date available on hover as a tooltip.

<!-- SCREENSHOT: last-updated-badge — the "Last updated: 3 days ago" badge at the bottom of a page -->

## Configuration

`sarde.yaml`
```yaml
plugins:
  config:
    last-updated:
      position: "bottom"
      use_relative_time: true
      date_format: "short"
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `position` | String | `"bottom"` | `"bottom"` places the badge above the page navigation. `"top"` places it below the title, next to the reading time badge if present. |
| `use_relative_time` | Boolean | `true` | Show relative time ("3 days ago") instead of an absolute date. The full date appears as a hover tooltip regardless. |
| `date_format` | String | `"short"` | Format for absolute dates: `"short"` (Jan 5, 2026), `"long"` (January 5, 2026), or `"iso"` (2026-01-05). |

## Date source configuration

The timestamp source is controlled by the site-wide `build.last_updated` setting, not the plugin itself:

`sarde.yaml`
```yaml
build:
  last_updated: "git"
```

| Value | Behavior |
|-------|----------|
| `"mtime"` (default) | Uses the file's filesystem modification time. |
| `"git"` | Uses the file's last git commit time. Falls back to mtime if the file is untracked or git is unavailable. |
| `"false"` | Disables timestamp resolution entirely. The badge does not appear on any page. |

Override the timestamp for a specific page with the `updated` frontmatter field:

```yaml
---
updated: 2026-03-15
---
```

Suppress the badge on a specific page with `show_updated: false` in frontmatter.

## Injection rule

This plugin activates on pages with a non-zero `updated` timestamp (`has_updated`). Pages without a resolvable timestamp receive no plugin config or behavior.

## Edge cases

- Relative time uses fixed-length approximations (30-day months, 365-day years), not calendar-aware math.
- Absolute dates use the visitor's browser locale for formatting, so month names and punctuation may vary.
- When positioned at the top alongside a reading time badge, both are wrapped in a shared flex row to avoid layout shifts.
- The badge icon is a hardcoded SVG. All user-facing text is set via `textContent` to prevent injection.
