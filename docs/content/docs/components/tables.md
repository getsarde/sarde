---
title: "Tables"
description: "Responsive tables with GitHub-Flavored Markdown syntax."
sidebar:
  order: 12
---

Tables use standard GitHub-Flavored Markdown (GFM) pipe syntax. Sarde wraps them in a responsive container so they scroll horizontally on narrow screens.

## Basic Usage

```
| Name | Type | Description |
|------|------|-------------|
| title | string | The page title shown in the browser tab and `<h1>`. |
| date | string | Publication date in `YYYY-MM-DD` format. |
| draft | boolean | When `true`, the page is excluded from the build. |
```

The separator row (`|---|`) is required. Column widths adjust to content.

## Alignment

Control column alignment with colons in the separator row.

```
| Left | Center | Right |
|:-----|:------:|------:|
| text | text   | text  |
| text | text   | text  |
```

- `:---` — left-align (default)
- `:---:` — center-align
- `---:` — right-align

## Practical Example

Frontmatter fields for a docs page:

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `title` | string | filename | Page title. |
| `description` | string | — | Used in meta tags and search results. |
| `sidebar.order` | number | `0` | Sort position within the sidebar section. |
| `sidebar.label` | string | title | Override the label shown in the sidebar. |
| `draft` | boolean | `false` | Exclude the page from the build output. |
| `date` | string | git date | Publication or last-modified date. |
