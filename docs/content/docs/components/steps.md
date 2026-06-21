---
title: "Steps"
description: "Numbered step-by-step guides with a visual timeline."
sidebar:
  order: 9
---

Steps display a numbered sequence of instructions connected by a vertical line. Two modes are available: heading mode for multi-paragraph steps, and list mode for brief single-line steps.

## Heading Mode

Use `### ` headings inside the block to define step boundaries. Content below each heading belongs to that step.

````
:::steps

### Install Sarde

Download the binary for your platform from the releases page.

### Initialize a project

Run the following command in the directory where the site will live:

```bash
sarde init my-site
```

### Start the dev server

```bash
cd my-site
sarde dev
```

Open `http://localhost:4727` to see the site.

:::
````

:::steps

### Install Sarde

Download the binary for your platform from the releases page.

### Initialize a project

Run the following command in the directory where the site will live:

```bash
sarde init my-site
```

### Start the dev server

```bash
cd my-site
sarde dev
```

Open `http://localhost:4727` to see the site.

:::

## List Mode

Use a standard ordered list for brief, single-line steps.

```
:::steps
1. Download the Sarde binary.
2. Place it somewhere on your `PATH`.
3. Run `sarde init my-site`.
4. Run `sarde dev` to start the dev server.
:::
```

:::steps
1. Download the Sarde binary.
2. Place it somewhere on your `PATH`.
3. Run `sarde init my-site`.
4. Run `sarde dev` to start the dev server.
:::

List mode is best when each step fits in one sentence. Use heading mode when steps need code blocks, notes, or multiple paragraphs.
