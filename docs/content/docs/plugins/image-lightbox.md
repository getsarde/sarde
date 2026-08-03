---
title: Image Lightbox
description: "Open content images in a full-screen overlay with zoom and pan controls"
sidebar:
  order: 14
---

Clicking an image in the content area opens it in a full-screen overlay with zoom and pan controls. Disabled by default.

## Enable the plugin

`sarde.yaml`
```yaml
plugins:
  enabled:
    - search
    - seo
    - sitemap
    # ... other default plugins
    - image_lightbox
```

## How it works

Images inside the content area display a zoom-in cursor on hover. Clicking opens the image in a dark overlay with a toolbar for zoom controls and a close button. The image's `alt` text appears as a caption below the image.

Lightboxable images are also keyboard-accessible: they are focusable with ::kbd[Tab], and ::kbd[Enter] or ::kbd[Space] opens the lightbox. While open, focus moves to the close button, ::kbd[Tab] cycles through the toolbar controls, and closing returns focus to the image that opened it.

### Zoom and pan

| Action | Effect |
|--------|--------|
| Scroll wheel | Zoom in (up) or out (down). |
| Toolbar buttons | Zoom in, zoom out, or click the percentage readout to reset. |
| ::kbd[+] / ::kbd[=] | Zoom in. |
| ::kbd[-] | Zoom out. |
| ::kbd[0] | Reset zoom and pan to default. |
| Click and drag | Pan the image when zoomed beyond 100%. |

Zoom ranges from 50% to 400% in 25% steps.

### Closing the lightbox

- Click the close button (top-right).
- Click the dark backdrop outside the image.
- Press ::kbd[Esc].

Page scrolling is locked while the lightbox is open and restored on close.

## Configuration

`sarde.yaml`
```yaml
plugins:
  config:
    image_lightbox:
      background_opacity: 0.92
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `background_opacity` | Number | `0.92` | Overlay darkness. `0.5` is semi-transparent, `1.0` is fully opaque. Range: 0.5 to 1.0. |

## Injection rule

This plugin activates on pages containing images (`has_images`). Pages without images receive no plugin config or behavior.

## Excluded images

Not all images open in the lightbox:

- Images inside a `:::gallery` extension are excluded (the gallery has its own lightbox).
- Images with the `no-lightbox` class are excluded.
- Small images (under 80 pixels wide, once loaded) are excluded to avoid lightboxing icons or decorative graphics.

## Edge cases

- If an image has a `data-src` attribute, the lightbox opens that URL instead of the rendered `src`. This supports lazy-loading setups where the full-resolution source is deferred.
- Images without `alt` text show no caption bar in the lightbox.
- Touch devices support tap-to-open and pinch-drag panning. Touch events prevent default scrolling while the lightbox is open.
- The lightbox overlay is created lazily on first click, not at page load.
- There is no next/previous image navigation. Each image opens independently.
- The fade-in animation and zoom transition respect `prefers-reduced-motion`.
