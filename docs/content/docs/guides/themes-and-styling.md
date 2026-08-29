---
title: Themes and Styling
description: "Choose a theme preset, override design tokens, and control dark mode and CSS layer order"
sidebar:
  order: 13
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
| `ocean` | `#0ea5e9` (sky blue) | Inter / JetBrains Mono | Bright, fresh |
| `forest` | `#16a34a` (green) | Inter / JetBrains Mono | Nature, organic |
| `rose` | `#e11d48` (rose) | Inter / JetBrains Mono | Bold, warm |
| `clean` | `#0f766e` (teal) | Plus Jakarta Sans / Fira Code | Large border radius |
| `minimal` | `#18181b` (near-black) | System fonts / JetBrains Mono | Tight, compact |
| `docs` | `#2563eb` (blue) | System fonts / JetBrains Mono | Technical documentation |
| `academic` | `#1e40af` (dark blue) | Merriweather (serif) / JetBrains Mono | Scholarly, formal |

Without a preset, Sarde uses the default theme tokens compiled into the binary. To define your own, see [Creating a Preset](/customization/creating-a-preset/).

### Preset typography

The color-only presets (`ocean`, `forest`, `rose`) keep the base theme's bundled Inter and JetBrains Mono fonts. The full-look presets (`clean`, `minimal`, `docs`, `academic`) set their own `font-sans` stack as part of their visual identity. In particular, `docs` uses a native system-font stack, so pages render with zero font download.

Sarde only emits the Inter font preload when the resolved font tokens actually reference Inter. If you use a full-look preset but want Inter back, one shortcut field restores it, since user config wins over preset tokens:

```yaml
theme:
  preset: "docs"
  font_family: "'Inter', system-ui, -apple-system, sans-serif"
```

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
| `sarde.variants` | Layout variant styles, such as the labs layout |
| `sarde.utils` | Utility classes, print styles |
| `sarde.plugins` | Styles contributed by plugins |
| `sarde.user` | Reserved for the project's own CSS. Nothing ships in it |

Custom CSS added via `head.custom_css` loads after all layers and can override any style.

To place your own rules inside the cascade instead of after it, wrap them in the reserved layer:

```css
@layer sarde.user {
  .sarde-sidebar { border-inline-end: 1px solid var(--sd-border); }
}
```

Rules in `sarde.user` beat every Sarde layer, including plugin styles, without `!important`.

## Theme eject

To customize a template, component, or stylesheet beyond token overrides, eject that file and edit the copy:

```sh
sarde theme eject layouts/components/Header.html
sarde theme eject css/components.css
```

Result: The files land in `themes/default/` (with a `theme.yaml` if none exists) and override their embedded counterparts. Everything not ejected keeps coming from the embedded theme, including fixes in later Sarde releases.

Run `sarde theme eject --list` to see every ejectable path, or `sarde theme eject` with no paths to copy the whole theme. See [`theme eject`](/reference/cli-commands/#theme-eject) for all options.

:::note
An ejected file is a snapshot. Sarde updates do not touch it, so eject only what you change, and merge upstream changes by hand after upgrading.
:::

### The theme `css/` directory

A theme's stylesheets live in `themes/<name>/css/`. Each of the 24 stylesheets Sarde looks for is read from the theme when present and from the embedded theme otherwise, so a theme can ship a single edited stylesheet. Files outside that set are ignored.

:::caution
`assets/` does not work this way. A theme `assets/` directory replaces the embedded fonts, vendor scripts, and JS entirely; eject it whole with `sarde theme eject assets`.
:::

To change a few rules, prefer `head.custom_css` or the `sarde.user` layer above.

## Content width toggle

Docs pages carry a header button that switches the article between a centered column and the full available width. It is pure presentation, needs no configuration, and is on by default.

The button appears only on the docs layout and only at viewports of 1280px or wider, since narrower screens have no spare width to give back. The choice is stored in `localStorage` under `sd-docs-centered` and applied by toggling a class of the same name on `<html>`, so it survives navigation and reloads.

To remove it, override the `Header` component and drop the `CenterToggle` call. To restyle it, target `.sarde-center-toggle`.

See [Theme Tokens](/reference/theme-tokens) for the complete token reference, and [Configuration](/reference/configuration#theme) for all theme settings.
