---
title: What Is Sarde
description: "Sarde reads a folder of Markdown files and writes a complete, themed static website."
aliases:
  - /start-here/why-sarde/
sidebar:
  order: 1
---

Sarde is a static site generator (SSG). It reads a folder of Markdown files and writes a complete, themed website that can be hosted anywhere. It suits product and API documentation, course materials, and blogs.

The first build produces a complete site, with navigation, syntax highlighting, and search in place. Configuration changes those defaults, and none of it is required.

## How it works

Content lives in `content/`. Every Markdown file becomes one page.

```text
content/
├── _index.md
└── docs/
    ├── _index.md
    └── photosynthesis.md
```

Build the site:

```sh
sarde build
```

→ The output ends with:

```text
Built 4 pages in 320 ms
  Output: /path/to/my-site/dist
```

`content/docs/photosynthesis.md` is now a page at `/docs/photosynthesis/`. The file path determines the URL, so moving or renaming a file changes the URL of its page.

## Build output

`sarde build` writes everything the site needs into `dist/`:

- One HTML page per Markdown file, on a responsive theme with light and dark modes
- A sidebar built from the directory structure, and a table of contents per page
- Full-text search that runs offline in the browser
- Syntax highlighting for code blocks
- Link prefetching on hover
- A compiled CSS bundle and a small JavaScript bundle
- Responsive images converted to WebP, with low-quality placeholders
- RSS and Atom feeds for blog collections, `sitemap.xml`, `robots.txt`, and `llms.txt`
- Social card images for link previews

Every build also checks internal links and anchors, and most broken ones stop the build.

Nothing in `dist/` needs Sarde or Go at runtime. The output is plain HTML, CSS, and JavaScript, so it runs on GitHub Pages, Netlify, Cloudflare Pages, Vercel, an object storage bucket, or a directory served by nginx. If Sarde stops being the right tool later, the built site keeps working and the Markdown sources stay readable.

## A single executable

Sarde is a single compiled executable. Installing it puts one file on the `PATH`.

There is no `node_modules` directory, no lockfile, and no dependency install before the first build. A continuous integration job needs one step to install Sarde, then `sarde build`.

## Content in plain files

Pages are Markdown with a short block of frontmatter at the top:

```markdown title="content/docs/photosynthesis.md"
---
title: Photosynthesis Overview
sidebar:
  order: 2
---

## How it works

Plants convert light energy into chemical energy through a series of reactions
in the chloroplast.
```

The file opens in any text editor. It diffs cleanly in review, merges like source code, and carries its history in Git. Moving the content to another tool needs no export step, because the Markdown files are the only copy.

## Convention-based defaults

Sarde reads the folder layout and infers how each group of content behaves. A `docs/` directory gets documentation navigation with a collapsible sidebar and Previous/Next links. A `blog/` directory gets posts sorted newest first, with a feed. Any other directory name also works, and an unrecognized name becomes a general collection sorted by title.

A `sarde.yaml` file at the project root changes the defaults. Every setting has a default, so the file is optional and needs only the values that differ.

[Core Concepts](/start-here/core-concepts/) covers which directory names mean what, and [Configuration](/reference/configuration/) lists every key.

## When to use Sarde

Sarde fits sites where files are the source of truth and content changes through edits and commits:

- Documentation and API references
- Course notes, lab handouts, and teaching material
- Handbooks and internal wikis
- Blogs and changelogs

That design comes with tradeoffs:

- **Directory names change behavior**: Renaming `docs/` to `handbook/` changes the inferred layout and sorting unless the collection is configured explicitly.
- **No server-side rendering**: Pages are built once, so anything that varies per visitor has to happen in the browser or in a separate service.
- **No browser-based editing**: Publishing means editing a file and running a build, usually through Git. Contributors who expect a publish button need a content management system instead.
- **A smaller ecosystem**: Hugo and Astro have more themes, more plugins, and more answered questions. Sarde targets documentation, course, and blog sites.

Continue to [Getting Started](/start-here/getting-started/) to install Sarde and build a first site. Once it runs, [Core Concepts](/start-here/core-concepts/) explains the model behind these defaults.
