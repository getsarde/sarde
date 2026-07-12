---
title: Accordion
description: "Group collapsible detail sections so opening one closes the others"
sidebar:
  order: 10
  group: Block
---

Accordions group multiple collapsible sections so that opening one closes the others. Wrap `:::details` blocks inside an `:::accordion` container.

## Basic syntax

````
:::accordion
:::details[What is photosynthesis?]
Photosynthesis is the process by which plants convert light energy into chemical energy stored in glucose.
:::
:::details[Why is chlorophyll green?]
Chlorophyll absorbs red and blue light wavelengths and reflects green, giving plants their color.
:::
:::details[What is cellular respiration?]
Cellular respiration breaks down glucose to release energy in the form of ATP.
:::
:::
````

:::accordion
:::details[What is photosynthesis?]
Photosynthesis is the process by which plants convert light energy into chemical energy stored in glucose.
:::
:::details[Why is chlorophyll green?]
Chlorophyll absorbs red and blue light wavelengths and reflects green, giving plants their color.
:::
:::details[What is cellular respiration?]
Cellular respiration breaks down glucose to release energy in the form of ATP.
:::
:::

→ Three collapsible panels appear in a vertical group. Opening one panel closes any other open panel in the same accordion.

<!-- SCREENSHOT: accordion-basic — three collapsible panels in an accordion group -->

## Independent mode

By default, opening one panel closes the others. Add the `independent` flag to allow multiple panels to remain open at the same time:

````
:::accordion(independent)
:::details[Prerequisites]
Install Go 1.21 or later and a text editor.
:::
:::details[Optional tools]
A Git client and a terminal multiplexer are helpful but not required.
:::
:::
````

:::accordion(independent)
:::details[Prerequisites]
Install Go 1.21 or later and a text editor.
:::
:::details[Optional tools]
A Git client and a terminal multiplexer are helpful but not required.
:::
:::

→ Both panels can be opened simultaneously. Each panel toggles independently.

## Options

| Option | Syntax | Default | Description |
|--------|--------|---------|-------------|
| `independent` | `:::accordion(independent)` | `false` | Allow multiple panels open at the same time. |

## Pre-opened panels

Combine with the `open` flag on individual `:::details` blocks to have specific panels expanded on page load:

````
:::accordion
:::details[Getting started] open
Start by creating a new project directory.
:::
:::details[Advanced configuration]
Override default settings in `sarde.yaml`.
:::
:::
````

→ The first panel is expanded when the page loads. The second panel is collapsed.

## Edge cases

- An accordion with no `:::details` children renders as an empty container.
- The accordion container itself has no visual decoration. All styling comes from the child details blocks.
- Without the `independent` flag, the mutual-exclusion behavior is handled client-side. With JavaScript disabled, all panels behave independently.
- Accordions cannot be nested inside other accordions.
