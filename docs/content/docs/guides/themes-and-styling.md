---
title: Themes and Styling
description: "Choose a theme preset, override design tokens, and control dark mode and CSS layer order"
sidebar:
  order: 13
---

Sarde controls all visual styling through design tokens, CSS custom properties prefixed `--sd-*`. Choose a preset for a ready-made look, override individual tokens to adjust it, or eject theme files when tokens are not enough.

## Choosing a preset

Set a preset in `sarde.yaml`:

```yaml title="sarde.yaml"
theme:
  preset: "docs"
```

The default theme defines seven presets:

| Preset | Accent (light mode) | Sans font | Radius |
|--------|---------------------|-----------|--------|
| `ocean` | `oklch(0.67 0.19 211)` | Inter | `0.5rem` |
| `forest` | `oklch(0.55 0.17 152)` | Inter | `0.5rem` |
| `rose` | `oklch(0.56 0.24 13)` | Inter | `0.5rem` |
| `clean` | `oklch(0.49 0.12 182)` | Plus Jakarta Sans | `0.75rem` |
| `minimal` | `oklch(0.210 0.006 286)` | System fonts | `0.375rem` |
| `docs` | `oklch(0.53 0.24 270)` | System fonts | `0.375rem` |
| `academic` | `oklch(0.42 0.21 264)` | Merriweather | `0.25rem` |

Every preset uses JetBrains Mono for code except `clean` (Fira Code). Without a preset, the accent is indigo (hue 264), the fonts are Inter and JetBrains Mono, and the radius is `0.5rem`.

A preset name that the active theme does not define is ignored without a warning, so a typo leaves the default look in place. Run `sarde theme info default` to list the presets of a theme. To define your own, see [Creating a Preset](/customization/creating-a-preset/).

### Preset typography

The color presets (`ocean`, `forest`, `rose`) keep the bundled Inter and JetBrains Mono fonts. The full-look presets (`clean`, `minimal`, `docs`, `academic`) set their own `font-sans` stack. The `docs` preset uses a native system-font stack, so pages render with no font download.

