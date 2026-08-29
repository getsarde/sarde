---
title: Creating a Preset
description: "Define a named token set in theme.yaml that a site can switch to with theme.preset"
sidebar:
  order: 3
---

A preset is a named set of token values inside a theme. A site switches to it with one line in `sarde.yaml`, which makes presets the lightest way to offer a site several looks without touching templates or CSS.

## Where presets live

Presets are declared in a theme's `theme.yaml` under `presets:`, and only the active theme's presets can be selected. The embedded theme ships its own set (`ocean`, `forest`, `rose`, and the rest, listed under [Choosing a preset](/guides/themes-and-styling/#choosing-a-preset)). Adding a preset therefore means editing a theme directory:

1. If the site already uses a theme from `themes/`, add the preset to that theme's `theme.yaml`.
2. If the site uses the embedded default, create a one-file theme to hold the preset: `themes/<name>/theme.yaml` with `name`, `slug`, and `presets`, then set `theme.name` in `sarde.yaml`. This replaces the embedded preset list, so copy over any built-in presets worth keeping from an ejected `themes/default/theme.yaml`.

The rest of this page assumes a theme named `magazine`; see [Using Themes](/customization/using-themes/) for how a theme is activated.

## Define a preset

The smallest useful preset sets one token. Add to `themes/magazine/theme.yaml`:

```yaml
presets:
  sunset:
    tokens:
      accent: "oklch(0.65 0.19 45)"
    dark_tokens:
      accent: "oklch(0.75 0.17 45)"
```

Setting `accent` is enough for a coherent look: Sarde derives `accent-hover`, `accent-high`, `accent-low`, and `accent-text` from it for light mode.

Activate the preset in `sarde.yaml`:

```yaml
theme:
  name: "magazine"
  preset: "sunset"
```

→ Links, buttons, and highlights turn orange in both light and dark mode. Everything else keeps the theme's base values.

:::caution
A preset key that does not exist in the active theme is ignored without a warning. If nothing changes after setting `theme.preset`, run `sarde theme info magazine` and compare the key against the `Presets:` line.
:::

## What a preset can set

`tokens` and `dark_tokens` accept any token name from [Theme Tokens](/reference/theme-tokens/). Two shapes cover most presets.

A palette preset changes the accent and tints the gray scale toward the same hue, so surfaces, borders, and muted text pick up the color as well. This is the embedded `ocean` preset:

```yaml
presets:
  ocean:
    name: Ocean
    tokens:
      accent: "oklch(0.67 0.19 211)"
      gray-1: "oklch(0.97 0.008 196)"
      gray-2: "oklch(0.92 0.010 196)"
      gray-3: "oklch(0.84 0.012 196)"
      gray-4: "oklch(0.66 0.012 196)"
      gray-5: "oklch(0.52 0.010 196)"
      gray-6: "oklch(0.35 0.012 196)"
      gray-7: "oklch(0.21 0.016 196)"
    dark_tokens:
      accent: "oklch(0.75 0.17 211)"
```

A full-look preset also sets backgrounds, text colors, borders, fonts, and radius:

```yaml
presets:
  paper:
    name: Paper
    tokens:
      accent: "oklch(0.45 0.10 60)"
      bg: "oklch(0.985 0.005 80)"
      bg-surface: "oklch(0.96 0.008 80)"
      text: "oklch(0.25 0.02 60)"
      text-muted: "oklch(0.50 0.02 60)"
      border: "oklch(0.88 0.01 80)"
      font-sans: "'Source Serif 4', Georgia, serif"
      font-mono: "'JetBrains Mono', Consolas, monospace"
      radius: "0.25rem"
    dark_tokens:
      accent: "oklch(0.72 0.10 60)"
      bg-surface: "oklch(0.20 0.01 60)"
      text: "oklch(0.93 0.01 80)"
      text-muted: "oklch(0.64 0.01 80)"
      border: "oklch(0.32 0.01 60)"
```

→ The site reads as warm paper: cream background, brown-black serif text, muted amber links.

| Shape | Tokens it touches |
|-------|-------------------|
| Palette | `accent`, `gray-1` through `gray-7`, dark `accent` |
| Full look | The palette tokens plus `bg`, `bg-surface`, `text`, `text-muted`, `border`, `font-sans`, `font-mono`, `radius` |

The `name` inside a preset is an optional label; `theme.preset` references the key (`ocean`, `paper`), not the label.

## Light and dark

Tokens resolve in this order, last wins: embedded defaults, the theme's `tokens`, the preset's `tokens`, then the site's `theme.overrides`. Dark mode follows the same order with the `dark_tokens` maps and `theme.dark_overrides`; see the [token cascade](/guides/themes-and-styling/#token-cascade).

A preset's `dark_tokens` merge over the theme's `dark_tokens`, so a preset that omits the map keeps the theme's dark values. Accent derivation runs on light tokens only. Set the dark `accent` explicitly, and set `accent-hover`, `accent-high`, and `accent-low` under `dark_tokens` too if the defaults do not suit the new hue.

## Check the result

`sarde dev` rebuilds the site whenever `theme.yaml` changes, so switching `theme.preset` or editing a value shows up on the next reload.

Confirm the theme exposes the preset:

```sh
sarde theme info magazine
```

```
Name:          Magazine
Slug:          magazine
Version:       1.0.0
Presets:       ocean, paper, sunset
```

To bundle presets with templates and stylesheets as a distributable package, continue with [Creating a Theme](/customization/creating-a-theme/).
