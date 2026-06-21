---
title: "Link Cards"
description: "Preview cards that link to other pages or external resources."
sidebar:
  order: 7
---

Link cards display a clickable preview card with a title, description, and optional icon. The entire card is a link. Use them to build navigation grids or highlight related resources.

## Basic Usage

````
:::link-card[Getting Started]{href="/docs/getting-started/" description="Install Sarde and scaffold your first site."}
:::
````

:::link-card[Getting Started]{href="/docs/getting-started/" description="Install Sarde and scaffold your first site."}
:::

## With Icon

Add `icon="name"` to display a Lucide icon:

````
:::link-card[Configuration]{href="/docs/configuration/" description="All sarde.yaml options explained." icon="settings"}
:::
````

:::link-card[Configuration]{href="/docs/configuration/" description="All sarde.yaml options explained." icon="settings"}
:::

## Body Description

Place the description as body content instead of an attribute:

````
:::link-card[Collections]{href="/docs/collections/"}
How Sarde detects blog and docs collections from directory names, and how to override the defaults.
:::
````

:::link-card[Collections]{href="/docs/collections/"}
How Sarde detects blog and docs collections from directory names, and how to override the defaults.
:::

## External Links

Link cards work with external URLs. Sarde adds an external link indicator automatically:

````
:::link-card[Go Documentation]{href="https://pkg.go.dev/std" description="Standard library reference for Go." icon="external-link"}
:::
````

## In a Grid

Link cards are most useful inside a [Card Grid](/docs/components/card-grids/):

````
:::card-grid

:::link-card[Callouts]{href="/docs/components/callouts/" description="Highlight important information."}
:::

:::link-card[Code Blocks]{href="/docs/components/code-blocks/" description="Syntax highlighting with advanced features."}
:::

:::link-card[Tabs]{href="/docs/components/tabs/" description="Switchable content panels."}
:::

:::
````

## Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `[Title]` | string | Yes | Card title, placed in brackets after `link-card` |
| `href` | string | Yes | Link destination (relative or absolute URL) |
| `description` | string | No | Short description shown below the title |
| `icon` | string | No | Lucide icon name displayed next to the title |
