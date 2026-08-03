---
title: Scroll to Top
description: "A floating button that scrolls back to the top, with an optional progress ring"
sidebar:
  order: 20
---

A floating button that scrolls the page back to the top when clicked. Optionally displays a circular progress ring showing how far down the page the reader has scrolled. Disabled by default.

## Enable the plugin

`sarde.yaml`
```yaml
plugins:
  enabled:
    - search
    - seo
    - sitemap
    # ... other default plugins
    - scroll_to_top
```

## How it works

The button appears centered at the bottom of the viewport after the reader scrolls past the configured threshold (default: 300 pixels from the top). It automatically hides when the reader scrolls near the page footer so it never overlaps footer content. Clicking the button scrolls smoothly back to the top.

The horizontal position is configurable: `center` (default), `left`, or `right`.

When `show_progress_ring` is enabled, a circular SVG ring around the button fills as the reader scrolls, providing a visual indicator of scroll position.

<!-- SCREENSHOT: scroll-to-top-with-ring - the scroll-to-top button with progress ring partially filled -->

## Configuration

`sarde.yaml`
```yaml
plugins:
  config:
    scroll_to_top:
      threshold: 300
      position: center
      show_progress_ring: true
      smooth_scroll: true
      border_radius: 15
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `threshold` | Number | `300` | Scroll distance in pixels before the button appears. |
| `position` | Select | `center` | Horizontal position of the button: `left`, `center`, or `right`. |
| `show_tooltip` | Boolean | `false` | Display a "Scroll to top" tooltip on hover. |
| `show_progress_ring` | Boolean | `false` | Show a circular progress indicator around the button. |
| `border_radius` | Number | `15` | Corner rounding as a percentage. `0` gives a square button, `50` gives a full circle. |
| `smooth_scroll` | Boolean | `true` | Use smooth scrolling animation when returning to the top. Ignored when the reader's system requests reduced motion; the scroll is then instant. |
| `progress_ring_color` | Color | `""` | Custom color for the progress ring. Empty uses the button's text color. |

## Injection rule

This plugin activates on every page (`always`).

## Edge cases

- Negative threshold values fall back to the default of 300px. `threshold: 0` is valid and shows the button as soon as any scrolling occurs.
- The button automatically hides when the page footer (`.sarde-footer`) comes into view, so it never overlaps footer content. On pages or themes without a footer element, this check is skipped and the button stays visible past the threshold.
- The `border_radius` value is applied as a CSS percentage, not pixels. A value of 15 produces a rounded square (the default).
- The button is automatically hidden when the browser zoom exceeds 300% to avoid layout issues.
- On mobile (below 768px), the button shrinks to 40x40 pixels.
- The button supports keyboard activation (::kbd[Enter]) and displays a visible focus ring when navigated via ::kbd[Tab].
- The button is hidden in print output.
- Touch devices use tap events with a visual active state.
