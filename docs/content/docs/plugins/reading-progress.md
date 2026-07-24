---
title: Reading Progress
description: "Show a scroll progress bar and an estimated reading time badge"
sidebar:
  order: 19
---

Adds a thin progress bar below the navigation header that fills as the reader scrolls, and an estimated reading time badge below the page title. Disabled by default.

## Enable the plugin

`sarde.yaml`
```yaml
plugins:
  enabled:
    - search
    - seo
    - sitemap
    # ... other default plugins
    - reading-progress
```

## How it works

The progress bar is fixed directly below the site header. It starts empty and fills from left to right as the reader scrolls down the page, reaching 100% at the bottom.

The reading time badge appears below the page title (or description, if present). It displays an estimated reading time based on the word count of the content area, formatted as "5 min read". Reads under one minute show "Less than 1 min read".

## Configuration

`sarde.yaml`
```yaml
plugins:
  config:
    reading-progress:
      bar_height: 3
      bar_color: "#6366f1"
      use_gradient: true
      show_reading_time: true
      words_per_minute: 200
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `bar_height` | Number | `3` | Bar height in pixels. Range: 1 to 10. |
| `bar_color` | Color | `"#6366f1"` | Base color for the progress bar. |
| `use_gradient` | Boolean | `true` | Apply a hue-rotated gradient derived from `bar_color` instead of a solid fill. |
| `lesson_pages_only` | Boolean | `true` | Only show the progress bar and reading time on content pages (not home or listing pages). |
| `show_reading_time` | Boolean | `true` | Display the estimated reading time badge below the page title. |
| `words_per_minute` | Number | `200` | Average reading speed for the time estimate. Range: 50 to 600. |

## Gradient behavior

When `use_gradient` is `true`, the bar displays a three-stop gradient derived from the configured `bar_color`. The gradient shifts the base hue left and right, keeping the same saturation and lightness. There is no separate gradient color configuration. Setting `use_gradient` to `false` fills the bar with the solid `bar_color`.

## Injection rule

This plugin activates on content pages (`is_content_page`). Home pages, section listing pages, and taxonomy pages do not receive the plugin config.

## Edge cases

- On pages shorter than the viewport, the progress bar stays at 0% width.
- Reading time always rounds up to the next whole minute.
- If no content area is found on the page, the reading time badge is not inserted.
- The progress bar's width transition respects `prefers-reduced-motion` by disabling the animation.
