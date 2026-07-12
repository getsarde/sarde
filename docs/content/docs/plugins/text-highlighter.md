---
title: Text Highlighter
description: "Let readers highlight text passages, persisted across visits via localStorage"
sidebar:
  order: 22
  group: Client-Side
---

Lets readers highlight text passages on a page with colored markers, persisted in `localStorage` so highlights survive page reloads and revisits. Disabled by default.

## Enable the plugin

`sarde.yaml`
```yaml
plugins:
  enabled:
    - search
    - seo
    - sitemap
    # ... other default plugins
    - text-highlighter
```

## How it works

Select any text within the content area. A floating toolbar appears above the selection with five color swatches (yellow, green, blue, pink, orange) and a clear-all button. Click a color to apply the highlight. The selection is wrapped in a `<mark>` element with the chosen color.

Click an existing highlight to open a remove popup. Click "Remove" to delete that single highlight.

Highlights are stored per page in `localStorage` with surrounding context characters, so they can be restored even if the page content shifts slightly between visits.

<!-- SCREENSHOT: text-highlighter-toolbar - the floating color swatch toolbar appearing above selected text -->

## Configuration

`sarde.yaml`
```yaml
plugins:
  config:
    text-highlighter:
      max_highlights_per_page: 50
      context_chars: 40
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `max_highlights_per_page` | Number | `50` | Maximum highlights stored per page. Range: 10 to 200. When exceeded, the oldest highlights are dropped. |
| `context_chars` | Number | `40` | Characters of surrounding text stored alongside each highlight for position matching on revisit. Range: 20 to 80. |

## Restoring highlights

On each page load, the plugin reads saved highlights from `localStorage` and attempts to relocate each one in the current page text. It uses the stored context (prefix and suffix characters) to find the best match, scoring candidates by character similarity. A match requires at least 40% context similarity. Highlights that can no longer be located (because the content changed too much) are silently dropped and removed from storage.

## Available colors

| Color | CSS Value |
|-------|-----------|
| Yellow | `hsla(48, 96%, 53%, 0.35)` |
| Green | `hsla(142, 71%, 45%, 0.3)` |
| Blue | `hsla(217, 91%, 60%, 0.3)` |
| Pink | `hsla(330, 81%, 60%, 0.3)` |
| Orange | `hsla(25, 95%, 53%, 0.35)` |

## Clearing highlights

The floating toolbar includes a trash icon button that removes all highlights on the current page and clears the `localStorage` entry. Individual highlights can be removed by clicking the highlighted text and selecting "Remove" from the popup.

## Skipped elements

The plugin does not allow highlighting inside `<script>`, `<style>`, `<code>`, `<pre>`, `<kbd>`, `<textarea>`, `<input>`, or `<svg>` elements. Text inside search-highlighter marks (from the Search Highlighter plugin) is also skipped to avoid conflicts.

## Injection rule

This plugin activates on content pages (`is_content_page`).

## Edge cases

- Overlapping highlights are resolved automatically: applying a new highlight over an existing one removes the old highlight first.
- Highlights that span multiple DOM nodes (across element boundaries) are wrapped per node segment.
- A background pruning task runs with a 10% probability on each page load, removing `localStorage` entries older than 90 days.
- The toolbar and remove popup are hidden in print output.
- Animations respect `prefers-reduced-motion` by disabling transitions.
- Pressing ::kbd[Escape] dismisses the toolbar or remove popup.
