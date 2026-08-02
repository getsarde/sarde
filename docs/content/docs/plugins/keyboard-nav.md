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

By default, the plugin injects chevron arrows fixed to the left and right edges of the viewport. Each arrow is a full-height column that tints on hover, so the whole edge of the page is a click target. Hovering an arrow (or reaching it with the keyboard) reveals a styled tooltip beside the chevron showing the target page's title and the arrow key that triggers it. The tooltip appears after a short delay, so brushing the edge with the pointer does not flash it, and long titles truncate with an ellipsis. Clicking navigates to that page.

The arrows grow with the viewport, since a wider screen has more page margin to spend on them:

| Viewport | Column width | Chevron |
|--------|--------|--------|
| 1280px | 56px | 28px |
| 1440px | 72px | 32px |
| 1600px and above | 90px | 40px |

To keep the arrows off your content, the plugin reserves matching page margin on both sides of the content column whenever a page has previous or next links. The prose column and the table of contents shift inward so the arrows sit in empty space rather than over text. Between 1280px and 1600px this narrows the prose column by up to about 90px; at 1600px and above the page margin is already wide enough and the prose column is unaffected.

Set `show_side_nav: false` to disable the side arrows while keeping keyboard navigation active. Because the reserved margin is part of the initial page render, disabling the arrows releases that margin once the plugin script runs, which shifts the content column one time on load. The default setting has no such shift.

### Compact buttons on smaller screens

Below 1280px there is no page margin to spend on full-height strips, so the arrows restyle into small circular buttons (44px, accent background with a white chevron, matching the scroll-to-top button) floating at the left and right viewport edges, vertically centered. This keeps the plugin's core promise, navigating without scrolling to the bottom pagination bar, on laptops, tablets, and phones. The reserved page margin does not apply at these widths; the buttons float above the content edge instead.

While the desktop sidebar is visible (1024px to 1279px), the previous button clears it, sliding along when the sidebar is collapsed. Below 1024px, where the sidebar becomes a drawer, both buttons sit at the viewport edges, and the open drawer's backdrop covers them.

The compact buttons are the same links as the strips, so the `aria-label`s, focus ring, and the flash highlight on key navigation all carry over unchanged. The hover tooltip does not: it is desktop-only (1280px and up), because on touch screens a tap leaves the button in a sticky hover state that would pin the tooltip open over the content. The target page title stays available to assistive technology through the `aria-label` at every width.

Set `show_compact_nav: false` to hide the side navigation entirely below 1280px and fall back to the bottom pagination bar there, the behavior of earlier releases.

### Auto-hide

Because the compact buttons float over the content edge, they fade out after 2 seconds of inactivity and come back once you scroll about 100px in either direction. Hovering or focusing a button keeps it visible, tabbing to a hidden button brings the pair back, and pages too short to scroll never hide the buttons at all (scrolling could not reveal them again).

The behavior is controlled by three options. `auto_hide` picks where it applies: `compact` (default) auto-hides only the floating buttons below 1280px, `always` also fades the desktop strips, and `off` keeps the navigation permanently visible. `hide_delay` and `scroll_threshold` tune the timings:

```yaml
plugins:
  config:
    keyboard-nav:
      auto_hide: compact
      hide_delay: 2000
      scroll_threshold: 100
```

### Arrow size

`side_nav_size` scales the whole ladder above:

| Value | Scale | Width at 1600px |
|--------|--------|--------|
| `small` | 0.8 | 72px |
| `medium` (default) | 1.0 | 90px |
| `large` | 1.3 | 117px |

Changing the size never changes the reserved margin, so switching between the three values does not reflow the page.

For a value outside the three presets, override the custom properties in your own CSS:

```css
:root {
  --sd-side-nav-width: 6rem;   /* column width before scaling */
  --sd-side-nav-icon: 2.75rem; /* chevron size before scaling */
  --sd-side-nav-scale: 1;      /* multiplier applied to both */
}
```

`--sd-side-nav-width` also drives the reserved page margin, so raising it keeps the arrows clear of your content.

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
      show_compact_nav: true
      show_tooltip: true
      side_nav_size: medium
      auto_hide: compact
      hide_delay: 2000
      scroll_threshold: 100
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `show_hint` | Boolean | `true` | Display a brief keyboard shortcut hint on page load. |
| `show_side_nav` | Boolean | `true` | Display chevron arrows on the left and right edges of the viewport. |
| `show_compact_nav` | Boolean | `true` | Below 1280px, show compact floating prev/next buttons instead of hiding the side navigation. Ignored when `show_side_nav` is `false`. |
| `auto_hide` | String | `compact` | Fade the side navigation out after inactivity and bring it back on scroll: `compact` (floating buttons only), `always` (strips too), or `off`. |
| `hide_delay` | Number | `2000` | Milliseconds of inactivity before the side navigation fades out. |
| `scroll_threshold` | Number | `100` | Scroll distance in pixels that brings the hidden side navigation back. |
| `show_tooltip` | Boolean | `true` | Show the target page title and its arrow key in a styled tooltip when hovering or focusing a side arrow. When `false`, the arrows fall back to the native browser `title` tooltip. |
| `side_nav_size` | String | `medium` | Size of the side arrows: `small`, `medium`, or `large`. An unrecognized value falls back to `medium`. |

## Injection rule

This plugin activates on pages with previous or next navigation links (`has_prev_next`).

## Accessibility

- Side arrows carry `aria-label` attributes combining the localized direction text from the pagination bar with the target page title (for example "Previous: Installation"). The visual tooltip is `aria-hidden`, since it duplicates the label, so screen readers hear the announcement exactly once.
- The tooltip appears on keyboard focus (`:focus-visible`) as well as hover, and the arrows show a focus ring, so tabbing through the page makes them and their targets visible.
- The tooltip's fade transition is disabled when the user prefers reduced motion.
- The hint toast uses `role="status"` for screen reader announcements. Below 1024px it is hidden with `display: none`, which also removes it from the accessibility tree, so nothing is announced there.
- Keyboard navigation respects `prefers-reduced-motion`: transition animations are disabled when the user prefers reduced motion.
- Side arrows and the hint toast are hidden in print output.
