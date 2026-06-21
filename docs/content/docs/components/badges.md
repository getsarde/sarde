---
title: "Badges"
description: "Status indicators and labels in pill form."
sidebar:
  order: 13
---

Badges are small pill-shaped labels. Use them for version labels, status indicators, or category tags inline with prose or headings.

## Basic Usage

```
:::badge{type="default"}
Stable
:::
```

:::badge{type="default"}
Stable
:::

## Variants

Nine type variants are available:

```
:::badge{type="default"}
Default
:::

:::badge{type="primary"}
Primary
:::

:::badge{type="secondary"}
Secondary
:::

:::badge{type="success"}
Released
:::

:::badge{type="warning"}
Beta
:::

:::badge{type="danger"}
Deprecated
:::

:::badge{type="info"}
Info
:::

:::badge{type="note"}
Note
:::

:::badge{type="tip"}
Tip
:::
```

:::badge{type="default"}
Default
:::

:::badge{type="primary"}
Primary
:::

:::badge{type="secondary"}
Secondary
:::

:::badge{type="success"}
Released
:::

:::badge{type="warning"}
Beta
:::

:::badge{type="danger"}
Deprecated
:::

:::badge{type="info"}
Info
:::

:::badge{type="note"}
Note
:::

:::badge{type="tip"}
Tip
:::

## With Icon

Add a Lucide icon name via `icon`.

```
:::badge{type="warning" icon="triangle-alert"}
Beta
:::

:::badge{type="success" icon="circle-check"}
Released
:::
```

:::badge{type="warning" icon="triangle-alert"}
Beta
:::

:::badge{type="success" icon="circle-check"}
Released
:::

## Attributes

| Attribute | Type | Default | Description |
|-----------|------|---------|-------------|
| `type` | string | `default` | Color variant: `default`, `primary`, `secondary`, `success`, `warning`, `danger`, `info`, `note`, `tip`. |
| `icon` | string | — | Lucide icon name displayed before the label. |
