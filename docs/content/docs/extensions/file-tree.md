---
title: File Tree
description: "Render directory structures as a styled tree with folder and file icons"
sidebar:
  order: 16
---

File trees render directory structures with folder/file icons, indentation lines, and optional highlights and comments. Write the tree as a standard Markdown list inside a `:::file-tree` block.

## Basic syntax

````
:::file-tree
- content/
  - docs/
    - getting-started.md
    - guides/
      - auth.md
  - blog/
    - first-post.md
- sarde.yaml
- package.json
:::
````

:::file-tree
- content/
  - docs/
    - getting-started.md
    - guides/
      - auth.md
  - blog/
    - first-post.md
- sarde.yaml
- package.json
:::

→ A styled directory tree appears with folder icons for directories and file icons for files. Indentation lines connect parent and child entries.

<!-- SCREENSHOT: file-tree-basic — a directory tree with folders and files -->

## Folders vs files

Entries ending with `/` are treated as folders and receive a folder icon. Entries with nested children are also treated as folders regardless of the trailing slash. All other entries are treated as files and receive a file icon with a CSS class based on the file extension (e.g., `file-md`, `file-yaml`).

## Highlighting entries

Bold an entry with `**` to highlight it:

````
:::file-tree
- content/
  - docs/
    - **getting-started.md**
    - guides/
- sarde.yaml
:::
````

:::file-tree
- content/
  - docs/
    - **getting-started.md**
    - guides/
- sarde.yaml
:::

→ The "getting-started.md" entry appears with an accent-colored highlight, drawing attention to it.

## Comments

Add a comment after ` #` (space then hash) to display muted annotation text beside an entry:

````
:::file-tree
- content/ #required
  - docs/
    - _index.md #collection root
    - guides/
      - auth.md #new page
- sarde.yaml #site configuration
:::
````

:::file-tree
- content/ #required
  - docs/
    - _index.md #collection root
    - guides/
      - auth.md #new page
- sarde.yaml #site configuration
:::

→ Each annotated entry shows its comment in muted text to the right of the filename.

## Placeholder entries

Use `...` as an entry to indicate omitted content:

````
:::file-tree
- content/
  - docs/
    - ...
  - blog/
    - ...
- sarde.yaml
:::
````

→ Ellipsis entries appear without icons, indicating that additional files exist but are not shown.

## Custom icons

Override the default file or folder icon with the `icon:` prefix:

````
:::file-tree
- icon:database data/
  - icon:file-json users.json
  - icon:file-json products.json
- icon:settings config.yaml
:::
````

→ Each entry displays the specified Lucide icon instead of the default folder or file icon.

The syntax is `icon:<name> <filename>`. The icon name and filename are separated by a space.

## Options

| Feature | Syntax | Description |
|---------|--------|-------------|
| Folder | Trailing `/` or nested children | Renders with folder icon. |
| File | No trailing `/` and no children | Renders with file icon. |
| Highlight | `**name**` | Accent-colored highlight on the entry. |
| Comment | `name #comment text` | Muted annotation text beside the entry. |
| Placeholder | `...` | Indicates omitted content. |
| Custom icon | `icon:name filename` | Overrides the default icon. |

## Edge cases

- Nesting is determined by standard Markdown list indentation (2 or 4 spaces per level).
- The `#` comment marker requires a preceding space. A filename like `#readme` is not treated as a comment.
- File extension CSS classes are alphanumeric only. Extensions with special characters (e.g., `.tar.gz`) use the last segment (`gz`).
- The closing fence accepts both `:::/file-tree` and `:::/filetree`.
