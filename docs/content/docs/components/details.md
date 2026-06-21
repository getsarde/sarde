---
title: "Details"
description: "Collapsible sections for supplementary content."
sidebar:
  order: 11
---

Details create collapsible sections using the native HTML `<details>` element. Use them for supplementary content that not every reader needs — prerequisites, troubleshooting tips, or verbose explanations.

## Basic Usage

```
:::details[Click to expand]
This content is hidden until the reader opens the section.
:::
```

:::details[Click to expand]
This content is hidden until the reader opens the section.
:::

The text in square brackets becomes the visible summary.

## Pre-expanded

Add `{open}` to render the section expanded by default.

```
:::details[Advanced configuration]{open}
Content visible immediately, but still collapsible.
:::
```

:::details[Advanced configuration]{open}
Content visible immediately, but still collapsible.
:::

## Rich Content Inside

Details sections accept any Markdown content, including code blocks, lists, and callouts.

````
:::details[Full example config]

```yaml
site:
  title: "My Site"
  url: "https://example.com"

theme:
  name: default
  preset: ocean
```

The `preset` key changes the color palette. Available presets: `ocean`, `forest`, `rose`, `slate`.

:::
````

:::details[Full example config]

```yaml
site:
  title: "My Site"
  url: "https://example.com"

theme:
  name: default
  preset: ocean
```

The `preset` key changes the color palette. Available presets: `ocean`, `forest`, `rose`, `slate`.

:::

````
:::details[Troubleshooting: port already in use]

If `sarde dev` fails with "address already in use", another process is listening on port 4727.

Override the port:

```bash
sarde dev --port 4728
```

Or find and stop the conflicting process:

```bash
lsof -i :4727
```

:::
````

:::details[Troubleshooting: port already in use]

If `sarde dev` fails with "address already in use", another process is listening on port 4727.

Override the port:

```bash
sarde dev --port 4728
```

Or find and stop the conflicting process:

```bash
lsof -i :4727
```

:::

## Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `[Summary]` | string | Yes | The visible summary text shown in square brackets. |
| `{open}` | flag | No | Renders the section expanded by default. |
