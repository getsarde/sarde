---
title: CSS Layout
description: Lay out pages with the box model, flexbox, and grid.
sidebar:
  order: 4
tags: [css]
---

This lesson covers CSS layout techniques. Replace this with your own content.

:::caution
Setting `width: 100%` together with padding and a border can overflow the parent. Use `box-sizing: border-box` to include them in the width.
:::

## Flexbox or grid

Both versions lay out `.card` elements in rows at least `200px` wide that wrap on narrow screens. Switch between the tabs to compare them:

:::code-group

```css title="Flexbox"
.cards {
  display: flex;
  flex-wrap: wrap;
  gap: 1rem;
}

.card {
  flex: 1 1 200px;
}
```

```css title="Grid"
.cards {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 1rem;
}
```

:::

## Topics

- The box model
- Flexbox
- Grid
