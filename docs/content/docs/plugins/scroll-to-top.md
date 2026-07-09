---
title: Scroll to Top
sidebar:
  order: 20
  group: Client-Side
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
    - scroll-to-top
```

## How it works

The button appears in the bottom-right corner after the reader scrolls past the configured threshold (default: 30% of the page). Clicking it scrolls smoothly back to the top.

When `showProgressRing` is enabled, a circular SVG ring around the button fills as the reader scrolls, providing a visual indicator of scroll position.

<!-- SCREENSHOT: scroll-to-top-with-ring - the scroll-to-top button with progress ring partially filled -->

## Configuration

`sarde.yaml`
```yaml
plugins:
  config:
    scroll-to-top:
      threshold: 30
      showProgressRing: true
      smoothScroll: true
      borderRadius: 15
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `threshold` | Number | `30` | Scroll percentage before the button appears. Effective range: 10 to 99. |
| `showTooltip` | Boolean | `false` | Display a "Scroll to top" tooltip on hover. |
| `showProgressRing` | Boolean | `false` | Show a circular progress indicator around the button. |
| `borderRadius` | Number | `15` | Corner rounding as a percentage. `0` gives a square button, `50` gives a full circle. |
| `smoothScroll` | Boolean | `true` | Use smooth scrolling animation when returning to the top. |
| `progressRingColor` | Color | `""` | Custom color for the progress ring. Empty uses the button's text color. |

## Injection rule

This plugin activates on every page (`always`).

## Edge cases

- Threshold values outside the range 10 to 99 fall back to the default of 30. Setting `threshold: 0` also falls back to 30.
- The `borderRadius` value is applied as a CSS percentage, not pixels. A value of 15 produces a rounded square (the default).
- The button is automatically hidden when the browser zoom exceeds 300% to avoid layout issues.
- On mobile (below 768px), the button shrinks to 40×40 pixels and repositions closer to the corner.
- The button supports keyboard activation (::kbd[Enter]) and displays a visible focus ring when navigated via ::kbd[Tab].
- The button is hidden in print output.
- Touch devices use tap events with a visual active state.
