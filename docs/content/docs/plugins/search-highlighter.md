---
title: Search Highlighter
description: "Highlight search terms on the page when readers arrive from site search"
sidebar:
  order: 21
---

Highlights search terms on the page when a reader arrives from the site search. Matched terms are wrapped in colored `<mark>` elements, and a sticky badge shows the match count with previous/next navigation. Disabled by default.

## Enable the plugin

`sarde.yaml`
```yaml
plugins:
  enabled:
    - search
    - seo
    - sitemap
    # ... other default plugins
    - search_highlighter
```

## How it works

When a reader clicks a search result, Sarde appends the query as a `?q=` parameter. The plugin reads that parameter, splits it into terms, and highlights every occurrence in the page's content area. Single-character terms are ignored.

A sticky badge appears above the content with the total match count (e.g., "3 matches"). The badge includes previous/next buttons to cycle through matches and a dismiss button to clear all highlights. Navigating to a match scrolls it smoothly into view and outlines it as the active match.

After highlighting, the `?q=` parameter is removed from the URL bar without a page reload.

<!-- SCREENSHOT: search-highlighter-badge - the sticky match count badge with prev/next/dismiss buttons -->

## Configuration

`sarde.yaml`
```yaml
plugins:
  config:
    search_highlighter:
      highlight_color: "hsla(38, 80%, 60%, 0.3)"
      active_outline_color: "hsl(38, 85%, 50%)"
      auto_scroll: true
      show_badge: true
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `highlight_color` | Color | `""` | Background color for highlighted terms. Empty uses the theme default (warm amber). |
| `active_outline_color` | Color | `""` | Outline color for the currently active match. Empty uses the theme default. |
| `auto_scroll` | Boolean | `true` | Scroll to the first match automatically when the page loads. |
| `show_badge` | Boolean | `true` | Display the sticky match count badge with prev/next navigation. |

## Match navigation

When the badge is visible, click the arrow buttons to cycle through matches. The list wraps around: pressing next on the last match returns to the first. The badge updates its counter to show the current position (e.g., "2 / 5").

Clicking the dismiss button (×) removes all highlights and cleans the URL.

## Skipped elements

The plugin does not highlight text inside `<script>`, `<style>`, `<code>`, `<pre>`, `<kbd>`, `<mark>`, `<textarea>`, `<input>`, or `<svg>` elements. Code blocks and inline code remain unaffected.

## Injection rule

This plugin activates on every page (`always`).

## Edge cases

- Terms shorter than two characters are filtered out before highlighting.
- If no matches are found on the page, the `?q=` parameter is removed silently and no badge appears.
- The badge is hidden in print output and highlights are stripped.
- Animations respect `prefers-reduced-motion` by disabling transitions.
- The badge is sticky below the navigation header, using the `--sd-nav-height` token for positioning.
