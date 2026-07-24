---
title: Reading Position Memory
description: "Remember a reader's scroll position and offer to restore it on return"
sidebar:
  order: 17
---

Remembers the scroll position on each page and offers to restore it on return. A toast notification lets the reader jump back to where they left off. Disabled by default.

## Enable the plugin

`sarde.yaml`
```yaml
plugins:
  enabled:
    - search
    - seo
    - sitemap
    # ... other default plugins
    - reading-position-memory
```

## How it works

As the reader scrolls, the plugin saves the scroll position to `localStorage` (debounced at 500ms). On a return visit to the same page, a toast appears with the saved percentage and a "Jump" button.

The toast shows: "Continue where you left off? (42%)" with two actions: **Jump** scrolls smoothly to the saved position, and **Dismiss** (×) closes the toast.

<!-- SCREENSHOT: reading-position-memory-toast — the "Continue where you left off?" toast with Jump and Dismiss buttons -->

The toast auto-dismisses after 8 seconds by default. Saved positions expire after 30 days.

## Configuration

`sarde.yaml`
```yaml
plugins:
  config:
    reading-position-memory:
      toast_duration: 8
      toast_position: "bottom-center"
      scroll_threshold: 5
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `toast_duration` | Number | `8` | Seconds before the toast auto-dismisses. Range: 3 to 30. |
| `toast_position` | String | `"bottom-center"` | Toast placement: `bottom-center`, `bottom-left`, or `bottom-right`. |
| `scroll_threshold` | Number | `5` | Minimum scroll percentage before the position is saved. Range: 1 to 50. |

## Injection rule

This plugin activates on content pages (`is_content_page`). Index pages, listing pages, and taxonomy pages do not receive the plugin.

## Edge cases

- Scrolling back near the top of the page (below the threshold) clears the saved position for that page.
- Positions saved less than 10 seconds ago do not trigger the toast, preventing it from appearing immediately after a quick page refresh.
- Saved positions expire after 30 days. A background cleanup runs with a 10% probability on each page load to prune expired entries.
- Storage is keyed by URL path only. Query strings and hash fragments do not create separate entries.
- If `localStorage` is unavailable (e.g., private browsing), the plugin degrades silently with no errors.
- On mobile (below 768px), the toast spans the full width regardless of the configured position.
- The toast is hidden in print output.
- The toast transition respects `prefers-reduced-motion`.
