---
title: Announcements
description: "Render dismissible announcement banners with scheduling and page targeting"
sidebar:
  order: 30
---

Renders dismissible announcement banners at the bottom of every page. Supports multiple announcements, three display modes, date scheduling, page targeting, and i18n message resolution. Disabled by default.

## Enable the plugin

`sarde.yaml`
```yaml
plugins:
  enabled:
    - search
    - seo
    - sitemap
    # ... other default plugins
    - announcements
```

## Define announcements

Each announcement is an entry in the `items` list under the plugin config.

`sarde.yaml`
```yaml
plugins:
  config:
    announcements:
      items:
        - id: maintenance
          message: "The documentation is being updated. Some pages may be incomplete."
          type: warning
          active: true
          dismissible: true
```

→ A yellow warning banner appears at the bottom of every page with a dismiss button.

<!-- SCREENSHOT: announcement-warning-banner - a warning announcement banner with dismiss button -->

## Item options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `id` | String | `"default"` | Unique identifier. Used for dismiss state in `localStorage`. |
| `message` | String | `""` | Banner text. Can be a literal string or an i18n key (see below). |
| `type` | String | `"info"` | Banner style. One of `info`, `warning`, `success`, `danger`. |
| `active` | Boolean | `true` | Whether this announcement is active. Set to `false` to hide without removing. |
| `dismissible` | Boolean | `true` | Show a dismiss button (×). Dismissed state persists in `localStorage`. |
| `start_date` | String | `""` | ISO 8601 date string. Banner is hidden before this date. |
| `end_date` | String | `""` | ISO 8601 date string. Banner is hidden after this date. |
| `show_on` | List | `["/**"]` | Glob patterns for pages where the banner appears. Defaults to all pages. |
| `hide_on` | List | `[]` | Glob patterns for pages where the banner is suppressed. Takes precedence over `show_on`. |

## Banner types

| Type | Color |
|------|-------|
| `info` | Blue |
| `warning` | Amber |
| `success` | Green |
| `danger` | Red |

All four types support light and dark mode with distinct color palettes.

## Display modes

The `display_mode` option controls how multiple active announcements appear.

`sarde.yaml`
```yaml
plugins:
  config:
    announcements:
      display_mode: stack
```

| Mode | Behavior |
|------|----------|
| `stack` | All active banners are visible at once, stacked vertically. Default. |
| `first` | Only the first active banner is shown. When dismissed, the next one takes its place. |
| `rotate` | Banners cycle automatically at the configured interval. A dot indicator shows the current position. |

### Rotate mode options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `rotate_interval` | Number | `5000` | Rotation interval in milliseconds. Minimum: 500. |
| `show_rotate_indicator` | Boolean | `true` | Show dot indicators below the banner for manual navigation. |

```yaml
plugins:
  config:
    announcements:
      display_mode: rotate
      rotate_interval: 4000
      show_rotate_indicator: true
      items:
        - id: update-v2
          message: "Version 2.0 is now available."
          type: success
        - id: workshop
          message: "Register for the live workshop on Friday."
          type: info
```

→ Two banners rotate every 4 seconds with dot navigation below.

## Date scheduling

Limit an announcement to a time window using `start_date` and `end_date`. Both are optional and use ISO 8601 format. Date checks happen client-side on each page load.

```yaml
- id: holiday
  message: "No office hours December 23 through January 2."
  type: info
  start_date: "2025-12-20"
  end_date: "2025-01-03"
```

## Page targeting

Restrict where an announcement appears using glob patterns. The `show_on` list defines which pages display the banner (default: all pages). The `hide_on` list overrides `show_on` to suppress specific paths.

```yaml
- id: docs-notice
  message: "This section is under revision."
  type: warning
  show_on:
    - /docs/**
  hide_on:
    - /docs/changelog
```

→ The banner appears on all pages under `/docs/` except `/docs/changelog`.

Patterns use `*` for single-segment matching and `**` for multi-segment matching, matched against the page's URL pathname.

## Internationalization

Set the `message` field to an i18n key instead of a literal string. Sarde resolves it against the active language's string table at render time.

`i18n/en.yaml`
```yaml
announcements:
  maint: "Scheduled maintenance on Saturday."
  dismiss: "Dismiss"
```

`i18n/fr.yaml`
```yaml
announcements:
  maint: "Maintenance prévue samedi."
  dismiss: "Fermer"
```

`sarde.yaml`
```yaml
plugins:
  config:
    announcements:
      items:
        - id: maint
          message: announcements.maint
```

The dismiss button label also uses `announcements.dismiss` from the string table. If no string table is available, the raw key is displayed as-is.

## Template function

The plugin registers an `announcementBanner` template function. Call it in a layout template to control where banners appear:

```go
{{ announcementBanner }}
```

The default theme already calls this function in the base layout. Override placement by ejecting the base template and moving the call.

## Edge cases

- Invalid `type` values fall back to `info`.
- If all items have `active: false` or all are dismissed, no container element is rendered.
- In rotate mode, the rotation pauses on hover and focus (WCAG 2.2.2 compliance). It also pauses when `prefers-reduced-motion` is active.
- Dot indicators support keyboard navigation with arrow keys, ::kbd[Home], and ::kbd[End].
- Banners are hidden in print output.
- Dismissal state is stored per announcement `id` in `localStorage`. Changing an `id` resets the dismissed state for that banner.
