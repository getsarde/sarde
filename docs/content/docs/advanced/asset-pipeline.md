---
title: Asset Pipeline
description: "CSS/JS bundling with esbuild, image optimization, fingerprinting, and the responsive image pipeline."
sidebar:
  order: 1
---

The asset pipeline handles two independent jobs: bundling hand-written CSS/JS through esbuild, and processing Markdown images into responsive variants. Both are content-addressed and cached on disk to keep rebuilds fast.

## CSS/JS bundling

Files listed under `head.custom_css` and `head.custom_js` in `sarde.yaml` are bundled through esbuild. The bundler resolves `@import` and `import` statements through a 3-layer lookup, in priority order:

1. `assets/` in the project root
2. `themes/<name>/assets/` in the active theme
3. Embedded theme assets (compiled into the binary)

The first layer that has a matching file wins, so a project-level file always overrides the same path in a theme or the embedded defaults.

```yaml
head:
  custom_css:
    - "css/custom.css"
  custom_js:
    - "js/analytics.js"
```

In dev mode, esbuild adds inline source maps for debugging and skips minification. Production builds minify output and embed a fingerprint in the filename.

## Fingerprinted filenames

Production builds embed a content hash in output filenames for cache-busting:

| Mode | Filename |
|------|----------|
| Dev | `main.css` |
| Production | `main.a1b2c3d4.css` |

The hash is the first 8 hex characters of the SHA-256 digest of the file content. Changing the file produces a new hash, which busts browser caches. An asset manifest tracks the mapping from original paths to fingerprinted URLs, and template functions like `fingerprint` and `asset` consult it to emit the correct URL.

## Image processing pipeline

Images referenced with a relative path in Markdown (page bundle images) are processed into responsive variants during `sarde build`. For each configured width below the image's original size, Sarde generates a resized copy; for each configured output format, each width is encoded separately. The result is a `<picture>` element with one `<source>` per format and a `srcset` listing all widths.

Resize operations, selectable via the `op` parameter:

| Operation | Behavior |
|-----------|----------|
| `scale` | Proportional width resize. Default. |
| `fit_width` | Resize to a target width, preserving aspect ratio. |
| `fit_height` | Resize to a target height, preserving aspect ratio. |
| `fit` | Resize to fit within a width x height bounding box, preserving aspect ratio. |
| `fill` | Crop to exactly width x height, center-cropped. |

Images are never upscaled past their original width: a 300px-wide source with `widths: [400, 800, 1200]` produces only the original-size variant. SVG files are not processed and pass through as-is.

Dev mode skips all image processing. Images render with their original source paths so that `sarde dev` rebuilds stay fast; dimensions are still read from the file header for layout-shift prevention.

## LQIP placeholders

Low Quality Image Placeholders provide a blurred preview while the full image loads. Sarde generates a 20px-wide JPEG, blurs it, and base64-encodes it as a data URI applied via `background-image` on the rendered `<img>` element. Set `placeholder: "none"` in the `images` config to disable LQIP generation.

## Image disk cache

Processed image variants are cached in `.cache/images/` under the project root. The cache key combines the source image's content hash with the processing parameters (widths, quality, formats, resize operation, max width, placeholder mode), so changing any of these settings invalidates the cache only for affected images.

On subsequent builds, cached variants are copied directly to the output directory without re-processing. Delete `.cache/images/` to force a full rebuild of all image variants.

Cache writes are atomic (temp file plus rename) and guarded by a per-destination-path lock, since content-addressed filenames mean two concurrent builds of the same image would otherwise race on the same output path.

## AVIF support

AVIF encoding requires building Sarde with `go build -tags avif`. If `avif` is configured as an output format but the build does not include AVIF support, Sarde logs a warning once and skips AVIF variants for that build, falling back to the other configured formats.
