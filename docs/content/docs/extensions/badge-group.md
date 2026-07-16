---
title: Badge Group
description: "Wrap multiple badges in a horizontal row with consistent spacing"
sidebar:
  order: 29
  group: Block
---

Badge Group wraps multiple [badges](/extensions/badges) in a horizontal row with consistent spacing.

## Basic syntax

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

→ Three badges appear in a horizontal row.

## Options

Badge Group takes no attributes of its own. Style and content are controlled by each nested `:::badge` block. See [Badges](/extensions/badges) for badge options.

## Edge cases

- Badge Group is a layout wrapper only. It accepts any block content, though it is intended for `:::badge` children.
- Spacing and alignment come from the group; individual badges do not need extra configuration to line up.
