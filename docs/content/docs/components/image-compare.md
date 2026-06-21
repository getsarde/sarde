---
title: "Image Compare"
description: "Before-and-after image comparison with a drag slider."
sidebar:
  order: 23
---

Image compare displays two images with a draggable slider to reveal each side. The first image is the "before" state; the second is the "after" state.

## Basic Usage

Place exactly two images inside `:::image-compare`:

```md
:::image-compare
![Before optimization](image-before.png)
![After optimization](image-after.png)
:::
```

:::image-compare
![Before optimization](image-before.png)
![After optimization](image-after.png)
:::

The slider starts at the center. Drag it left or right to reveal either image. The `alt` text of each image becomes its accessible label.

## With Caption

Add a label in square brackets to display a caption below the comparison:

```md
:::image-compare[Homepage redesign comparison]
![Old design](old-homepage.png)
![New design](new-homepage.png)
:::
```

:::note
The block requires exactly two images. More or fewer will produce unexpected output.
:::
