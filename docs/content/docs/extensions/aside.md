---
title: Aside
sidebar:
  order: 11
  group: Block
---

Asides add colored callout blocks (notes, tips, cautions, and dangers) that stand out from the surrounding content. Each type has a distinct color and icon.

## Basic syntax

````
:::note
Photosynthesis requires both water and carbon dioxide.
:::
````

:::note
Photosynthesis requires both water and carbon dioxide.
:::

→ A blue callout box appears with a "Note" label and a book-open icon.

## Aside types

Set the type as the directive name:

| Type | Color | Icon | Use for |
|------|-------|------|---------|
| `note` | Blue | `book-open` | Supplementary information the reader might find helpful. |
| `tip` | Green | `sparkles` | Helpful suggestions or best practices. |
| `info` | Blue | `info` | Neutral informational context. |
| `warning` | Amber | `flame` | Potential issues that could cause problems. |
| `caution` | Amber | `triangle-alert` | Proceed carefully, risk of mistakes. |
| `important` | Purple | `flag` | Key information the reader must not miss. |
| `danger` | Red | `x-circle` | Breaking changes, data loss, or destructive actions. |

````
:::tip
Use descriptive variable names to make code self-documenting.
:::
````

:::tip
Use descriptive variable names to make code self-documenting.
:::

→ A green callout box appears with a "Tip" label and a sparkles icon.

````
:::danger
Dropping a database table cannot be undone. Back up your data first.
:::
````

:::danger
Dropping a database table cannot be undone. Back up your data first.
:::

→ A red callout box appears with a "Danger" label and an x-circle icon.

````
:::warning
Changing the URL slug breaks existing bookmarks and external links.
:::
````

:::warning
Changing the URL slug breaks existing bookmarks and external links.
:::

→ An amber callout box appears with a "Warning" label and a flame icon.

## Custom title

Override the default title with square brackets after the type name:

````
:::note[Before you begin]
Make sure Node.js 18 or later is installed.
:::
````

:::note[Before you begin]
Make sure Node.js 18 or later is installed.
:::

→ The aside displays "Before you begin" instead of the default "Note" title.

## Custom icon

Override the default icon with the `icon` parameter. The value is a Lucide icon name:

````
:::tip[Performance] icon=zap
Enable parallel builds for faster compilation on multi-core machines.
:::
````

:::tip[Performance] icon=zap
Enable parallel builds for faster compilation on multi-core machines.
:::

→ The tip displays a zap (lightning) icon instead of the default sparkles icon.

## GitHub-style variants

Sarde also supports GitHub-style alert types for compatibility with content originally authored on GitHub:

| Type | Equivalent | Default title |
|------|-----------|---------------|
| `gh-note` | `note` | Note |
| `gh-tip` | `tip` | Tip |
| `gh-important` | `important` | Important |
| `gh-warning` | `warning` | Warning |
| `gh-caution` | `caution` | Caution |

GitHub-style variants receive an additional `sarde-aside-github` CSS class for separate styling if needed. They use the same icons as their standard counterparts.

## Nesting content

Asides can contain any Markdown content: paragraphs, lists, code blocks, images, and even other extensions. Use four colons for the outer fence when nesting:

````
::::note[Course prerequisites]
Complete these steps before starting:

1. Install the development tools
2. Clone the repository
3. Run the setup script

:::tip
The setup script handles dependency installation automatically.
:::
::::
````

## Edge cases

- An unrecognized type name (e.g., `:::custom`) is silently ignored. The content renders as a plain paragraph.
- Asides cannot interrupt a paragraph. A blank line must precede the opening `:::`.
- A named closing fence (`:::/note`) must match the opening type. A mismatched name (e.g., opening with `:::note` and closing with `:::/tip`) does not close the block.
- When no icon is found for a type (after checking the explicit `icon` parameter and the built-in icon map), the fallback is the `info` icon.
- The default title for an unrecognized GitHub-style variant falls back to "Note".
