---
title: Card Grid
description: "Arrange multiple cards in a responsive multi-column grid layout"
sidebar:
  order: 14
---

Card grids arrange multiple cards in a responsive multi-column layout. Wrap cards in `:::card-grid` to display them side by side.

## Basic syntax

````
::::card-grid(cols=3)
:::card[Biology]
Study of living organisms and their interactions.
:::
:::card[Chemistry]
Study of matter, its properties, and reactions.
:::
:::card[Physics]
Study of energy, motion, and fundamental forces.
:::
::::
````

::::card-grid(cols=3)
:::card[Biology]
Study of living organisms and their interactions.
:::
:::card[Chemistry]
Study of matter, its properties, and reactions.
:::
:::card[Physics]
Study of energy, motion, and fundamental forces.
:::
::::

→ Three cards appear side by side in a three-column grid. On smaller screens, the grid collapses to fewer columns.

## Column count

Set the number of columns with the `cols` attribute. Valid values are 2, 3, or 4:

````
::::card-grid(cols=2)
:::card[Frontend]
HTML, CSS, and JavaScript.
:::
:::card[Backend]
Server-side logic and databases.
:::
::::
````

::::card-grid(cols=2)
:::card[Frontend]
HTML, CSS, and JavaScript.
:::
:::card[Backend]
Server-side logic and databases.
:::
::::

→ Two cards appear side by side in a two-column grid.

## Staggered layout

Add the `stagger` attribute to offset even-numbered cards vertically. This forces a two-column layout regardless of any `cols` value:

````
::::card-grid(stagger)
:::card[Week 1]
Introduction and orientation.
:::
:::card[Week 2]
Core concepts and first assignment.
:::
:::card[Week 3]
Lab work and group projects.
:::
:::card[Week 4]
Review and final assessment.
:::
::::
````

→ Cards appear in two columns with alternating vertical offsets.

## Options

| Option | Syntax | Default | Description |
|--------|--------|---------|-------------|
| `cols` | `cols=2`, `cols=3`, `cols=4` | (auto) | Number of columns. Valid values: 2, 3, or 4. |
| `stagger` | `stagger` | `false` | Offset even-numbered cards vertically for a staggered look. Forces 2 columns. |

## Edge cases

- The `cols` value is clamped to 2-4. Values outside this range are ignored.
- When `stagger` is set, the column count is forced to 2 regardless of any `cols` value.
- A card grid can contain any card variant (default, highlighted, subtle).
