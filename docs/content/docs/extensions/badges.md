---
title: Badges
description: "Display short colored pill labels with optional icons for status, versions, or categories"
sidebar:
  order: 12
  group: Block
---

Badges display short labels with colored backgrounds and optional icons. Use them for status indicators, version labels, or category tags.

## Basic syntax

````
:::badge(success)
Completed
:::
````

:::badge(success)
Completed
:::

→ A green pill-shaped label appears with a checkmark icon and the text "Completed".

## Badge types

| Type | Color | Default icon | Use for |
|------|-------|-------------|---------|
| `default` | Gray | `info` | Neutral labels. |
| `primary` | Accent | `circle-check` | Primary actions or status. |
| `secondary` | Muted | `circle-minus` | De-emphasized labels. |
| `success` | Green | `circle-check` | Completed, passed, active. |
| `warning` | Amber | `triangle-alert` | Needs attention. |
| `caution` | Amber | `triangle-alert` | Proceed carefully. |
| `danger` | Red | `circle-x` | Failed, removed, breaking. |
| `info` | Blue | `info` | Informational labels. |
| `note` | Blue | `pencil` | Editorial notes. |
| `tip` | Green | `sparkles` | Recommendations. |

Set the type as the first parameter:

````
:::badge(danger)
Deprecated
:::
````

:::badge(danger)
Deprecated
:::

→ A red badge appears with an x-circle icon and the text "Deprecated".

## Custom icon

Override the default icon with the `icon` parameter:

````
:::badge(primary icon="rocket")
Launching
:::
````

:::badge(primary icon="rocket")
Launching
:::

→ The badge displays a rocket icon instead of the default circle-check.

## Outline style

Add `style="outline"` for a bordered variant with a transparent background:

````
:::badge(info style="outline")
v3.2.0
:::
````

:::badge(info style="outline")
v3.2.0
:::

→ A bordered blue badge appears with transparent fill and blue text.

## Size and icon options

Set `size="sm"` or `size="lg"` to adjust the badge size. The default is the standard size between the two.

Add `no-icon="true"` to hide the icon entirely:

````
:::badge(warning no-icon="true")
Beta
:::
````

:::badge(warning no-icon="true")
Beta
:::

→ An amber badge appears with text only, no icon.

## Badge groups

Wrap multiple badges in a `:::badge-group` to display them in a horizontal row:

````
:::badge-group
:::badge(success)
Passed
:::
:::badge(danger)
Failed
:::
:::badge(info)
Skipped
:::
:::
````

:::badge-group
:::badge(success)
Passed
:::
:::badge(danger)
Failed
:::
:::badge(info)
Skipped
:::
:::

→ Three badges appear in a horizontal row with consistent spacing.

## Options

| Option | Syntax | Default | Description |
|--------|--------|---------|-------------|
| `type` | `:::badge(type)` or `type="value"` | `"default"` | Badge color variant. |
| `icon` | `icon="name"` | Per-type default | Lucide icon name. |
| `style` | `style="outline"` | (solid) | `"outline"` for bordered variant. |
| `size` | `size="sm"` or `size="lg"` | (default) | Badge size. |
| `no-icon` | `no-icon="true"` | `false` | Hide the icon. |

## Edge cases

- The type can be passed as a bare word (`:::badge(success)`) or as a key-value pair (`:::badge(type="success")`). Both are equivalent.
- An unrecognized type renders with the `default` styling (gray).
- When both a custom `icon` and `no-icon="true"` are set, the icon is hidden.
- Badge content that spans multiple lines is joined with spaces into a single line.
