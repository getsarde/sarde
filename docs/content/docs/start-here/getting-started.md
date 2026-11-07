---
title: Getting Started
description: "Scaffold a new site, preview it locally, add a page, and produce a production build"
sidebar:
  order: 3
---

Create a site and preview it in the browser. By the end, a themed site runs locally and updates on every save, a new page appears in its sidebar, and a production build sits in `dist/`, ready to publish. The steps need Sarde [installed](/start-here/installation/), a terminal, and a text editor, and assume no experience with build tools.

## Install Sarde

Follow [Installation](/start-here/installation/) to install Sarde, then check that the terminal finds it:

```sh
sarde version
```

→ The terminal prints the Sarde version, for example `sarde 1.8.0`, followed by the Go version and the platform.

## Create a site

`sarde new site` creates a working site in a new folder: a configuration file, a homepage, a sample blog post, and a sample docs page. Create a site named `my-site` and move into its folder:

```sh
sarde new site my-site
cd my-site
```

→ The terminal prints:

```text
Created new site at /path/to/my-site
  Run 'cd my-site' then 'sarde dev' to start the dev server.
```

The `my-site` folder contains:

```text
my-site/
  sarde.yaml            # Site configuration
  kazari.config.yaml    # Code block highlighting settings
  content/
    _index.md            # Homepage
    blog/
      _index.md
      hello-world.md     # Sample blog post
    docs/
      _index.md
      getting-started.md # Sample docs page
  public/
    images/
      hero-light.svg     # Homepage illustration, light mode
      hero-dark.svg      # Homepage illustration, dark mode
  .gitignore
```

:::tip
To build a course site, start from the course template instead: `sarde new site my-course --template course` creates a course website with lessons, assignments, schedules, announcements, and hands-on labs. It downloads the template from GitHub, so it needs an internet connection. See [Course Template](/teaching/course-template/).
:::

The `sarde.yaml` file controls the site title, theme preset, homepage hero, and all other settings. [Configuration Examples](/reference/configuration/examples/) has complete files to start from, and [Configuration](/reference/configuration/) documents every key. The `kazari.config.yaml` file controls syntax highlighting, covered in [Code Blocks](/guides/code-blocks/).

## Start the dev server

The dev server is a private preview of the site that runs only on this machine. It watches the project files, and on every save it rebuilds the affected pages and refreshes the browser. Start it and leave it running while writing:

```sh
sarde dev
```

→ The terminal prints:

```text
 sarde  v1.8.0 ready in 320 ms
┃ Local    http://localhost:4727
┃ Network  use --host 0.0.0.0 to expose
```

Open `http://localhost:4727` in a browser.

→ The site appears with a homepage hero section, a navigation bar that links to the sample blog and docs pages, search, and a dark mode toggle. All of it comes from the scaffold, with no configuration.

From here on, edit a file, save it, and check the browser. The page refreshes on its own, and CSS changes apply without a page reload. To stop the dev server, press `Ctrl+C` in its terminal.

## Add content

The dev server occupies its terminal, so open a second terminal window and move into the `my-site` folder. Create a page with `sarde new`, which takes the name of a *collection* (a top-level folder in `content/`) and a page title:

```sh
sarde new docs "My First Page"
```

→ The terminal prints:

```text
Created content/docs/my-first-page.md
```

Open `content/docs/my-first-page.md` in a text editor. The file starts with *frontmatter*, a short metadata block between `---` fences that sets the page title and other fields. Sarde fills it in:

```yaml
---
date: "2026-03-15T09:00:00-05:00"
draft: true
title: My First Page
---
```

The `draft: true` line marks the page as work in progress. The dev server shows drafts, and `sarde build` leaves them out. [Drafts, scheduled, and expiring content](/guides/writing-content/#drafts-scheduled-and-expiring-content) explains the publishing rules.

Add Markdown content below the frontmatter and save the file:

```markdown
## Welcome

This page was created with `sarde new docs`.

:::note
Sarde's Markdown goes beyond the basics. Asides like this one, tabs, cards,
and more are built in.
:::
```

Open `http://localhost:4727/docs/my-first-page/` in the browser.

→ **My First Page** appears in the docs sidebar, and the `:::note` block renders as a colored callout.

The page joined the docs sidebar because of the folder it is in. Sarde infers how a collection behaves from its directory name: `docs/` gets sidebar navigation, and `blog/` gets posts sorted newest first. [Core Concepts](/start-here/core-concepts/#collections) lists the recognized names, and [Using Extensions](/extensions/using-extensions/) covers the full extended syntax.

## Build the site

`sarde build` writes the whole site into `dist/` as plain files that a web host can serve without running Sarde. A build leaves drafts out, so remove the `draft: true` line from `content/docs/my-first-page.md` and save the file.

The dev server and the build both write to `dist/`, and only one of them can use it at a time. A build started while the dev server runs stops with `another sarde process ... is already writing to output directory`. Stop the dev server with `Ctrl+C`, then build:

```sh
sarde build
```

→ The output ends with:

```text
Built 9 pages in 320 ms
  Output: /path/to/my-site/dist
```

Before those lines, the build prints the link check result and a table of page counts.

The `dist/` directory is the complete site as HTML, CSS, and JavaScript. [Deploying](/start-here/deploying/) covers publishing it to GitHub Pages, Netlify, Cloudflare Pages, Vercel, or another host.

## Next steps

- [Write Your First Pages](/start-here/writing-your-first-pages/) continues with this site: Markdown, images, links, sidebar order, and the homepage
- [Core Concepts](/start-here/core-concepts/) explains how content, configuration, and a theme combine into a site
- [Deploying](/start-here/deploying/) publishes `dist/` to a web host
