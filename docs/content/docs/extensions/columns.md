---
title: Columns
description: "Split content into side-by-side columns with a configurable column count"
sidebar:
  order: 28
  group: Block
---

Columns split content into a side-by-side grid. Use it to place related blocks, such as a code sample and its explanation, next to each other.

## Basic syntax

````
:::columns
:::column
Left column content.
:::
:::column
Right column content.
:::
:::
````

:::columns
:::column
Left column content.
:::
:::column
Right column content.
:::
:::

→ Two columns of equal width appear side by side.

## Column count

Set the number of columns with `cols`. The value must be a quoted string and ranges from 1 to 4. The default is 2.

````
:::columns(cols="3")
:::column
One
:::
:::column
Two
:::
:::column
Three
:::
:::
````

## Rich content

Each `:::column` accepts any Markdown, including headings, lists, code blocks, and other block extensions.

````
:::columns
:::column
## Option A

- Fast setup
- Fewer dependencies
:::
:::column
## Option B

- More flexible
- Steeper learning curve
:::
:::
````

## Options

| Option | Syntax | Default | Description |
|--------|--------|---------|-------------|
| `cols` | `cols="N"` | `"2"` | Number of columns. Range: 1-4. |

## Edge cases

- The `cols` value is clamped to the 1-4 range. Values outside that range fall back to the default of 2.
- The value must be quoted (`cols="3"`). An unquoted value (`cols=3`) is not recognized and the default is used instead.
- On small screens, columns stack vertically.
- A bare `:::` closes the nearest open block, whether that is a `:::column` or the outer `:::columns` container.
