---
title: Reading Preferences
description: "Let readers adjust font, size, spacing, alignment, and content width"
sidebar:
  order: 18
---

Adds a floating panel where readers can adjust font family, font size, spacing, text alignment, and content width. Preferences persist across pages via `localStorage`. Disabled by default.

## Enable the plugin

`sarde.yaml`
```yaml
plugins:
  enabled:
    - search
    - seo
    - sitemap
    # ... other default plugins
    - reading_preferences
```

## How it works

A floating button with a typography icon appears in the bottom-right corner. Clicking it opens a panel with the following controls:

| Control | Range | Default |
|---------|-------|---------|
| Font family | Sans, Serif, Mono, Dyslexic | Sans |
| Font size | 75%–300% (configurable) | 100% |
| Letter spacing | -0.5px to 3px | 0px |
| Line spacing | 1.2× to 2.4× | 1.85× |
| Paragraph spacing | 0.5em to 3em | 1.25em |
| Text alignment | Left, Justified | Left |
| Content width | 600px to 1400px | 768px |

Changes apply immediately to the content area. Preferences are saved to `localStorage` and restored on subsequent visits.

<!-- SCREENSHOT: reading-preferences-panel — the reading preferences panel open with font and spacing controls -->

Close the panel by clicking outside it, pressing ::kbd[Esc], or clicking the toggle button again.

## Configuration

`sarde.yaml`
```yaml
plugins:
  config:
    reading_preferences:
      min_font_size: 75
      max_font_size: 300
      show_width_control: true
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `min_font_size` | Number | `75` | Smallest font size readers can select (percentage). Range: 25 to 200. |
| `max_font_size` | Number | `300` | Largest font size readers can select (percentage). Range: 100 to 500. |
| `show_width_control` | Boolean | `true` | Show the content width slider. Set to `false` to hide it. |

Individual controls (font family, letter spacing, line spacing, paragraph spacing, alignment) cannot be disabled separately.

## Injection rule

This plugin activates on pages with a sidebar (`has_sidebar`), which includes the `docs` and `wide` layouts. Pages using other layouts receive no plugin config or behavior.

## Edge cases

- Selecting the "Dyslexic" font family loads the Open Dyslexic font from an external CDN (`fonts.cdnfonts.com`). This requires an internet connection and makes a third-party network request.
- Changing `min_font_size` or `max_font_size` in config retroactively clamps any previously saved out-of-range font size on the next page load.
- When `show_width_control` is `false`, any previously saved width preference is preserved in `localStorage` but not applied. Turning the option back on restores the saved width.
- The content width setting also applies while Focus Mode is active. The reading-preferences width takes precedence over Focus Mode's default centered width. Both are desktop-only, so this interaction only arises at 1024px and above.
- Paragraph spacing adjustments do not apply inside elements marked with the `not-content` class (e.g., certain callout blocks).
- The "Reset" button restores all controls to their defaults and saves the reset state.
- Below 1024px the toggle button and panel are hidden and saved preferences are not applied, so the page renders at its default typography. The values stay in `localStorage` and reapply as soon as the viewport is wide enough, including when the reader resizes or rotates across the breakpoint without reloading. This keeps a preference set on a desktop from following the reader to a phone where there is no control to undo it.
- The panel and toggle button are hidden in print output.
