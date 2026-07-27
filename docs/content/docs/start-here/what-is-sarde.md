---
title: What Is Sarde
description: "Sarde reads a folder of Markdown files and writes a complete, themed static website."
aliases:
  - /start-here/why-sarde/
sidebar:
  order: 1
---

Sarde is a static site generator (SSG). It reads a folder of Markdown files and writes a complete, themed website that can be hosted anywhere.

There is no frontend project to set up first, and no build tooling to assemble. A site is complete on the first build, with navigation, syntax highlighting, and search already in place. Configuration adjusts those defaults rather than producing them.

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

→ The terminal prints:

```text
Built in 320 ms
  Output: /path/to/my-site/dist
```

`content/docs/photosynthesis.md` is now a page at `/docs/photosynthesis/`. The file path determines the URL, so moving a file moves its page and renaming a file renames its URL.

## Build output

`sarde build` writes everything the site needs into `dist/`:

- One HTML page per Markdown file, on a responsive theme with light and dark modes
- A sidebar built from the directory structure, and a table of contents per page
- A compiled CSS bundle and a small JavaScript bundle
- Full-text search that runs offline in the browser
- Syntax highlighting for code blocks
- Responsive images converted to WebP, with low-quality placeholders
- RSS and Atom feeds for date-sorted collections, `sitemap.xml`, and `robots.txt`
- Social card images for link previews
- Link prefetching on hover, and internal link validation on every build

Nothing in `dist/` needs Sarde or Go at runtime. The output is plain HTML, CSS, and JavaScript, so it runs on GitHub Pages, Netlify, Cloudflare Pages, Vercel, an object storage bucket, or a directory served by nginx. If Sarde stops being the right tool later, the built site keeps working and the Markdown sources stay readable.

## No toolchain

Sarde is a single compiled executable. Installing it puts one file on the `PATH`.

There is no `node_modules` directory, no lockfile, and no dependency install before the first build. A project checked out two years from now builds with the same binary and produces the same output. A continuous integration job needs one step to install Sarde, then `sarde build`.

## Content in plain files

Pages are Markdown with a short block of frontmatter at the top:

`content/docs/photosynthesis.md`

```markdown
---
title: Photosynthesis Overview
sidebar:
  order: 2
---

## How it works

Plants convert light energy into chemical energy through a series of reactions
in the chloroplast.
```

That file opens in any text editor. It diffs cleanly in review, merges like source code, and carries its history in Git. Moving the content elsewhere needs no export step, because it was never held in a proprietary store.

## Convention-based defaults

Sarde reads the folder layout and infers how each group of content should behave. A `docs/` directory gets documentation navigation with a collapsible sidebar and Previous/Next links. A `blog/` directory gets posts sorted newest first, with a feed. These names are a convention, not a requirement: any name works, and an unrecognized directory becomes a general collection sorted by title.

`sarde.yaml` at the project root changes the defaults. Every setting has one, so the file only needs the values that differ, and an empty file is valid.

[Core Concepts](/start-here/core-concepts/) covers which directory names mean what, and [Configuration](/reference/configuration/) lists every key.

## When to use Sarde

Sarde fits sites where files are the source of truth and content changes through edits and commits:

- Documentation and API references
- Course notes, lab handouts, and teaching material
- Handbooks and internal wikis
- Blogs and changelogs

The choices that make those sites easy cost flexibility elsewhere:

- **Conventions carry weight**: Directory names determine layout and sorting, so renaming `docs/` to `handbook/` changes the inferred behavior unless the collection is configured explicitly.
- **No server-side rendering**: Pages are built once, so anything that varies per visitor has to happen in the browser or in a separate service.
- **No browser-based editing**: Publishing means editing a file and running a build, usually through Git. Contributors who expect a publish button need a content management system instead.
- **A smaller ecosystem**: Hugo and Astro have more themes, more plugins, and more answered questions. Sarde covers documentation and content sites thoroughly rather than trying to cover everything.

Continue to [Getting Started](/start-here/getting-started/) to install Sarde, or to [Core Concepts](/start-here/core-concepts/) for the model behind these defaults.
