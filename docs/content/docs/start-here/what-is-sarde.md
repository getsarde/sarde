---
title: What Is Sarde
description: "Sarde turns Markdown files into a themed static website."
sidebar:
  order: 1
---

## Markdown in, website out

Add pages to `content/`, then build the site:

```bash
sarde build
```

Sarde writes the generated HTML, CSS, JavaScript, and other files to `dist/`.

## Built around content

Directory names set useful defaults. A `docs/` directory gets documentation navigation, while a `blog/` directory gets date-sorted posts and feeds.

```text
content/
├── docs/
│   └── installation.md
└── blog/
    └── first-post.md
```

Use `sarde.yaml` to change site-wide settings such as the title, theme, navigation, and build output.

## Static by default

The generated site does not need Sarde or Go at runtime. Deploy `dist/` to any static hosting provider.

Sarde is best suited to documentation, blogs, handbooks, and other sites where files are the source of truth.
