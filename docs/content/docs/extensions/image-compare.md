---
title: Image Compare
sidebar:
  order: 18
  group: Block
---

Image Compare displays two images side by side with a draggable slider, letting readers compare a "before" and "after" view.

## Basic syntax

Place two Markdown images inside an `:::image-compare` block. The first image is the "before" (left), the second is the "after" (right):

````
:::image-compare
![Original photo](/images/photo-original.jpg)
![Enhanced photo](/images/photo-enhanced.jpg)
:::
````

→ Both images overlap in the same container. A vertical slider divides them at the 50% mark. Dragging the slider reveals more of one image and less of the other.

<!-- SCREENSHOT: image-compare-slider — a before/after comparison with the slider at center -->

## Label

Add a label in square brackets to display a title above the comparison:

````
:::image-compare[Before and after color correction]
![Before](/images/color-before.jpg)
![After](/images/color-after.jpg)
:::
````

→ The text "Before and after color correction" appears above the slider container.

## How the slider works

The slider starts at 50%. The "before" image is fully visible on the left side of the divider. The "after" image is clipped to the right side using CSS `clip-path`.

- **Mouse**: Click and drag anywhere in the container to move the slider.
- **Touch**: Drag on mobile and tablet devices.
- **Keyboard**: Focus the slider handle and use arrow keys.

The alt text from each image is displayed as labels on the left ("before") and right ("after") sides of the container.

## Keyboard controls

| Key | Action |
|-----|--------|
| Arrow Left / Arrow Down | Move slider 5% to the left |
| Arrow Right / Arrow Up | Move slider 5% to the right |
| Home | Move slider to 0% (full "after") |
| End | Move slider to 100% (full "before") |

The slider handle has `role="slider"` with `aria-valuenow`, `aria-valuemin`, and `aria-valuemax` attributes for screen readers.

## Options

| Option | Syntax | Default | Description |
|--------|--------|---------|-------------|
| Label | `:::image-compare[Label text]` | — | Title displayed above the comparison. |

## Edge cases

- Exactly two images are required. With fewer than two, an error message renders: "Image compare requires two images (before and after)."
- Extra images beyond the first two are ignored.
- Both images should have the same aspect ratio for the best visual result. Mismatched sizes still work but may produce uneven edges.
- Multiple image-compare blocks on the same page each have their own independent slider instance.
- The images have `draggable="false"` to prevent browser drag behavior from interfering with the slider.
