---
title: Branding
description: "Add a site logo to the header and set a favicon. Covers light and dark variants, sizing, image formats, and favicon auto-detection."
sidebar:
  order: 14
---

Sarde renders a site logo in the header, immediately before the site title. Point `site.logo` at a file in `public/`:

```yaml
site:
  logo: /images/logo.svg
```

## Site logo

The image lives anywhere under `public/`:

```text
public/
  images/
    logo.svg
```

The config path drops the `public/` segment, so that file is `/images/logo.svg`. Paths resolve against `build.base_path`, so a logo works unchanged on a subpath deploy.

### Light and dark variants

The object form takes a separate image per theme:

```yaml
site:
  logo:
    light: /images/logo-light.svg
    dark: /images/logo-dark.svg
    alt: My Site
```

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `light` | string | `""` | Logo for light mode. |
| `dark` | string | `""` | Logo for dark mode. |
| `alt` | string | `""` | Alt text for the image. |
| `replaces_title` | bool | `false` | Hide the site title text so only the logo shows. |

Declaring only `light` or only `dark` uses that image for both themes. The string form does the same.

When both keys resolve to the same image, Sarde emits one `<img>`. When they differ, it emits both and swaps them with CSS on the `data-theme` attribute. That attribute is set before first paint, so the correct variant renders immediately with no flash.

:::caution
Artwork with a transparent background disappears against one of the two themes. Give the image its own background, or supply both variants.
:::

### Replacing the site title

Set `replaces_title` to show the logo alone:

```yaml
site:
  logo:
    light: /images/wordmark.svg
    alt: My Site
    replaces_title: true
```

The title text stays in the DOM with the theme's `sr-only` class, so the header link keeps its accessible name. Set a meaningful `alt` when the logo is the only visible branding.

With the title text visible, an empty `alt` is correct. The text beside the image already names the link, and alt text would make screen readers announce the site name twice.

### Sizing the logo

Logo height comes from the `logo-height` token, not a config key. It defaults to `1.75rem` and shrinks to 85 percent of that below 768px. Override it like any other token:

```yaml
theme:
  overrides:
    logo-height: "2.25rem"
```

See [Theme Tokens](/reference/theme-tokens#layout) for the full layout token list.

### Image formats

| Format | Dimensions |
|--------|------------|
| PNG, JPEG, GIF, WebP | Read from the file at build time |
| SVG | Sized by CSS |

For raster formats, Sarde reads the intrinsic dimensions during the build and emits `width` and `height` attributes, which prevents layout shift while the image loads. SVG has no reliable intrinsic pixel size, so CSS sizes it alone.

Files in `public/` bypass the responsive image pipeline, so a raster logo gets no generated `srcset`. Supply it at two to three times the rendered height to stay sharp on high-density displays.

One more format consideration: the [Social Cards plugin](/plugins/social-cards#logo-and-watermark) reuses `site.logo` to brand generated Open Graph images, but it can only composite raster formats. An SVG header logo works fine in the header while the cards silently render without it. To get both, keep the SVG for the header and point `social_cards.logo` at a PNG export.

## Favicon

Point `site.favicon` at a file in `public/`:

```yaml
site:
  favicon: /favicon.svg
```

Sarde sets the `type` attribute from the extension: `image/svg+xml`, `image/x-icon`, or `image/png`.

### Auto-detection

With `site.favicon` unset, Sarde looks in `public/` for `favicon.svg`, then `favicon.ico`, then `favicon.png`, and uses the first one found. A project that follows that naming needs no favicon config at all.

Set `site.favicon` explicitly to use a different filename or a file in a subdirectory.

## Where branding assets live

Both the logo and the favicon are read from `public/`, which Sarde copies to the output directory without processing. One file can serve as both when the artwork reads at header size and at 16px.

A configured file missing from `public/` logs a build warning and leaves the rest of the build untouched.

See [Project Structure](/guides/project-structure#public) for what else belongs in `public/`, and [Configuration](/reference/configuration#site) for every `site` key.
