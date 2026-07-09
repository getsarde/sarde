---
title: Cards
sidebar:
  order: 13
  group: Block
---

Cards display content in bordered, elevated containers with optional titles and icons. Use `:::card-grid` to arrange multiple cards in a responsive column layout.

## Basic syntax

````
:::card[Lesson Overview]
This module covers the fundamentals of plant biology, including cell structure, photosynthesis, and reproduction.
:::
````

:::card[Lesson Overview]
This module covers the fundamentals of plant biology, including cell structure, photosynthesis, and reproduction.
:::

→ A bordered container appears with "Lesson Overview" as the header and the paragraph below it.

## Card with icon

Add an icon to the card header with the `icon` attribute:

````
:::card[Getting Started](icon="rocket")
Install the CLI, create a project, and run the dev server.
:::
````

:::card[Getting Started](icon="rocket")
Install the CLI, create a project, and run the dev server.
:::

→ A rocket icon appears beside the title in the card header.

## Card variants

Set the `variant` attribute to change the card's visual style:

````
:::card[Important Update](variant="highlighted")
The exam schedule has changed. Check the calendar for new dates.
:::
````

:::card[Important Update](variant="highlighted")
The exam schedule has changed. Check the calendar for new dates.
:::

→ The card appears with an accent-colored border or background, making it stand out.

| Variant | Description |
|---------|-------------|
| (default) | Standard bordered card. |
| `highlighted` | Accent-colored emphasis. |
| `subtle` | Reduced visual weight. |

## Card grid

Wrap cards in `:::card-grid` for a responsive multi-column layout:

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

## Card grid options

| Option | Syntax | Default | Description |
|--------|--------|---------|-------------|
| `cols` | `cols=2`, `cols=3`, `cols=4` | (auto) | Number of columns. Valid values: 2, 3, or 4. |
| `stagger` | `stagger` | `false` | Offset even-numbered cards vertically for a staggered look. Forces 2 columns. |

````
:::card-grid(stagger)
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
:::
````

→ Cards appear in two columns with alternating vertical offsets.

## Options

| Option | Syntax | Default | Description |
|--------|--------|---------|-------------|
| Title | `:::card[Title]` or `title="Title"` | — | Card header text. |
| `icon` | `icon="name"` | — | Lucide icon name displayed in the header. |
| `variant` | `variant="highlighted"` or `variant="subtle"` | (default) | Visual style variant. |

## Edge cases

- A card with no title and no icon renders without a header section. Only the body content appears.
- The `cols` value is clamped to 2-4. Values outside this range are ignored.
- When `stagger` is set, the column count is forced to 2 regardless of any `cols` value.
- Cards can contain any Markdown content, including code blocks, lists, images, and nested extensions.
