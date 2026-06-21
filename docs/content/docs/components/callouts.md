---
title: "Callouts"
description: "Highlight important information with colored admonition blocks."
sidebar:
  order: 2
---

Callouts draw attention to important information. Sarde supports 7 callout types and GitHub-style alert syntax.

## Basic Usage

Wrap content in a fenced block starting with `:::type`:

```
:::note
Check the config file before running the build.
:::
```

:::note
Check the config file before running the build.
:::

## Types

Sarde provides 7 callout types:

```
:::note
Use this for general information.
:::

:::tip
Use this for helpful suggestions.
:::

:::info
Use this for supplementary context.
:::

:::warning
Use this for potential pitfalls.
:::

:::danger
Use this for destructive or irreversible actions.
:::

:::important
Use this for critical requirements.
:::

:::caution
Use this for actions that require care.
:::
```

:::note
Use this for general information.
:::

:::tip
Use this for helpful suggestions.
:::

:::info
Use this for supplementary context.
:::

:::warning
Use this for potential pitfalls.
:::

:::danger
Use this for destructive or irreversible actions.
:::

:::important
Use this for critical requirements.
:::

:::caution
Use this for actions that require care.
:::

## Custom Title

Override the default title by adding `[Title]` after the type:

```
:::warning[Watch Out]
This will overwrite existing files.
:::
```

:::warning[Watch Out]
This will overwrite existing files.
:::

## Custom Icon

Override the default icon with any Lucide icon name:

```
:::tip icon=rocket
Deploy to production with `sarde build`.
:::
```

:::tip icon=rocket
Deploy to production with `sarde build`.
:::

Combine a custom title and icon:

```
:::note[See Also] icon=book
Read the configuration reference for all available options.
:::
```

## GitHub Alerts

Sarde also renders GitHub-style blockquote alerts:

```
> [!NOTE]
> Check the config file before running the build.

> [!TIP]
> Use `sarde dev` to preview changes as you write.

> [!IMPORTANT]
> Back up your data before migrating.

> [!WARNING]
> This action cannot be undone.

> [!CAUTION]
> Enabling this flag disables all caching.
```

> [!NOTE]
> Check the config file before running the build.

> [!WARNING]
> This action cannot be undone.

## Default Icons

| Type | Icon |
|------|------|
| `note` | `book-open` |
| `tip` | `sparkles` |
| `info` | `info` |
| `warning` | `flame` |
| `danger` | `x-circle` |
| `important` | `flag` |
| `caution` | `triangle-alert` |

## Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| type | string | Yes | One of `note`, `tip`, `info`, `warning`, `danger`, `important`, `caution` |
| `[Title]` | string | No | Custom title, placed in brackets after the type |
| `icon` | string | No | Lucide icon name to replace the default icon |
