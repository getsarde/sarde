---
title: Tabs
description: "Switch between related content panels, with tab selection synced across the site"
sidebar:
  order: 24
---

Tabs let readers switch between related content panels without leaving the page. Tab selections persist in localStorage and sync across all tab groups on the site that share the same label.

## Basic syntax

Use `== Label` lines to define tab boundaries inside a `:::tabs` block:

````
:::tabs
== npm

```bash
npm install sarde
```

== yarn

```bash
yarn add sarde
```

== pnpm

```bash
pnpm add sarde
```
:::
````

→ Three tabs appear ("npm", "yarn", "pnpm"). Clicking a tab shows its panel and hides the others. The first tab is active by default.

<!-- SCREENSHOT: tabs-basic — three package manager tabs with npm selected -->

## Tab icons

Add an icon to a tab label with `(icon="name")` after the label text:

````
:::tabs
== Biology (icon="leaf")

The study of living organisms.

== Chemistry (icon="flask-conical")

The study of matter and its reactions.
:::
````

→ Each tab button shows a Lucide icon beside its label.

## Tab content

Each tab panel supports any Markdown content: paragraphs, code blocks, lists, images, and nested extensions. Use four colons for the outer fence when nesting block extensions inside tabs:

````
::::tabs
== Overview

:::note
This course requires prior knowledge of basic algebra.
:::

== Syllabus

1. Introduction to functions
2. Limits and continuity
3. Derivatives
::::
````

## localStorage sync

When a reader clicks a tab, the selected label is saved to `localStorage` under the `sarde-tabs` key. On page load, any tab group containing a previously selected label activates that tab.

This means selecting "yarn" in one tab group activates "yarn" in every other tab group on the site that has a "yarn" tab. This is useful for package manager or language preference tabs where the reader's choice applies globally.

The sync matches on the exact label text. Two tab groups sync when they share at least one label.

## Keyboard navigation

Tab buttons use `role="tablist"` and `role="tab"` ARIA attributes. Each panel has `role="tabpanel"` with `aria-labelledby` pointing to its tab button.

Non-active panels have the `hidden` attribute and are removed from the tab order.

## Options

| Feature | Syntax | Description |
|---------|--------|-------------|
| Tab label | `== Label text` | Starts a new tab. The text after `==` becomes the tab button label. |
| Tab icon | `== Label (icon="name")` | Lucide icon displayed in the tab button. |

## Edge cases

- Content before the first `== Label` line creates an implicit tab labeled "Tab 1".
- Tab labels are case-sensitive for localStorage sync. "npm" and "NPM" are treated as different tabs.
- Multiple tab groups on the same page each have independent IDs. ARIA references (`aria-controls`, `aria-labelledby`) are scoped per group.
- With JavaScript disabled, only the first tab panel is visible. The others remain hidden.
- Empty tab panels (a `== Label` line with no content before the next label or closing fence) render as empty panels.
