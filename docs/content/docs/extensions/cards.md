---
title: Card
description: "Display content in a bordered, elevated container with an optional title and icon"
sidebar:
  order: 13
---

Cards display content in bordered, elevated containers with optional titles and icons. Use them to visually group related information.

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

## Options

| Option | Syntax | Default | Description |
|--------|--------|---------|-------------|
| Title | `:::card[Title]` or `title="Title"` | — | Card header text. |
| `icon` | `icon="name"` | — | Lucide icon name displayed in the header. |
| `variant` | `variant="highlighted"` or `variant="subtle"` | (default) | Visual style variant. |

## Edge cases

- A card with no title and no icon renders without a header section. Only the body content appears.
- Cards can contain any Markdown content, including code blocks, lists, images, and nested extensions.
