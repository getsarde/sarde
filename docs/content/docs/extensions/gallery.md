---
title: Gallery
description: "Display multiple images in a responsive grid with a built-in full-screen lightbox"
sidebar:
  order: 17
  group: Block
---

Galleries display multiple images in a responsive grid with a built-in lightbox for full-screen viewing.

## Basic syntax

Place standard Markdown images inside a `:::gallery` block:

````
:::gallery
![Leaf cross-section](/images/leaf-cross.jpg)
![Root tip cells](/images/root-tip.jpg)
![Flower anatomy](/images/flower-anatomy.jpg)
![Pollen grain](/images/pollen-grain.jpg)
:::
````

→ A grid of image thumbnails appears. Each image shows a zoom icon overlay on hover. Clicking an image opens the lightbox.

<!-- SCREENSHOT: gallery-grid — four images in a responsive grid layout -->

## Gallery label

Add a label in square brackets to display a title above the grid:

````
:::gallery[Plant Cell Structures]
![Cell wall](/images/cell-wall.jpg)
![Chloroplast](/images/chloroplast.jpg)
![Vacuole](/images/vacuole.jpg)
:::
````

→ The text "Plant Cell Structures" appears as a heading above the image grid.

## Lightbox

Clicking any image opens a full-screen lightbox overlay with:

- The image displayed at full resolution
- The image's alt text as a caption below the image
- A counter showing the current position (e.g., "2 / 4")
- Previous and next navigation buttons
- A close button

<!-- SCREENSHOT: gallery-lightbox — the lightbox overlay showing an image with caption and navigation -->

## Keyboard navigation

The lightbox supports keyboard controls:

| Key | Action |
|-----|--------|
| Arrow Left | Previous image |
| Arrow Right | Next image |
| Escape | Close the lightbox |
| Tab | Cycle focus between lightbox buttons |

Focus returns to the thumbnail that opened the lightbox when it closes.

## Accessibility

Each thumbnail has `role="button"` and `tabindex="0"`, making it reachable by keyboard. The lightbox uses `role="dialog"` and `aria-modal="true"`. Pressing Enter or Space on a focused thumbnail opens the lightbox.

## Options

| Option | Syntax | Default | Description |
|--------|--------|---------|-------------|
| Label | `:::gallery[Label text]` | — | Title displayed above the grid. |

## Edge cases

- A gallery with no images renders an error message: "Gallery requires at least one image."
- Images are lazy-loaded (`loading="lazy"`) in the grid.
- The lightbox wraps around: navigating past the last image returns to the first.
- Alt text from the Markdown image syntax is used as both the `alt` attribute and the lightbox caption. The optional title from `![alt](src "title")` is not used.
- Multiple galleries on the same page each have their own independent lightbox instance.
