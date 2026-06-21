---
title: "File Tree"
description: "Display directory structures with file icons."
sidebar:
  order: 10
---

File trees render directory structures from indented list syntax. Directories end with `/`; files do not.

## Basic Usage

```
:::file-tree
- src/
  - index.go
  - config.go
- README.md
:::
```

:::file-tree
- src/
  - index.go
  - config.go
- README.md
:::

Indent child items with two spaces per level. Any list item ending in `/` is treated as a directory.

## Realistic Example

```
:::file-tree
- my-site/
  - content/
    - docs/
      - _index.md
      - getting-started.md
    - blog/
      - _index.md
      - hello-world.md
    - _index.md
  - static/
    - images/
      - logo.svg
  - sarde.yaml
:::
```

:::file-tree
- my-site/
  - content/
    - docs/
      - _index.md
      - getting-started.md
    - blog/
      - _index.md
      - hello-world.md
    - _index.md
  - static/
    - images/
      - logo.svg
  - sarde.yaml
:::

## Highlighted Entries

Wrap an entry in `**double asterisks**` to highlight it.

```
:::file-tree
- my-site/
  - content/
  - static/
  - **sarde.yaml**
:::
```

:::file-tree
- my-site/
  - content/
  - static/
  - **sarde.yaml**
:::

Use highlighting to draw attention to the file being discussed in surrounding prose.
