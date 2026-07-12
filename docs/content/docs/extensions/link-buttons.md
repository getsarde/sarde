---
title: Link Buttons
description: "Render styled anchor links that look like buttons, with multiple color variants"
sidebar:
  order: 19
  group: Block
---

Link buttons render styled anchor elements that look like buttons. Each button requires an `href` attribute.

## Basic syntax

````
:::link-button[Get Started](href="/docs/getting-started/")
:::
````

→ A primary-colored button appears with the text "Get Started" linking to the getting-started page.

## Variants

Set the `variant` attribute to change the button style:

````
:::link-button[Primary](href="/docs/" variant="primary")
:::
:::link-button[Secondary](href="/docs/" variant="secondary")
:::
:::link-button[Outline](href="/docs/" variant="outline")
:::
:::link-button[Ghost](href="/docs/" variant="ghost")
:::
:::link-button[Minimal](href="/docs/" variant="minimal")
:::
````

| Variant | Description |
|---------|-------------|
| `primary` | Solid accent-colored background. Default. |
| `secondary` | Muted background color. |
| `outline` | Transparent with a border. |
| `ghost` | Transparent with no border. Hover reveals background. |
| `minimal` | Text-only, no background or border. |

## Icon

Add an icon with the `icon` attribute. Control placement with `iconPlacement`:

````
:::link-button[View on GitHub](href="https://github.com/example" icon="github" iconPlacement="start")
:::
````

→ A GitHub icon appears before the button text.

| Value | Description |
|-------|-------------|
| `end` | Icon after the text. Default. |
| `start` | Icon before the text. |

An icon-only button (no label text) receives a compact square layout:

````
:::link-button(href="https://github.com/example" icon="github")
:::
````

## Size

Set `size` to `sm` or `lg`:

````
:::link-button[Small](href="/docs/" size="sm")
:::
:::link-button[Large](href="/docs/" size="lg")
:::
````

## Layout and state

Add `block="true"` for a full-width button, or `center="true"` to center it. Add `disabled="true"` for a grayed-out state with `aria-disabled="true"`.

External URLs (`http://` or `https://`) automatically open in a new tab. Override with `new-tab="false"`.

## Button groups

Wrap multiple link buttons in `:::link-button-group` for horizontal alignment:

````
::::link-button-group
:::link-button[Get Started](href="/docs/getting-started/" variant="primary")
:::
:::link-button[View Demo](href="/demo/" variant="outline")
:::
::::
````

→ Two buttons appear side by side with consistent spacing.

## Body text as label

When no label is provided in square brackets, body text between the fences becomes the button label:

````
:::link-button(href="/docs/")
Read the documentation
:::
````

If both a bracket label and body text are present, the bracket label takes precedence.

## Options

| Option | Syntax | Default | Description |
|--------|--------|---------|-------------|
| Label | `[Text]` or body content | — | Button text. |
| `href` | `href="/path/"` | — | Link destination. Required. |
| `variant` | `variant="secondary"` | `"primary"` | Visual style. |
| `icon` | `icon="name"` | — | Lucide icon name. |
| `iconPlacement` | `iconPlacement="start"` | `"end"` | Icon position relative to text. |
| `size` | `size="sm"` or `size="lg"` | (default) | Button size. |
| `block` | `block="true"` | `false` | Full-width button. |
| `center` | `center="true"` | `false` | Center the button horizontally. |
| `disabled` | `disabled="true"` | `false` | Disabled appearance and `aria-disabled`. |
| `new-tab` | `new-tab="true"` or `"false"` | Auto (external) | Override new-tab behavior. |

## Edge cases

- A link button without `href` is silently ignored and produces no output.
- Links with blocked schemes (`javascript:`, `data:`, `vbscript:`) are suppressed.
- An unrecognized `variant` value falls back to `"primary"`.
- An unrecognized `size` value is ignored (default size is used).
