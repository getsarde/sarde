---
title: Images and Assets
description: "Place and link images, PDFs, and other assets from Markdown. Configure responsive image generation, LQIP placeholders, and CSS/JS bundling."
sidebar:
  order: 9
---

Sarde has two ways to include assets in your site: co-located page bundles (with automatic responsive image processing) and the `public/` directory (copied as-is). This page covers where to place assets, how to link them in Markdown, and how to configure the image processing pipeline.

## Placing assets

### Page bundles (co-located assets)

A [page bundle](/guides/content-and-collections#page-bundles) is a directory with an `index.md` file and sibling non-Markdown files. The sibling files become assets of that page.

```text
content/blog/
  my-post/
    index.md          # Page content
    cover.jpg          # Bundle asset
    diagram.svg        # Bundle asset
    report.pdf         # Bundle asset
```

Reference bundle assets with a bare relative filename in the Markdown:

```markdown
![Cover image](cover.jpg)
![Diagram](diagram.svg)
[Download the report](report.pdf)
```

Images referenced this way go through the responsive image pipeline automatically, producing multiple widths, WebP variants, and LQIP placeholders. SVGs pass through unprocessed.

### `public/` directory (site-wide assets)

Files in [`public/`](/guides/project-structure#public) are copied to the output directory without processing. Use it for images, fonts, favicons, or any file shared across pages.

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

Images in `public/` are **not** processed through the responsive pipeline. They render as plain `<img>` tags with the original file.

### Which to use

| Scenario | Placement | Path style |
|----------|-----------|------------|
| Image for one specific page | Page bundle (next to `index.md`) | `![alt](photo.jpg)` |
| Site logo, favicon, shared icons | `public/` directory | `![alt](/images/logo.png)` |
| PDF download for one page | Page bundle | `[Download](report.pdf)` |
| PDF shared across the site | `public/` directory | `[Download](/files/report.pdf)` |

## Per-image overrides

Override the default responsive settings on individual images using Goldmark attributes:

```markdown
![Hero](hero.jpg){width=800 op=fill format=webp quality=90}
```

| Attribute | Type | Description |
|-----------|------|-------------|
| `width` | int | Target width in pixels. |
| `height` | int | Target height (used with `fill` and `fit` operations). |
| `op` | string | Resize operation: `scale`, `fit_width`, `fit_height`, `fit`, `fill`. |
| `quality` | int | Override encoding quality (1-100). |
| `format` | string | Output format: `jpeg`, `png`, `webp`, `avif`. |

Per-image overrides only apply to page bundle images. Images from `public/` are not processed.

## Responsive image generation

Markdown images are automatically processed into multiple widths and formats during `sarde build`. Each image produces a `<picture>` element with `<source>` entries and a `srcset` attribute.

Configure the default widths, formats, and quality in `sarde.yaml`:

```yaml
images:
  widths: [400, 800, 1200]
  formats: ["webp"]
  quality: 80
  max_width: 2400
  lazy_loading: true
  dimensions: true
```

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `widths` | int[] | `[400, 800, 1200]` | Breakpoint widths for responsive variants. Images are never upscaled past their original width. |
| `formats` | string[] | `["webp"]` | Output formats. Options: `jpeg`, `png`, `webp`, `avif`. |
| `quality` | int | `80` | Encoding quality (1-100) for lossy formats. |
| `max_width` | int | `2400` | Maximum output width. Variants wider than this are skipped. |
| `lazy_loading` | bool | `true` | Add `loading="lazy" decoding="async"` to `<img>` tags. |
| `dimensions` | bool | `true` | Include `width` and `height` attributes to prevent layout shift. |
| `placeholder` | string | `"lqip"` | Placeholder strategy: `"lqip"` or `"none"`. |

For a 1600px-wide source image with the defaults above, Sarde generates three WebP variants (400w, 800w, 1200w) plus the original at 1600w.

## LQIP placeholders

Low Quality Image Placeholders (LQIP) provide a blurred preview while the full image loads. Sarde generates a 20px-wide JPEG, blurs it, and base64-encodes it as a data URI applied via `background-image`.

Result: A blurred version of the image appears instantly, then fades to the sharp version once loaded.

Set `placeholder: "none"` to disable LQIP generation:

```yaml
images:
  placeholder: "none"
```

## `<picture>` element output

The generated HTML uses `<picture>` with `<source>` elements for each format, ordered by priority (AVIF first, then WebP, then original format). The `<img>` fallback uses the variant closest to 800px.

```html
<picture>
  <source type="image/webp"
    srcset="/assets/images/hero-a1b2-400w.webp 400w,
           /assets/images/hero-a1b2-800w.webp 800w,
           /assets/images/hero-a1b2-1200w.webp 1200w"
    sizes="(max-width: 600px) 400px, (max-width: 1024px) 800px, 1200px">
  <img src="/assets/images/hero-a1b2-800w.webp" alt="Course hero image"
    width="1600" height="900" loading="lazy" decoding="async"
    style="background-image: url(data:image/jpeg;base64,...); background-size: cover;">
</picture>
```

The `sizes` attribute is computed from the configured `widths` array.

## Format support

| Format | Status | Notes |
|--------|--------|-------|
| JPEG | Built-in | Default fallback format. |
| PNG | Built-in | Lossless. |
| WebP | Built-in | Default output format. Good compression with broad browser support. |
| AVIF | Build tag | Requires `go build -tags avif`. If configured but unavailable, Sarde logs a warning and skips AVIF variants. |

Add multiple formats to produce variants in each:

```yaml
images:
  formats: ["avif", "webp"]
```

The `<picture>` element lists formats in priority order (AVIF, WebP, then original), and the browser picks the first it supports.

## Image disk cache

Processed image variants are cached in `.cache/images/` under the project root. The cache key combines the source image hash with the processing parameters (widths, quality, formats, resize operation, max width, placeholder mode). Changing any of these settings invalidates the cache for affected images.

On subsequent builds, cached variants are copied directly to the output directory without re-processing. Delete `.cache/images/` to force a full rebuild of all image variants.

## CSS/JS bundling

CSS and JS files listed in `head.custom_css` and `head.custom_js` are bundled through esbuild. The bundler resolves `@import` and `import` statements through a 3-layer lookup:

1. `assets/` in the project root
2. `themes/<name>/assets/` in the active theme
3. Embedded theme assets (compiled into the binary)

```yaml
head:
  custom_css:
    - "css/custom.css"
  custom_js:
    - "js/analytics.js"
```

Place the files in `assets/css/custom.css` and `assets/js/analytics.js`. The bundler processes imports, resolves dependencies, and outputs a single file per entry point.

In dev mode, esbuild adds inline source maps for debugging. Minification is skipped.

## Fingerprinted filenames

Production builds embed a content hash in output filenames for cache-busting:

| Mode | Filename |
|------|----------|
| Dev | `main.css` |
| Production | `main.a1b2c3d4.css` |

The hash is the first 8 hex characters of the SHA-256 digest of the file content. Changing the file produces a new hash, which busts browser caches. The asset manifest tracks the mapping from original paths to fingerprinted URLs for template use.

## `resize_image` template function

Use `resize_image` in templates to process a page bundle resource with custom parameters:

```go
{{ $img := getResource .Resources "hero.jpg" }}
{{ resize_image $img "width=800&quality=85&format=webp" }}
```

The function accepts a `Resource` and a query string with these parameters:

| Parameter | Type | Description |
|-----------|------|-------------|
| `width` | int | Target width in pixels. |
| `height` | int | Target height (used with `fill` and `fit` operations). |
| `op` | string | Resize operation: `scale`, `fit_width`, `fit_height`, `fit`, `fill`. |
| `quality` | int | Override encoding quality. |
| `format` | string | Output format: `jpeg`, `png`, `webp`, `avif`. |

The function returns a `<picture>` HTML element with responsive variants. When no image processor is available in dev mode, it falls back to a plain `<img>` tag with the original source.

## Edge cases

- Images smaller than a configured width are not upscaled. A 300px-wide image with `widths: [400, 800, 1200]` produces only the original-size variant.
- SVG files are not processed. They pass through as-is.
- Dev mode (`sarde dev`) skips all image processing and LQIP generation. Images render with their original source paths. Dimensions are still read from the file header.
- Concurrent builds use per-path locks to prevent race conditions when writing identical output filenames.
