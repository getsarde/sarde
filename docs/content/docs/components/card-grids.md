---
title: "Card Grids"
description: "Arrange cards in a responsive grid layout."
sidebar:
  order: 6
---

Card grids arrange multiple cards in a responsive grid that adapts to screen width. Wrap any number of `:::card` or `:::link-card` blocks inside `:::card-grid`.

## Basic Usage

````
:::card-grid

:::card[Install]
Drop the `sarde` binary into your `PATH`.
:::

:::card[Scaffold]
Run `sarde init my-site` to create a new project.
:::

:::card[Write]
Add Markdown files to the `content/` folder.
:::

:::card[Build]
Run `sarde build` to generate the static site.
:::

:::
````

:::card-grid

:::card[Install]
Drop the `sarde` binary into your `PATH`.
:::

:::card[Scaffold]
Run `sarde init my-site` to create a new project.
:::

:::card[Write]
Add Markdown files to the `content/` folder.
:::

:::card[Build]
Run `sarde build` to generate the static site.
:::

:::

## With Link Cards

`:::card-grid` works with `:::link-card` blocks too. This combination is useful for navigation index pages:

````
:::card-grid

:::link-card[Configuration]{href="/docs/configuration/" description="All sarde.yaml options explained."}
:::

:::link-card[Collections]{href="/docs/collections/" description="How Sarde detects and configures content collections."}
:::

:::link-card[Plugins]{href="/docs/plugins/" description="Built-in plugins and how to configure them."}
:::

:::
````

:::card-grid

:::link-card[Configuration]{href="/docs/configuration/" description="All sarde.yaml options explained."}
:::

:::link-card[Collections]{href="/docs/collections/" description="How Sarde detects and configures content collections."}
:::

:::link-card[Plugins]{href="/docs/plugins/" description="Built-in plugins and how to configure them."}
:::

:::

## With Icons

Combine cards with icons for a richer grid:

````
:::card-grid

:::card[Markdown]{icon="file-text"}
Write content in standard Markdown with optional frontmatter.
:::

:::card[Extensions]{icon="puzzle"}
Use callouts, tabs, cards, code groups, and more out of the box.
:::

:::card[Themes]{icon="palette"}
Override colors, fonts, and layouts with a theme overlay.
:::

:::
````

:::card-grid

:::card[Markdown]{icon="file-text"}
Write content in standard Markdown with optional frontmatter.
:::

:::card[Extensions]{icon="puzzle"}
Use callouts, tabs, cards, code groups, and more out of the box.
:::

:::card[Themes]{icon="palette"}
Override colors, fonts, and layouts with a theme overlay.
:::

:::
