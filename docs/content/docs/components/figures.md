---
title: "Figures"
description: "Images with captions using semantic HTML."
sidebar:
  order: 21
---

Figures wrap an image in a `<figure>` element with a `<figcaption>`. Use them when an image needs a descriptive caption that is part of the document's content — not just alt text for accessibility.

## Basic Usage

Place a standard Markdown image inside a `:::figure` block. The caption text in square brackets is required:

```md
:::figure[The Sarde build pipeline — six phases from source files to output HTML.]
![Sarde build pipeline diagram](./pipeline.png)
:::
```

:::figure[The Sarde build pipeline — six phases from source files to output HTML.]
![Sarde build pipeline diagram](https://via.placeholder.com/720x300)
:::

The bracket text becomes the visible `<figcaption>` below the image. The alt attribute on the image is separate and should still describe the image for screen readers.

## Attributes

| Attribute | Type   | Required | Description                                  |
|-----------|--------|----------|----------------------------------------------|
| `[text]`  | string | Yes      | Caption text rendered as `<figcaption>`      |
