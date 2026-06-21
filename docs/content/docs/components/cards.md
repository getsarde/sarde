---
title: "Cards"
description: "Display content in styled card containers."
sidebar:
  order: 5
---

Cards display content in a bordered container with a subtle hover elevation. Add an optional title and icon to the opening fence.

## Basic Usage

````
:::card[Getting Started]
Install Sarde and scaffold your first project in under a minute.
:::
````

:::card[Getting Started]
Install Sarde and scaffold your first project in under a minute.
:::

## With Icon

Add `{icon="name"}` to display a Lucide icon next to the title:

````
:::card[Configuration]{icon="settings"}
Control your site's title, theme, collections, and plugins from `sarde.yaml`.
:::
````

:::card[Configuration]{icon="settings"}
Control your site's title, theme, collections, and plugins from `sarde.yaml`.
:::

## Without Title

Omit the `[Title]` brackets for a plain content card:

````
:::card
Sarde outputs a static `dist/` folder ready for any CDN or static host.
:::
````

:::card
Sarde outputs a static `dist/` folder ready for any CDN or static host.
:::

## Rich Content

Card bodies accept any Markdown content — lists, code blocks, headings:

````
:::card[Supported Frontmatter]{icon="file-text"}
| Key | Type | Description |
|-----|------|-------------|
| `title` | string | Page title |
| `description` | string | Page description |
| `date` | string | Publication date |
| `tags` | list | Taxonomy tags |
:::
````

:::card[Supported Frontmatter]{icon="file-text"}
| Key | Type | Description |
|-----|------|-------------|
| `title` | string | Page title |
| `description` | string | Page description |
| `date` | string | Publication date |
| `tags` | list | Taxonomy tags |
:::

## Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `[Title]` | string | No | Card title, placed in brackets after `card` |
| `{icon}` | string | No | Lucide icon name displayed next to the title |
