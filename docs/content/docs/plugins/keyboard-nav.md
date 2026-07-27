---
title: Keyboard Nav
description: "Navigate between previous and next pages using the arrow keys, with optional side navigation arrows"
sidebar:
  order: 15
---

Navigate between previous and next pages using the ::kbd[Left] and ::kbd[Right] arrow keys. Optionally displays chevron arrows on the left and right edges of the viewport for click navigation. Disabled by default.

## Enable the plugin

`sarde.yaml`
```yaml
plugins:
  enabled:
    - search
    - seo
    - sitemap
    # ... other default plugins
    - keyboard-nav
```

## How it works

Once enabled, pressing ::kbd[Left] navigates to the previous page and ::kbd[Right] navigates to the next page. Navigation follows the same order as the bottom pagination bar (the prev/next links below the page content).

Arrow keys are ignored when focus is inside a text input, textarea, select element, or a CodeMirror editor. Modifier keys (::kbd[Ctrl], ::kbd[Alt], ::kbd[Shift], ::kbd[Meta]) also suppress navigation to avoid conflicts with browser shortcuts.

## Side navigation arrows

By default, the plugin injects chevron arrows fixed to the left and right edges of the viewport. Hovering an arrow shows the target page title as a tooltip. Clicking navigates to that page.

The side arrows are hidden on viewports narrower than 1280px, where the bottom pagination bar provides the same functionality without competing for screen space.

Set `show_side_nav: false` to disable the side arrows while keeping keyboard navigation active.

## Hint toast

On page load, a brief toast appears at the bottom of the screen showing ::kbd[Left] and ::kbd[Right] keys with the label "Navigate between pages". This helps new visitors discover the keyboard shortcut.

Set `show_hint: false` to suppress the toast.

The hint is hidden on viewports narrower than 1024px, where it would advertise keys a touch device does not have. The arrow-key handlers stay bound at every width, so a tablet with an attached keyboard still navigates.

## Configuration

`sarde.yaml`
```yaml
plugins:
  config:
    keyboard-nav:
      show_hint: true
      show_side_nav: true
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `show_hint` | Boolean | `true` | Display a brief keyboard shortcut hint on page load. |
| `show_side_nav` | Boolean | `true` | Display chevron arrows on the left and right edges of the viewport. |

## Injection rule

This plugin activates on pages with previous or next navigation links (`has_prev_next`).

## Accessibility

- Side arrows include `aria-label` attributes ("Previous page" / "Next page") and native `title` tooltips with the target page title.
- The hint toast uses `role="status"` for screen reader announcements. Below 1024px it is hidden with `display: none`, which also removes it from the accessibility tree, so nothing is announced there.
- Keyboard navigation respects `prefers-reduced-motion`: transition animations are disabled when the user prefers reduced motion.
- Side arrows and the hint toast are hidden in print output.
