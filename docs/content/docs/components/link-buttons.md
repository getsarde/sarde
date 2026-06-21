---
title: "Link Buttons"
description: "Styled button links for calls to action."
sidebar:
  order: 8
---

Link buttons render as styled `<a>` elements with button appearance. Three variants control the visual weight.

## Basic Usage

```
:::link-button[Get Started]{href="/docs/" variant="primary"}
:::
```

:::link-button[Get Started]{href="/docs/" variant="primary"}
:::

The label goes in square brackets. The block is self-closing — no body content required.

## Variants

Three variants are available: `primary`, `secondary`, and `minimal`.

```
:::link-button[Get Started]{href="/docs/" variant="primary"}
:::

:::link-button[View Source]{href="https://github.com/" variant="secondary"}
:::

:::link-button[Learn more]{href="/about/" variant="minimal"}
:::
```

:::link-button[Get Started]{href="/docs/" variant="primary"}
:::

:::link-button[View Source]{href="https://github.com/" variant="secondary"}
:::

:::link-button[Learn more]{href="/about/" variant="minimal"}
:::

The default variant is `primary` when `variant` is omitted.

## With Icon

Add an icon by name (Lucide icon set) using the `icon` attribute.

```
:::link-button[Download]{href="/download/" variant="primary" icon="arrow-down"}
:::
```

:::link-button[Download]{href="/download/" variant="primary" icon="arrow-down"}
:::

Control icon placement with `iconPlacement`. Accepted values: `start` and `end` (default).

```
:::link-button[Read the docs]{href="/docs/" variant="secondary" icon="arrow-right" iconPlacement="start"}
:::
```

:::link-button[Read the docs]{href="/docs/" variant="secondary" icon="arrow-right" iconPlacement="start"}
:::

## Body Text

Place text in the body instead of the label brackets.

```
:::link-button{href="/changelog/" variant="secondary"}
See what's changed
:::
```

:::link-button{href="/changelog/" variant="secondary"}
See what's changed
:::

## Attributes

| Attribute | Type | Default | Description |
|-----------|------|---------|-------------|
| `href` | string | — | **Required.** The link destination URL. |
| `variant` | string | `primary` | Visual style: `primary`, `secondary`, or `minimal`. |
| `icon` | string | — | Lucide icon name to display alongside the label. |
| `iconPlacement` | string | `end` | Icon position relative to label: `start` or `end`. |
