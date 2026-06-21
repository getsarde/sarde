---
title: "Gallery"
description: "Responsive image grids from standard Markdown images."
sidebar:
  order: 22
---

Galleries arrange multiple images in a responsive grid layout. Each image inside the block becomes a grid cell.

## Basic Usage

```md
:::gallery
![A forest path](forest.jpg)
![Mountain lake](lake.jpg)
![Desert dunes](desert.jpg)
:::
```

:::gallery
![A forest path](forest.jpg)
![Mountain lake](lake.jpg)
![Desert dunes](desert.jpg)
:::

## Optional Title

Add a title in square brackets after `gallery`:

```md
:::gallery[My Photos]
![Cityscape at dusk](city.jpg)
![Coastal cliffs](cliffs.jpg)
![Autumn leaves](leaves.jpg)
![Snowy peak](peak.jpg)
:::
```

:::gallery[My Photos]
![Cityscape at dusk](city.jpg)
![Coastal cliffs](cliffs.jpg)
![Autumn leaves](leaves.jpg)
![Snowy peak](peak.jpg)
:::

The title appears above the grid. Images render in a responsive multi-column layout that collapses to a single column on small screens.
