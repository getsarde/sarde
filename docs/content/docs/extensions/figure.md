---
title: Figure
sidebar:
  order: 15
  group: Block
---

Figures wrap images (or other content) in an HTML `<figure>` element with an optional `<figcaption>`. Use them when an image needs a visible caption below it.

## Basic syntax

````
:::figure[Diagram of the water cycle]
![Water cycle](/images/water-cycle.png)
:::
````

→ The image appears inside a `<figure>` element with "Diagram of the water cycle" as the caption below it.

<!-- SCREENSHOT: figure-with-caption — an image wrapped in a figure with a caption -->

## Without a caption

Omit the caption text to render a `<figure>` without a `<figcaption>`:

````
:::figure[]
![Lab equipment](/images/lab-setup.jpg)
:::
````

→ The image renders inside a `<figure>` container with no caption. This is useful when the semantic `<figure>` wrapper is needed for styling but no visible text is appropriate.

## Rich content

Figures can contain any Markdown content, not only images. Use them to caption code blocks, tables, or diagrams:

````
:::figure[Student enrollment by department, 2024]
| Department | Students |
|------------|----------|
| Biology | 142 |
| Chemistry | 98 |
| Physics | 76 |
:::
````

→ A table appears inside a `<figure>` with the caption "Student enrollment by department, 2024" below it.

## Options

| Option | Syntax | Default | Description |
|--------|--------|---------|-------------|
| Caption | `:::figure[Caption text]` | — | Text displayed in the `<figcaption>`. Empty brackets produce no caption. |

## Edge cases

- The caption is required in the syntax (square brackets must be present), but the text inside can be empty.
- The caption text is HTML-escaped. Markdown formatting inside the caption is not rendered.
- Figures can nest other block extensions. Use four colons (`::::`) on the outer fence when nesting.
- Multiple images inside a single figure share the same caption.
