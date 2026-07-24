---
title: Link Card
description: "Render a clickable card-style link with a title, description, and optional icon or image"
sidebar:
  order: 20
---

Link cards render a clickable card-style link with a title, optional description, and optional icon or image. External links automatically show the domain name.

## Basic syntax

````
:::link-card[Sarde Documentation](href="/docs/")
:::
````

→ A bordered card appears with "Sarde Documentation" as the title and a right-arrow indicator.

## With description

Add a description using the `description` attribute or body text:

````
:::link-card[Getting Started](href="/docs/getting-started/" description="Install Sarde and create a site in under five minutes.")
:::
````

→ The card shows the title on top and the description below it in muted text.

Body text between the fences also works as the description:

````
:::link-card[Getting Started](href="/docs/getting-started/")
Install Sarde and create a site in under five minutes.
:::
````

If both an attribute and body text are present, the body text overrides the attribute.

## Icon

Add a Lucide icon to the card with the `icon` attribute:

````
:::link-card[API Reference](href="/docs/reference/" icon="book-open")
:::
````

→ A book-open icon appears to the left of the card title.

## Image

Add a thumbnail image with the `image` attribute:

````
:::link-card[Course Overview](href="/courses/intro/" image="/images/course-thumb.jpg")
:::
````

→ A thumbnail image appears on the left side of the card. When both `icon` and `image` are set, the image takes precedence.

## External links

External URLs automatically open in a new tab. The domain name appears at the bottom of the card:

````
:::link-card[GitHub Repository](href="https://github.com/getsarde/sarde")
:::
````

→ The card shows "GitHub Repository" as the title and "github.com" as the domain label. The `www.` prefix is stripped from the domain display.

Override the new-tab behavior with `new-tab="false"`:

````
:::link-card[Partner Site](href="https://example.com" new-tab="false")
:::
````

## Auto title

When no title is provided in square brackets, the domain name from the URL is used as the title:

````
:::link-card(href="https://github.com/getsarde/sarde")
:::
````

→ The card title displays "github.com".

## Options

| Option | Syntax | Default | Description |
|--------|--------|---------|-------------|
| Title | `[Title text]` or `title="..."` | Domain from URL | Card heading. |
| `href` | `href="/path/"` | — | Link destination. Required. |
| `description` | `description="..."` or body text | — | Text below the title. |
| `icon` | `icon="name"` | — | Lucide icon displayed beside the title. |
| `image` | `image="/path/to/img.jpg"` | — | Thumbnail image on the left side. |
| `new-tab` | `new-tab="true"` or `"false"` | Auto (external) | Override new-tab behavior. |

## Edge cases

- A link card without `href` is silently ignored.
- Links with blocked schemes (`javascript:`, `data:`, `vbscript:`) are suppressed.
- When `image` is set, the `icon` attribute is ignored.
- Multiline body text is joined with spaces into a single description.
- Internal links (no `http://` or `https://` prefix) do not show a domain label and do not open in a new tab by default.
