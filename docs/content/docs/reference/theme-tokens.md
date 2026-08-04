---
title: Theme Tokens
description: "Reference for every CSS custom property token and how the four-layer token cascade resolves them"
sidebar:
  order: 4
---

Sarde uses CSS custom properties (prefixed `--sd-*`) called tokens to control every visual aspect of the generated site. Override any token in [`theme.overrides`](/reference/configuration#theme) or [`theme.dark_overrides`](/reference/configuration#theme) in `sarde.yaml`.

```yaml
theme:
  overrides:
    accent: "#2563eb"
    radius: "0.25rem"
    font-sans: "'Inter', sans-serif"
  dark_overrides:
    bg: "#111113"
```

## Token cascade

Tokens resolve through four layers. Each layer overrides the one before it. Empty-string values never override a lower layer.

| Priority | Layer | Source |
|----------|-------|--------|
| 1 (lowest) | Embedded defaults | Hardcoded Go maps compiled into the binary |
| 2 | Theme base tokens | `theme.yaml` top-level `tokens:` / `dark_tokens:` |
| 3 | Preset tokens | Selected preset's `tokens:` / `dark_tokens:` |
| 4 (highest) | User overrides | `sarde.yaml` `theme.overrides` / `theme.dark_overrides` |

:::note
Dark tokens resolve independently through the same four layers using `dark_tokens` / `dark_overrides` at each level. The light and dark cascades never mix.
:::

## Primitives

The base color palette. Grays follow the Zinc scale in OKLCH.

| Token | Default |
|-------|---------|
| `white` | `oklch(1 0 0)` |
| `black` | `oklch(0.145 0.005 286)` |
| `gray-1` | `oklch(0.985 0.002 286)` |
| `gray-2` | `oklch(0.967 0.003 286)` |
| `gray-3` | `oklch(0.920 0.004 286)` |
| `gray-4` | `oklch(0.705 0.015 286)` |
| `gray-5` | `oklch(0.553 0.014 286)` |
| `gray-6` | `oklch(0.442 0.012 286)` |
| `gray-7` | `oklch(0.370 0.011 286)` |
| `gray-8` | `oklch(0.274 0.006 286)` |
| `gray-9` | `oklch(0.210 0.006 286)` |

Dark mode remaps semantic tokens (text, bg, border) instead of redefining the primitive gray scale.

## Status colors

### Hues

OKLCH hue angles used as the base for status color calculations.

| Token | Default |
|-------|---------|
| `hue-accent` | `264` |
| `hue-green` | `155` |
| `hue-amber` | `75` |
| `hue-red` | `25` |
| `hue-cyan` | `196` |

### Color scales

Each status color has three variants: the base color, a `-low` variant for subtle background tints, and a `-high` variant for stronger foreground emphasis.

| Token | Default |
|-------|---------|
| `green` | `oklch(0.52 0.15 var(--sd-hue-green))` |
| `green-low` | `oklch(0.95 0.04 var(--sd-hue-green))` |
| `green-high` | `oklch(0.65 0.18 var(--sd-hue-green))` |
| `amber` | `oklch(0.75 0.16 var(--sd-hue-amber))` |
| `amber-low` | `oklch(0.96 0.04 var(--sd-hue-amber))` |
| `amber-high` | `oklch(0.79 0.16 var(--sd-hue-amber))` |
| `red` | `oklch(0.55 0.20 var(--sd-hue-red))` |
| `red-low` | `oklch(0.96 0.05 var(--sd-hue-red))` |
| `red-high` | `oklch(0.65 0.22 var(--sd-hue-red))` |
| `blue` | `oklch(0.62 0.20 231)` |
| `blue-low` | `oklch(0.97 0.02 231)` |
| `blue-high` | `oklch(0.70 0.18 231)` |
| `purple` | `oklch(0.59 0.23 293)` |
| `purple-low` | `oklch(0.97 0.02 293)` |
| `purple-high` | `oklch(0.70 0.22 293)` |

## Extended hues

Additional hue angles and color scales for cyan and indigo.

### Hues

| Token | Default |
|-------|---------|
| `hue-blue` | `230` |
| `hue-indigo` | `264` |
| `hue-purple` | `305` |

### Color scales

| Token | Default |
|-------|---------|
| `cyan` | `oklch(0.52 0.14 var(--sd-hue-cyan))` |
| `cyan-low` | `oklch(0.95 0.04 var(--sd-hue-cyan))` |
| `cyan-high` | `oklch(0.65 0.18 var(--sd-hue-cyan))` |
| `indigo` | `oklch(0.60 0.18 var(--sd-hue-indigo))` |
| `indigo-low` | `oklch(0.95 0.04 var(--sd-hue-indigo))` |
| `indigo-high` | `oklch(0.72 0.20 var(--sd-hue-indigo))` |

## Semantic tokens

### Text

| Token | Default | Description |
|-------|---------|-------------|
| `text` | `var(--sd-gray-9)` | Primary body text |
| `text-high` | `var(--sd-black)` | Higher-contrast headings |
| `text-muted` | `var(--sd-gray-6)` | Secondary text, captions |
| `text-subtle` | `var(--sd-gray-5)` | Tertiary text, placeholders |
| `text-accent` | `var(--sd-accent, oklch(0.60 0.22 var(--sd-hue-accent)))` | Accent-colored text |
| `text-invert` | `var(--sd-white)` | Text on dark backgrounds |
| `text-success` | `var(--sd-green)` | Success state text |
| `text-warning` | `var(--sd-amber)` | Warning state text |
| `text-danger` | `var(--sd-red)` | Error state text |

### Backgrounds

| Token | Default | Description |
|-------|---------|-------------|
| `bg` | `var(--sd-white)` | Page background |
| `bg-surface` | `var(--sd-gray-1)` | Card and panel background |
| `bg-nav` | `var(--sd-bg-sidebar)` | Top navigation background |
| `bg-sidebar` | `var(--sd-gray-1)` | Sidebar background |
| `bg-footer` | `var(--sd-bg)` | Footer background |
| `bg-toc` | `transparent` | Table of contents background |
| `bg-inline-code` | `var(--sd-gray-2)` | Inline code background |
| `bg-accent` | `var(--sd-accent, oklch(0.60 0.22 var(--sd-hue-accent)))` | Accent-colored background |
| `bg-accent-subtle` | `var(--sd-accent-low, oklch(0.60 0.22 var(--sd-hue-accent) / 0.1))` | Subtle accent background |
| `hover` | `var(--sd-gray-2)` | Hover state background |

### Borders and interactive

| Token | Default | Description |
|-------|---------|-------------|
| `border` | `var(--sd-gray-3)` | Standard border color |
| `border-muted` | `var(--sd-gray-3)` | Subtle border color |
| `border-accent` | `var(--sd-accent, oklch(0.60 0.22 var(--sd-hue-accent)))` | Accent-colored border |
| `hairline` | `color-mix(in oklch, var(--sd-gray-5) 20%, transparent)` | Ultra-thin divider line |
| `focus-ring` | `var(--sd-accent, oklch(0.60 0.22 var(--sd-hue-accent)))` | Focus ring color for interactive elements |

## Typography

### Fonts

| Token | Default |
|-------|---------|
| `font-sans` | `'Inter', system-ui, -apple-system, sans-serif` |
| `font-mono` | `'JetBrains Mono', ui-monospace, monospace` |

Inter and JetBrains Mono are bundled with the theme and served locally. The Inter preload tag is only emitted when the resolved `font-sans` or `font-mono` value actually references Inter, so presets that use system fonts (such as `docs`) ship pages with no font download.

### Size scale

| Token | Default |
|-------|---------|
| `text-xs` | `0.75rem` |
| `text-sm` | `0.875rem` |
| `text-base` | `1rem` |
| `text-lg` | `1.125rem` |
| `text-xl` | `1.25rem` |
| `text-2xl` | `1.5rem` |
| `text-3xl` | `1.875rem` |
| `text-4xl` | `2.25rem` |
| `text-5xl` | `2.625rem` |

### Heading aliases

Heading tokens map to the size scale and shift up one step on wider viewports.

| Token | Default (< 50em) | Default (>= 50em) |
|-------|-------------------|---------------------|
| `text-h1` | `var(--sd-text-4xl)` | `var(--sd-text-5xl)` |
| `text-h2` | `var(--sd-text-3xl)` | `var(--sd-text-4xl)` |
| `text-h3` | `var(--sd-text-2xl)` | `var(--sd-text-3xl)` |
| `text-h4` | `var(--sd-text-xl)` | `var(--sd-text-2xl)` |
| `text-h5` | `var(--sd-text-lg)` | `var(--sd-text-lg)` |
| `text-h6` | `var(--sd-text-base)` | `var(--sd-text-base)` |

### Line height, weight, and tracking

| Token | Default |
|-------|---------|
| `line-height` | `1.75` |
| `line-height-headings` | `1.2` |
| `line-height-relaxed` | `1.8` |
| `weight-normal` | `400` |
| `weight-medium` | `500` |
| `weight-semibold` | `600` |
| `weight-bold` | `700` |
| `tracking-tight` | `-0.025em` |
| `tracking-wide` | `0.05em` |

## Spacing

| Token | Default |
|-------|---------|
| `space-1` | `0.25rem` |
| `space-2` | `0.5rem` |
| `space-3` | `0.75rem` |
| `space-4` | `1rem` |
| `space-5` | `1.25rem` |
| `space-6` | `1.5rem` |
| `space-8` | `2rem` |
| `space-10` | `2.5rem` |
| `space-12` | `3rem` |

## Layout

| Token | Default | Description |
|-------|---------|-------------|
| `nav-height` | `4rem` | Top navigation bar height |
| `logo-height` | `1.75rem` | Site logo height in the header |
| `sidebar-width` | `280px` | Sidebar width on desktop |
| `sidebar-collapsed-width` | `40px` | Sidebar width when collapsed |
| `toc-width` | `240px` | Table of contents panel width |
| `mobile-toc-height` | `3rem` | Mobile table of contents bar height |
| `content-width` | `800px` | Maximum content area width |
| `docs-centered-width` | `87.5rem` | Maximum width of the docs-layout container |

## Radius

| Token | Default |
|-------|---------|
| `radius` | `0.5rem` |
| `radius-sm` | `0.25rem` |
| `radius-md` | `var(--sd-radius, 0.375rem)` |
| `radius-lg` | `0.5rem` |
| `radius-xl` | `0.75rem` |
| `radius-full` | `9999px` |

Overriding `radius` does not automatically change the size variants. Override each variant individually to maintain a consistent scale.

## Z-index

Stacking layers for the header, sidebar, table of contents, overlays, and modals.

| Token | Default |
|-------|---------|
| `z-sticky` | `10` |
| `z-toc` | `25` |
| `z-sidebar` | `30` |
| `z-header` | `35` |
| `z-overlay` | `40` |
| `z-modal` | `50` |
| `z-toast` | `400` |

## Shadows

| Token | Default |
|-------|---------|
| `shadow-xs` | `0 1px 1px oklch(0 0 0 / 0.04)` |
| `shadow-sm` | `0 1px 2px oklch(0 0 0 / 0.05)` |
| `shadow-md` | `0 4px 6px -1px oklch(0 0 0 / 0.08), 0 2px 4px -1px oklch(0 0 0 / 0.04)` |
| `shadow-lg` | `0 10px 15px -3px oklch(0 0 0 / 0.1), 0 4px 6px -4px oklch(0 0 0 / 0.1)` |
| `shadow-accent` | `0 4px 14px -2px oklch(0.60 0.22 264 / 0.25)` |

## Transitions and motion

### Transitions and easing

| Token | Default |
|-------|---------|
| `transition-fast` | `150ms ease` |
| `transition-normal` | `200ms ease` |
| `transition-slow` | `300ms ease` |
| `duration-100` | `100ms` |
| `duration-150` | `150ms` |
| `duration-200` | `200ms` |
| `duration-700` | `700ms` |
| `ease-default` | `cubic-bezier(0.4, 0, 0.2, 1)` |
| `ease-out` | `cubic-bezier(0, 0, 0.2, 1)` |
| `ease-bounce` | `cubic-bezier(0.34, 1.56, 0.64, 1)` |

### Motion transforms

| Token | Default |
|-------|---------|
| `motion-lift` | `translateY(-2px)` |
| `motion-lift-lg` | `translateY(-4px)` |
| `motion-scale-sm` | `scale(1.02)` |

## Glass

Frosted-glass effect tokens used for overlays and backdrop panels.

| Token | Default |
|-------|---------|
| `glass-bg` | `oklch(1 0 0 / 0.8)` |
| `glass-blur` | `12px` |
| `glass-border` | `oklch(1 0 0 / 0.2)` |

## Code block tokens

| Token | Default | Description |
|-------|---------|-------------|
| `code-bg` | `var(--sd-bg-surface)` | Code block background |
| `code-text` | `var(--sd-text)` | Code block text color |
| `code-border` | `var(--sd-border)` | Code block border color |
| `code-font` | `var(--sd-font-mono)` | Code block font family |
| `code-font-size` | `var(--sd-text-sm)` | Code block font size |
| `code-line-height` | `1.7` | Code block line height |
| `code-padding` | `var(--sd-space-4)` | Code block inner padding |
| `code-radius` | `var(--sd-radius-lg)` | Code block corner radius |
| `code-btn-size` | `2rem` | Toolbar button size |
| `code-btn-bg` | `transparent` | Toolbar button background |
| `code-btn-text` | `var(--sd-text-muted)` | Toolbar button text color |
| `code-btn-bg-hover` | `var(--sd-hover)` | Toolbar button hover background |
| `code-btn-text-hover` | `var(--sd-text)` | Toolbar button hover text color |

## Aside tokens

Foreground and background color pairs for each aside type.

| Token | Default |
|-------|---------|
| `aside-note` | `var(--sd-blue)` |
| `aside-note-bg` | `var(--sd-blue-low)` |
| `aside-tip` | `var(--sd-green)` |
| `aside-tip-bg` | `var(--sd-green-low)` |
| `aside-info` | `var(--sd-cyan)` |
| `aside-info-bg` | `var(--sd-cyan-low)` |
| `aside-caution` | `var(--sd-amber)` |
| `aside-caution-bg` | `var(--sd-amber-low)` |
| `aside-danger` | `var(--sd-red)` |
| `aside-danger-bg` | `var(--sd-red-low)` |
| `aside-important` | `var(--sd-purple)` |
| `aside-important-bg` | `var(--sd-purple-low)` |
| `aside-success` | `var(--sd-green)` |
| `aside-success-bg` | `var(--sd-green-low)` |

Each type also has an `aside-{type}-text` token for the title and icon color. These are
deliberately darker than the base accent so titles keep WCAG AA contrast on the tinted
backgrounds; dark mode remaps them to the `-high` primitives (e.g. `aside-note-text` becomes
`var(--sd-blue-high)`).

| Token | Light default |
|-------|---------------|
| `aside-note-text` | `oklch(0.48 0.19 231)` |
| `aside-tip-text` | `oklch(0.48 0.14 155)` |
| `aside-info-text` | `oklch(0.48 0.13 196)` |
| `aside-caution-text` | `oklch(0.52 0.13 75)` |
| `aside-danger-text` | `oklch(0.50 0.19 25)` |
| `aside-important-text` | `oklch(0.48 0.20 293)` |
| `aside-success-text` | `oklch(0.48 0.14 155)` |

## Accent derivation

Setting [`theme.accent_color`](/reference/configuration#theme) or `theme.overrides.accent` triggers automatic derivation of three variant tokens. If any variant is already set explicitly, that variant is kept as-is.

| Derived token | Purpose |
|---------------|---------|
| `accent-hover` | Slightly darker accent for hover states |
| `accent-high` | Lighter accent for dark-mode emphasis |
| `accent-low` | Accent at 10% opacity for subtle backgrounds |

The derivation formulas depend on the color format of the `accent` value.

:::tabs

== Hex

For hex colors (`#RGB` or `#RRGGBB`), derivation converts to HSL, adjusts lightness, and converts back.

| Token | Formula |
|-------|---------|
| `accent-hover` | Lightness - 0.10 (10% darker) |
| `accent-high` | Lightness + 0.20 (20% lighter) |
| `accent-low` | `rgba(r, g, b, 0.1)` |

== OKLCH

For OKLCH colors (`oklch(L C H)`), derivation adjusts the lightness component directly.

| Token | Formula |
|-------|---------|
| `accent-hover` | L - 0.08 |
| `accent-high` | L + 0.12 |
| `accent-low` | `oklch(L C H / 0.1)` |

:::

Accent values in other formats (named colors, `hsl()`, CSS variables) skip derivation gracefully. Set `accent-hover`, `accent-high`, and `accent-low` manually in that case.

:::note
Accent derivation runs on light-mode tokens only. Dark-mode accent variants come from CSS `var()` fallback chains in the stylesheet, not from Go-level derivation. To customize dark-mode accent variants, set them explicitly in `theme.dark_overrides`.
:::

## Presets

Seven built-in presets provide ready-made visual identities. Set one with [`theme.preset`](/reference/configuration#theme) in `sarde.yaml`.

```yaml
theme:
  preset: ocean
```

Palette presets override the accent color and gray scale (gray-1 through gray-7) to tint the entire UI. Full-look presets change colors, typography, and border radius together.

| Preset | Type | Accent hue | Custom grays | Font | Radius |
|--------|------|------------|--------------|------|--------|
| `ocean` | Palette | 211 (cyan-blue) | Hue 196 | Default | Default |
| `forest` | Palette | 152 (green) | Hue 75 | Default | Default |
| `rose` | Palette | 13 (red-pink) | Hue 355 | Default | Default |
| `clean` | Full-look | 182 (teal) | Default | Plus Jakarta Sans | `0.75rem` |
| `minimal` | Full-look | 286 (near-monochrome) | Default | system-ui | `0.375rem` |
| `docs` | Full-look | 264 (indigo) | Default | System sans-serif | `0.375rem` |
| `academic` | Full-look | 264 (deep indigo) | Default | Merriweather (serif) | `0.25rem` |

Inspect a preset's tokens with [`sarde theme info`](/reference/cli-commands#theme).
