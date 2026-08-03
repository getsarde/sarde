---
title: Focus Mode
description: "Hide the sidebar and table of contents for distraction-free reading"
sidebar:
  order: 13
---

Focus mode hides the sidebar and table of contents for distraction-free reading. The content area re-centers at a comfortable reading width. Disabled by default.

## Enable the plugin

`sarde.yaml`
```yaml
plugins:
  enabled:
    - search
    - seo
    - sitemap
    # ... other default plugins
    - focus_mode
```

## How it works

Toggle focus mode with ::kbd[Shift+F] or the floating button in the bottom-right corner. When active:

- The sidebar slides off-screen to the left.
- The table of contents slides off-screen to the right.
- The content area re-centers with a maximum width of 55rem.
- A red dot appears on the toggle button to confirm the mode is active.

Focus mode persists across page loads via `localStorage`. Returning visitors land in the same mode they left.

The feature is desktop-only. Below 1024px the sidebar is a slide-in drawer and the table of contents is a collapsible bar at the top, so there is nothing for focus mode to collapse. The toggle button is hidden at those widths and focus mode has no effect on the layout.

<!-- SCREENSHOT: focus-mode-active — the page with sidebar and TOC hidden, content centered -->

## Configuration

`sarde.yaml`
```yaml
plugins:
  config:
    focus_mode:
      enable_hotkey: true
      show_button: true
      button_position: "bottom-right"
      animation_speed: 300
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enable_hotkey` | Boolean | `true` | Enable the ::kbd[Shift+F] keyboard shortcut. |
| `show_button` | Boolean | `true` | Display the floating toggle button. |
| `button_position` | String | `"bottom-right"` | Button placement: `bottom-right` or `bottom-left`. |
| `animation_speed` | Number | `300` | Transition duration in milliseconds. Range: 0 to 1000. |

## Injection rule

This plugin activates on pages with a sidebar (`has_sidebar`), which includes the `docs` and `wide` layouts. Pages using other layouts receive no plugin config or behavior.

## Edge cases

- The hotkey is disabled while typing in input fields, textareas, select elements, contenteditable regions, and CodeMirror editors.
- ::kbd[Shift+F] is required (not a bare ::kbd[F]) to prevent accidental activation, since the effect persists across reloads.
- Setting both `enable_hotkey` and `show_button` to `false` leaves no way to activate focus mode.
- The sidebar and TOC slide directions reverse automatically for RTL layouts.
- Below 1024px the toggle button is hidden and focus mode has no layout effect, including a state carried over from an earlier desktop visit. The preference is still stored, so the same visitor returning on a wider screen finds focus mode as they left it.
- ::kbd[Shift+F] stays bound below 1024px, but produces no visible change there.
- The toggle button is hidden in print output.
- The button's own fade-in animation uses a fixed 0.3 seconds regardless of `animation_speed`. Only the sidebar/TOC/content transitions respect the configured speed.
