---
title: "Tabs"
description: "Organize content into switchable tabbed panels."
sidebar:
  order: 4
---

Tabs organize related content into panels. Wrap content in `:::tabs` and mark each panel boundary with `== Tab Name`.

## Basic Usage

````
:::tabs

== npm

```bash
npm install
```

== pnpm

```bash
pnpm install
```

== yarn

```bash
yarn install
```

:::
````

:::tabs

== npm

```bash
npm install
```

== pnpm

```bash
pnpm install
```

== yarn

```bash
yarn install
```

:::

## Prose Content

Tabs are not limited to code blocks. Any Markdown content works inside a tab panel:

```
:::tabs

== macOS

Open **System Settings** → **Privacy & Security** → **Full Disk Access**.

== Windows

Open **Settings** → **System** → **Storage** → **Advanced storage settings**.

== Linux

Edit `/etc/fstab` and add the mount point for the target directory.

:::
```

:::tabs

== macOS

Open **System Settings** → **Privacy & Security** → **Full Disk Access**.

== Windows

Open **Settings** → **System** → **Storage** → **Advanced storage settings**.

== Linux

Edit `/etc/fstab` and add the mount point for the target directory.

:::

## Synchronized Tabs

Tabs with matching labels stay synchronized across the page. Selecting **pnpm** in one tab group automatically switches all other tab groups with a **pnpm** tab to match.

:::tip
For tabbed code blocks showing the same content in multiple languages, use [Code Groups](/docs/components/code-blocks/#code-groups) instead — they use the same syncing behavior and are designed specifically for that pattern.
:::
