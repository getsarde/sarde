---
title: Link Button Group
description: "Wrap multiple link buttons in a horizontal row with consistent spacing"
sidebar:
  order: 30
  group: Block
---

Link Button Group wraps multiple [link buttons](/extensions/link-buttons) in a horizontal row with consistent spacing.

## Basic syntax

````
::::link-button-group
:::link-button[Get Started](href="/docs/getting-started/" variant="primary")
:::
:::link-button[View Demo](href="/demo/" variant="outline")
:::
::::
````

→ Two buttons appear side by side.

Using 4 colons for the outer group and 3 for each nested button is a style convention for readability. Both fences accept 3 or more colons.

## Options

Link Button Group takes no attributes of its own. Style and content are controlled by each nested `:::link-button` block. See [Link Buttons](/extensions/link-buttons) for link button options.

## Edge cases

- Link Button Group is a layout wrapper only. It accepts any block content, though it is intended for `:::link-button` children.
- A link button with no `href` is silently dropped, per the [Link Buttons](/extensions/link-buttons#edge-cases) edge cases. The group still renders around whatever buttons remain.
