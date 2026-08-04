---
title: Badges
description: "Display short colored pill labels with optional icons for status, versions, or categories"
sidebar:
  order: 12
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

| Type | Example | Color | Default icon | Use for |
|------|---------|-------|-------------|---------|
| `default` | <span class="sarde-badge sarde-badge-default">:icon[info]Default</span> | Gray | `info` | Neutral labels. |
| `primary` | <span class="sarde-badge sarde-badge-primary">:icon[circle-check]Primary</span> | Accent | `circle-check` | Primary actions or status. |
| `secondary` | <span class="sarde-badge sarde-badge-secondary">:icon[circle-minus]Secondary</span> | Muted | `circle-minus` | De-emphasized labels. |
| `success` | <span class="sarde-badge sarde-badge-success">:icon[circle-check]Success</span> | Green | `circle-check` | Completed, passed, active. |
| `warning` | <span class="sarde-badge sarde-badge-warning">:icon[triangle-alert]Warning</span> | Amber | `triangle-alert` | Needs attention. |
| `caution` | <span class="sarde-badge sarde-badge-caution">:icon[triangle-alert]Caution</span> | Amber | `triangle-alert` | Proceed carefully. |
| `danger` | <span class="sarde-badge sarde-badge-danger">:icon[circle-x]Danger</span> | Red | `circle-x` | Failed, removed, breaking. |
| `info` | <span class="sarde-badge sarde-badge-info">:icon[info]Info</span> | Blue | `info` | Informational labels. |
| `note` | <span class="sarde-badge sarde-badge-note">:icon[pencil]Note</span> | Blue | `pencil` | Editorial notes. |
| `tip` | <span class="sarde-badge sarde-badge-tip">:icon[sparkles]Tip</span> | Green | `sparkles` | Recommendations. |

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