Sarde bundles only Inter and JetBrains Mono. Plus Jakarta Sans, Fira Code, and Merriweather are not bundled. By default, `clean` and `academic` use them only on devices that have them installed, and fall back to the rest of the stack elsewhere. Set [`web_fonts`](#web-fonts) to load them for every visitor.

Sarde emits the Inter font preload only when the resolved font tokens reference Inter. To bring Inter back under a full-look preset, set `font_family`, which wins over preset tokens:

```yaml title="sarde.yaml"
theme:
  preset: "docs"
  font_family: "'Inter', system-ui, -apple-system, sans-serif"
```

## Overriding tokens

Override individual tokens in `sarde.yaml` without changing the preset. Token names omit the `--sd-` prefix:

```yaml title="sarde.yaml"
theme:
  preset: "docs"
  overrides:
    accent: "#e63946"
    font-sans: "'Fira Sans', sans-serif"
    radius-md: "0.5rem"
```

An unknown token name fails the build and suggests the closest known name, for example `unknown token "accnt" (did you mean "accent"?)`.

`dark_overrides` sets tokens for dark mode only:

```yaml title="sarde.yaml"
theme:
  dark_overrides:
    bg: "#0a0a0a"
    text: "#e5e5e5"
```

When `dark_overrides` is empty, dark mode uses `overrides` as well. Once `dark_overrides` has any entry, dark mode uses only that map, so repeat any token from `overrides` that dark mode needs.

Overrides win over preset and theme values. See [Theme Tokens](/reference/theme-tokens/) for the full list of token names.

## Shortcut fields

Common overrides have dedicated fields under `theme`:

```yaml title="sarde.yaml"
theme:
  accent_color: "#e63946"
  font_family: "'Fira Sans', sans-serif"
  font_mono: "'Fira Code', monospace"
  font_heading: "'Fraunces', Georgia, serif"
  font_scale: 1.1
```

| Field | Maps to |
|-------|---------|
| `accent_color` | `accent` token, plus derived variants |
| `primary_color` | `accent` token, used only when `accent_color` is not set |
| `font_family` | `font-sans` token |
| `font_mono` | `font-mono` token |
| `font_heading` | `font-heading` token (headings `h1` to `h6`) |
| `font_scale` | `text-scale` token, a multiplier from `0.5` to `2` for the text size scale |

An explicit `theme.overrides` entry for the same token wins over its shortcut field.

A font field that holds a single family name is quoted and given a generic fallback: `font_family: Plus Jakarta Sans` becomes `'Plus Jakarta Sans', system-ui, sans-serif`. The fallback follows the family's Google Fonts category (`serif`, `ui-monospace, monospace`, or `cursive` for handwriting fonts), and otherwise the field: `font_mono` falls back to `ui-monospace, monospace`, the others to `system-ui, sans-serif`. A value with a comma or a quote is used as written, and so is a generic keyword such as `system-ui`.

To change the code block themes, see [Code Blocks](/guides/code-blocks/#configuration).

## Web fonts

A font renders only when the visitor has it installed or the site loads it. Sarde bundles Inter and JetBrains Mono. To load the other fonts the theme names, set `web_fonts` to a font service:

```yaml title="sarde.yaml"
theme:
  preset: "clean"
  web_fonts: "bunny"
```

| Value | Service |
|-------|---------|
| `google` | [Google Fonts](https://fonts.google.com) |
| `bunny` | [Bunny Fonts](https://fonts.bunny.net), which serves the Google Fonts library |

Sarde reads the first family of the resolved `font-sans`, `font-heading`, and `font-mono` tokens, whether a preset, a shortcut field, or `overrides` set it. Every one of those families that is in the Google Fonts library and not bundled is requested in a single stylesheet, after a preconnect to the service, at the weights the theme uses (400 to 800, plus italics). A family outside the library, such as a system font, is left to the visitor's device.

With `web_fonts` set, each visitor's browser fetches the fonts from the service, which receives the visitor's IP address. Bunny Fonts states that it keeps no logs. A German court ruled in 2022 that loading Google Fonts from Google's servers without the visitor's consent breached the GDPR, so sites with EU visitors usually prefer `bunny`.

To self-host a font instead, put the font file and an `@font-face` rule in a stylesheet under `assets/` and list the stylesheet in `head.custom_css`.

## Accent color derivation

Setting `accent_color` or the `accent` token generates four variant tokens:

| Derived token | Derivation |
|---------------|------------|
| `accent-hover` | 0.10 lower lightness (hex) or 0.08 lower lightness (OKLCH) |
| `accent-high` | 0.20 higher lightness (hex) or 0.12 higher lightness (OKLCH) |
| `accent-low` | The accent at 10% opacity, for subtle backgrounds |
| `accent-text` | The accent with lightness capped so text stays readable on light surfaces |

A variant that is already set, whether in `overrides`, the theme, or a preset, is kept as is. Derivation covers hex (`#e63946`) and `oklch(L C H)` values. For any other format, such as `hsl()` or a color name, set the four variants yourself. See [Theme Tokens](/reference/theme-tokens/#accent-derivation) for the exact formulas.

## Token cascade

Tokens resolve through four layers. Later layers win:

1. **Embedded defaults** compiled into the binary
2. **Theme tokens** from `theme.yaml` of the active theme
3. **Preset tokens** from the selected preset
4. **User overrides** from `theme.overrides` in `sarde.yaml`

Light and dark tokens run through the same four layers, using each layer's `dark_tokens` for dark mode. An empty value never replaces a value from a lower layer.

## Dark mode

The header carries a three-position toggle: light, system, and dark.

- **Light** forces the light theme
- **System** follows the operating system's `prefers-color-scheme`
- **Dark** forces the dark theme

The choice is saved in `localStorage` under `sd-theme` and applied as a `data-theme` attribute on the `<html>` element, set before first paint. Dark rules in the theme CSS use `:root[data-theme="dark"]` selectors, so use the same selector in your own CSS.

To remove the toggle, [eject](#theme-eject) the `Header` component and delete its `ThemeToggle` call.

## CSS layer order

Sarde places its CSS in `@layer` blocks, declared in this order. Later layers override earlier ones:

| Layer | Contents |
|-------|----------|
| `sarde.base` | Design tokens |
| `sarde.reset` | Reset, base typography, dark mode rules outside the token set |
| `sarde.core` | Layout grid |
| `sarde.content` | Prose, site chrome, blog, taxonomy, slides, and homepage styles |
| `sarde.components` | UI components, Markdown extensions, search |
| `sarde.variants` | Layout variants, such as labs and the galaxy aside style |
| `sarde.utils` | Utility classes and print styles |
| `sarde.plugins` | Styles contributed by plugins |
| `sarde.user` | Reserved for your own CSS. Nothing ships in it |

CSS from `head.custom_css` loads after the theme stylesheet. Rules outside any layer beat layered rules, so custom CSS overrides every Sarde style without `!important`.

To keep your rules inside the cascade instead, wrap them in the reserved layer:

```css title="assets/css/custom.css"
@layer sarde.user {
  .sarde-sidebar { border-inline-end: 1px solid var(--sd-border); }
}
```

Rules in `sarde.user` beat every Sarde layer, including plugin styles.

## Theme eject

To change a template, component, or stylesheet beyond what tokens allow, eject that file and edit the copy:

```sh
sarde theme eject layouts/components/Header.html
sarde theme eject css/components.css
```

Sarde writes the files to `themes/default/`, where they override their embedded counterparts. If the destination has no `theme.yaml`, Sarde copies one so the bundled presets stay selectable. Everything not ejected keeps coming from the embedded theme, including fixes in later Sarde releases.

Run `sarde theme eject --list` to see every ejectable path. Run `sarde theme eject` with no paths to copy the whole theme. See [`theme eject`](/reference/cli-commands/#theme-eject) for all options.

:::note
An ejected file is a snapshot. Sarde updates do not touch it, so eject only what you change and merge upstream changes by hand after upgrading.
:::

### The theme `css/` directory

A theme's stylesheets live in `themes/<name>/css/`. Sarde loads 24 stylesheets in a fixed order. Each one comes from the theme when present and from the embedded theme otherwise, so a theme can ship a single edited stylesheet. Files outside that set are ignored.

:::caution
`assets/` does not work this way. A theme `assets/` directory replaces the embedded fonts, vendor scripts, and JS entirely. Eject it whole with `sarde theme eject assets`.
:::

To change a few rules, prefer `head.custom_css` or the `sarde.user` layer.

## Content width toggle

Docs pages show a header button that switches the article between the full available width and a centered column. The default is the full width. The centered column is capped at the `docs-centered-width` token (`87.5rem`).

The button appears only on the docs layout and only at viewports of 1280px or wider. The centered choice is stored in `localStorage` under `sd-docs-centered` and applied as a class of the same name on `<html>`, so it survives navigation and reloads.

To remove the button, eject the `Header` component and delete the `CenterToggle` call. To restyle it, target `.sarde-center-toggle`.

See [Theme Tokens](/reference/theme-tokens/) for the complete token reference and [Configuration](/reference/configuration/theme-and-appearance/#theme) for all theme settings.
