---
title: Themes and Styling
description: "Choose a theme preset, override design tokens, and control dark mode and CSS layer order"
sidebar:
  order: 12
---

Sarde controls all visual styling through design tokens (CSS custom properties prefixed `--sd-*`). Choose a preset for a ready-made look, override individual tokens for customization, or eject the full theme for complete control.

## Choosing a preset

Set a preset in `sarde.yaml` to apply a cohesive visual identity:

```yaml
theme:
  preset: "docs"
```

| Preset | Accent | Font | Character |
|--------|--------|------|-----------|
| `default` | `#6366f1` (indigo) | Inter / JetBrains Mono | Modern, clean |
| `ocean` | `#0ea5e9` (sky blue) | System fonts | Bright, fresh |
| `forest` | `#16a34a` (green) | System fonts | Nature, organic |
| `rose` | `#e11d48` (rose) | System fonts | Bold, warm |
| `clean` | `#0f766e` (teal) | Plus Jakarta Sans | Large border radius |
| `minimal` | `#18181b` (near-black) | System UI | Tight, compact |
| `docs` | `#2563eb` (blue) | System fonts | GitHub-style docs |
| `academic` | `#1e40af` (dark blue) | Merriweather (serif) | Scholarly, formal |

Without a preset, Sarde uses the default theme tokens compiled into the binary.

## Overriding tokens

Override individual tokens in `sarde.yaml` without changing the preset:

```yaml
theme:
  preset: "docs"
  overrides:
    accent: "#e63946"
    font-sans: "'Fira Sans', sans-serif"
    radius-md: "0.5rem"
```

Dark mode has its own override map:

```yaml
theme:
  dark_overrides:
    bg: "#0a0a0a"
    text: "#e5e5e5"
```

Overrides always win over preset and theme values. See [Theme Tokens](/reference/theme-tokens) for the full list of available token names.

## Shortcut fields

Common overrides have dedicated config fields that map to tokens:

```yaml
theme:
  accent_color: "#e63946"
  primary_color: "#1a1a2e"
  font_family: "'Fira Sans', sans-serif"
  font_mono: "'Fira Code', monospace"
  code_light: "github-light"
  code_dark: "github-dark"
```

| Field | Maps to |
|-------|---------|
| `accent_color` | `accent` token + auto-derived variants |
| `primary_color` | `primary` token |
| `font_family` | `font-sans` token |
| `font_mono` | `font-mono` token |
| `code_light` | Kazari light theme |
| `code_dark` | Kazari dark theme |

## Accent color derivation

Setting `accent_color` or the `accent` token triggers automatic generation of three variant tokens:

| Derived token | Derivation |
|---------------|------------|
| `accent-hover` | 10% darker (hex) or 0.08 lower lightness (OKLCH) |
| `accent-high` | 20% lighter (hex) or 0.12 higher lightness (OKLCH) |
| `accent-low` | 10% opacity version for subtle backgrounds |

If any variant is already set explicitly in `overrides`, that variant is kept as-is. Both hex (`#e63946`) and OKLCH (`oklch(0.65 0.2 25)`) accent values are supported.

## Token cascade

Tokens resolve through a 4-layer cascade (last wins):

1. **Embedded defaults** (compiled into the binary)
2. **Theme tokens** (`theme.yaml` from the active theme)
3. **Preset tokens** (the selected preset's overrides)
4. **User overrides** (`theme.overrides` in `sarde.yaml`)

Light and dark mode tokens resolve independently through the same cascade.

## Dark mode

Sarde ships with full dark mode support. The theme toggle is a 3-position control: light, system, and dark.

- **Light**: forces light theme
- **System**: follows `prefers-color-scheme` from the OS
- **Dark**: forces dark theme

The active choice is persisted to `localStorage` (`sd-theme`) and applied via the `data-theme` attribute on the `<html>` element. CSS uses `:root[data-theme="dark"]` selectors for dark mode overrides.

Disable dark mode entirely:

```yaml
theme:
  dark: false
```

Result: The theme toggle and dark mode CSS are removed from the output.

## CSS layer order

Sarde organizes CSS into layers using `@layer` declarations. Layers cascade in this order (later layers override earlier ones):

| Layer | Purpose |
|-------|---------|
| `sarde.base` | Design tokens (CSS custom properties) |
| `sarde.reset` | CSS reset, base typography, dark mode non-token overrides |
| `sarde.core` | Layout grid systems |
| `sarde.content` | Prose styles, blog styles, homepage styles |
| `sarde.components` | UI components, extensions, search |
| `sarde.utils` | Utility classes, print styles |
| `kazari` | Code block highlighting styles (managed by Kazari) |

Custom CSS added via `head.custom_css` loads after all layers and can override any style.

## Theme eject

To customize templates, components, or CSS beyond token overrides, eject the embedded theme:

```sh
sarde theme eject
```

Result: Copies the full embedded theme (templates, CSS, JS, components, partials, `theme.yaml`) to `themes/default/`. All files in `themes/default/` override their embedded counterparts.

After ejecting, edit any file in `themes/default/`. The template overlay resolution ensures ejected files take precedence over the compiled-in defaults.

:::note
Ejecting creates a snapshot of the current embedded theme. Future Sarde updates do not automatically update ejected files. Merge changes manually after upgrading.
:::

See [Theme Tokens](/reference/theme-tokens) for the complete token reference, and [Configuration](/reference/configuration#theme) for all theme settings.
