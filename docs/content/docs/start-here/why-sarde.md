---
title: Why Sarde
description: "Sarde provides the common parts of a content site without requiring a frontend toolchain."
sidebar:
  order: 2
---

## Working defaults

New sites include a theme, navigation, search, and production build settings.

```bash
sarde init my-site
cd my-site
sarde dev
```

Configuration changes the defaults instead of creating the site from scratch.

## File-based structure

The structure inside `content/` controls collections, sections, and URLs. Pages remain ordinary Markdown files that work well with Git and any text editor.

## Portable output

`sarde build` creates static files with no server-side runtime. The same output can move between static hosts.

## Tradeoffs

Sarde favors conventions over application-level control. Use an application framework for server rendering or custom runtime behavior, and use a content management system for browser-based editing and publishing workflows.
