---
title: Images and Assets
description: "Place and link images, PDFs, and other assets from Markdown. Configure responsive image generation, LQIP placeholders, and CSS/JS bundling."
sidebar:
  order: 9
---

Sarde handles assets in two ways: page bundles, which sit next to the page and get responsive image processing, and the `public/` directory, which is copied as is. This page covers where to place assets, how to link them from Markdown, and how to configure image processing and CSS/JS bundling.

## Placing assets

The location decides the path style and whether Sarde optimizes an image. Put an asset next to the page that uses it, or in `public/` for files shared across the site.

### Page bundles

A [page bundle](/guides/content-and-collections/#page-bundles) is a directory with an `index.md` file and sibling non-Markdown files. The sibling files become assets of that page.

```text
content/blog/
  my-post/
    index.md          # Page content
    cover.jpg         # Bundle asset
    diagram.svg       # Bundle asset
    report.pdf        # Bundle asset
```

Reference bundle assets by filename, relative to the page:

```markdown
![Cover image](cover.jpg)
![Diagram](diagram.svg)
[Download the report](report.pdf)
```

Raster images referenced this way go through the responsive image pipeline, which produces several widths, converts them to the configured formats, and adds a blurred placeholder. SVGs are not processed and render as a plain `<img>`. Sarde also copies every bundle file, including the original image, to the output next to the page's `index.html`. A Markdown image that matches no file in the page's bundle renders as a plain `<img>` with the path as written.

### The `public/` directory

Files in [`public/`](/guides/project-structure/#public) are copied to the output directory without processing. Use it for images, fonts, favicons, and any file shared across pages.

```text
public/
  images/
    logo.png
  files/
    brochure.pdf
  favicon.svg
```

Reference these with absolute paths:

```markdown
![Logo](/images/logo.png)
[Download brochure](/files/brochure.pdf)
```

Images from `public/` are not processed. They render as a plain `<img>` with the original file.

### Which to use

Choose the placement by how widely the file is used:

| Scenario | Placement | Path style |
|----------|-----------|------------|
| Image for one specific page | Page bundle (next to `index.md`) | `![alt](photo.jpg)` |
| Site logo, favicon, shared icons | `public/` directory | `![alt](/images/logo.png)` |
| Logo in the site header | `public/` directory | [`site.logo`](/guides/branding/) config |
| PDF download for one page | Page bundle | `[Download](report.pdf)` |
| PDF shared across the site | `public/` directory | `[Download](/files/report.pdf)` |

## Responsive image generation

During `sarde build`, each page bundle image becomes a `<picture>` element with a `<source>` per format and a `srcset` of widths. Configure the defaults in `sarde.yaml`:

```yaml title="sarde.yaml"
images:
  widths: [400, 800, 1200]
  formats: ["webp"]
  quality: 80
  max_width: 2400
  lazy_loading: true
  placeholder: "lqip"
```

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `widths` | int[] | `[400, 800, 1200]` | Widths of the responsive variants. A width at or above the source width is skipped. |
| `formats` | string[] | `["webp"]` | Output formats: `jpeg`, `png`, `webp`, or `avif`. |
| `quality` | int | `80` | Encoding quality, 1 to 100, for lossy formats. |
| `max_width` | int | `2400` | Skips any entry of `widths` above this value. Does not shrink the full-size variant. |
| `lazy_loading` | bool | `true` | Adds `loading="lazy" decoding="async"` to `<img>` tags. The first image on a page always loads eagerly. |
| `placeholder` | string | `"lqip"` | `"lqip"` embeds a blurred preview. `"none"` disables it. |

Sarde also writes a variant at the source's full width, in each configured format. For a 1600px source with the defaults, Sarde generates WebP files at 400, 800, 1200, and 1600 pixels wide. A 300px source produces only its full-width WebP. A 3000px source keeps its 3000px variant even though `max_width` is 2400.

`sarde dev` skips image processing and placeholder generation. Images render from their original files, with `width` and `height` still read from the file.

## LQIP placeholders

A Low Quality Image Placeholder (LQIP) is a blurred preview that shows while the full image loads. Sarde resizes the image to 20px wide, blurs it, encodes it as a JPEG data URI, and sets it as the `<img>` element's `background-image`.

To disable it:

```yaml title="sarde.yaml"
images:
  placeholder: "none"
```

## `<picture>` element output

Each format gets a `<source>` element, listed AVIF first, then WebP, then JPEG or PNG. The `<img>` fallback uses the generated variant closest to 800px wide.

```html
<picture>
  <source type="image/webp"
    srcset="/assets/images/hero-a1b2-400w.webp 400w,
           /assets/images/hero-c3d4-800w.webp 800w,
           /assets/images/hero-e5f6-1200w.webp 1200w,
           /assets/images/hero-a7b8-1600w.webp 1600w"
    sizes="(max-width: 600px) 400px, (max-width: 1024px) 800px, 1200px">
  <img src="/assets/images/hero-c3d4-800w.webp" alt="Course hero image"
    width="1600" height="900" loading="lazy" decoding="async"
    style="background-image: url(data:image/jpeg;base64,...); background-size: cover;">
</picture>
```

The `<img>` always carries `width` and `height` from the source file, and every image gets the same `sizes` value.

## Format support

Sarde can write four image formats:

| Format | Status | Notes |
|--------|--------|-------|
| JPEG | Built-in | Output only when listed in `formats`. |
| PNG | Built-in | Lossless. |
| WebP | Built-in | Default output format. |
| AVIF | Build tag | Requires a binary built with `go build -tags avif`. Otherwise Sarde logs a warning and skips AVIF variants. |

List several formats to produce variants in each. The browser picks the first `<source>` it supports:

```yaml title="sarde.yaml"
images:
  formats: ["avif", "webp"]
```

## Image disk cache

Sarde caches processed variants in `.cache/images/` under the project root. The cache key combines the source image hash with the widths, quality, formats, resize operation, maximum width, and placeholder mode. Changing any of these settings regenerates the affected images.

Later builds copy cached variants to the output directory without reprocessing. Delete `.cache/images/` to regenerate every variant.

## CSS and JS bundling

Sarde bundles the files listed in `head.custom_css` and `head.custom_js` with esbuild. Each entry is a path relative to an `assets/` directory, and `@import` and `import` statements resolve through the same lookup, in this order:

1. `assets/` in the project root
2. `themes/<name>/assets/` in the active theme
3. Embedded theme assets, compiled into the binary

```yaml title="sarde.yaml"
head:
  custom_css:
    - "css/custom.css"
  custom_js:
    - "js/analytics.js"
```

Put the files at `assets/css/custom.css` and `assets/js/analytics.js`. Each entry produces one output file. An entry that Sarde cannot find, including a full URL, fails the build. To load an external stylesheet or script, use a `head.tags` entry instead.

Production builds minify the output unless `build.minify` is `false`. `sarde dev` skips minification and adds inline source maps.

### Fingerprinted filenames

Production builds add a content hash to each bundled filename for cache busting:

| Mode | Filename |
|------|----------|
| Dev | `custom.css` |
| Production | `custom.a1b2c3d4.css` |

The hash is the first 8 hex characters of the SHA-256 digest of the output file. Changing the file produces a new hash and bypasses browser caches.

## `resize_image` template function

Use `resize_image` in a template to process a page bundle resource with custom parameters:

```html
{{ $img := getResource .Resources "hero.jpg" }}
{{ resize_image $img "width=800&quality=85&format=webp" }}
```

The function takes a resource and a query string:

| Parameter | Type | Description |
|-----------|------|-------------|
| `width` | int | Target width in pixels. Replaces the configured `widths` for this image. |
| `height` | int | Target height, used with `fill` and `fit`. |
| `op` | string | Resize operation: `scale`, `fit_width`, `fit_height`, `fit`, or `fill`. |
| `quality` | int | Encoding quality for this image. |
| `format` | string | Output format: `jpeg`, `png`, `webp`, or `avif`. |

The function returns a `<picture>` element. In `sarde dev`, no variants are generated and it returns a plain `<img>` with the original source.
