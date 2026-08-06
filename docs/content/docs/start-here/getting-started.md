---
title: Getting Started
description: "Install Sarde, scaffold a new site, preview it locally, and produce a production build"
sidebar:
  order: 2
---

This guide covers installing Sarde and creating a first website with it. No step assumes prior experience with build tools. By the end, a themed site is running in the browser and updating itself on every save, a first page sits in the sidebar, and a production build waits in `dist/`, ready to put on the web.

## Install Sarde

:::tabs

== Homebrew

On macOS or Linux with [Homebrew](https://brew.sh/):

```sh
brew install getsarde/sarde/sarde
```

== Shell script

The install script supports macOS and Linux. On Windows, use the Windows tab instead.

```sh
curl -sSfL https://raw.githubusercontent.com/getsarde/sarde/main/install.sh | sh
```

== Windows

1. Download `sarde_windows_amd64.zip` from [GitHub Releases](https://github.com/getsarde/sarde/releases).
2. Extract the archive and move `sarde.exe` into a permanent folder, for example `C:\Users\<name>\sarde`.
3. Add that folder to the `PATH` environment variable so any terminal can find the program: open **Settings > System > About > Advanced system settings**, click **Environment Variables**, select `Path` under *User variables*, then click **Edit > New** and paste the folder path.
4. Open a new terminal window. Terminals that were already open do not pick up the `PATH` change.

== Binary download

Download the latest release archive for your platform from [GitHub Releases](https://github.com/getsarde/sarde/releases). Extract it and place the `sarde` binary in a directory on the `PATH`.

== From source

Requires [Go](https://go.dev/dl/) 1.25 or later.

```sh
go install github.com/getsarde/sarde/cmd/sarde@latest
```

:::

Verify the installation:

```sh
sarde version
```

→ The terminal prints:

```text
sarde 1.0.0
Go: go1.25.0
OS/Arch: linux/amd64
```

## Create a site

One command produces a complete, working site: configuration, a homepage, a sample blog post, and a sample docs page. There is nothing to assemble before seeing results; writing can start from a site that already works.

```sh
sarde new site my-site
cd my-site
```

→ The terminal prints:

```text
Created new site at /path/to/my-site
  Run 'sarde dev' to start the dev server.
```

The scaffolded project contains everything needed to start:

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
  .gitignore
```

The `sarde.yaml` file controls the site title, theme preset, homepage hero, and all other settings; see [Configuration](/reference/configuration/) for the full reference. The `kazari.config.yaml` file controls syntax highlighting, covered in [Code Blocks](/guides/code-blocks/).

## Start the dev server

The dev server is a private preview of the site, running only on this machine; nobody else can see it. It watches the project's files, and every time a file is saved it rebuilds the affected pages and refreshes the browser on its own. Leave it running in the terminal while writing.

```sh
sarde dev
```

→ The terminal prints:

```text
 sarde  v1.0.0 ready in 320 ms
┃ Local    http://localhost:4727
┃ Network  use --host 0.0.0.0 to expose
```

Open `http://localhost:4727` in a browser.

→ A finished-looking site appears: a homepage with a hero section, a navigation bar linking to the sample blog and docs pages, working search, and a dark mode toggle. All of it comes from the scaffold; no configuration was involved.

From here on, the loop is: edit a file, save, glance at the browser. The page refreshes automatically, and CSS changes appear without even a page reload.

[Draft, scheduled, and expired content](/guides/writing-content/#drafts-scheduled-and-expiring-content) is included by default in dev mode. Use `--no-drafts` to exclude it.

## Add content

Time to add a page of your own. `sarde new` creates the file in the right folder with the metadata already filled in:

```sh
sarde new docs "My First Page"
```

→ The terminal prints:

```text
Created content/docs/my-first-page.md
```

Open `content/docs/my-first-page.md` in an editor. The file starts with *frontmatter*, a short metadata block between `---` fences that sets the page title and other fields. Sarde pre-fills it:

```yaml
---
draft: true
title: My First Page
date: 2026-03-15T09:00:00-05:00
---
```

The `draft: true` line marks the page as work in progress: the dev server shows it, but `sarde build` leaves it out. Remove the line (or set it to `false`) when the page is ready to publish.

Add Markdown content below the frontmatter and save the file:

```markdown
## Welcome

This page was created with `sarde new docs`.

:::note
Sarde's Markdown goes beyond the basics. Asides like this one, tabs, cards,
and more are built in.
:::
```

The browser updates automatically, the new page appears in the sidebar navigation, and the `:::note` block renders as a colored callout. See [Using Extensions](/extensions/using-extensions/) for the full extended syntax.

Sarde auto-detects the collection type from the directory name. Content in `docs/` gets the docs layout with sidebar navigation, while content in `blog/` gets the blog layout with date-sorted posts.

## Build the site

Building is the step that turns the project into something publishable. The dev server renders pages on demand for one viewer; a build writes every page out ahead of time as plain files, so a web host can serve them to anyone without running Sarde at all.

Remove the `draft: true` line from the new page, then build:

```sh
sarde build
```

→ The terminal prints:

```text
Built in 320 ms
  Output: /path/to/my-site/dist
```

The `dist/` directory is the complete site as plain HTML, CSS, and JavaScript. It needs no server-side runtime, so any static host can serve it. [Deploying](/start-here/deploying/) covers publishing it to GitHub Pages, Netlify, Cloudflare Pages, Vercel, or a custom target.

## Next steps

- [Writing Content](/guides/writing-content/) covers frontmatter, drafts, and page bundles
- [Content and Collections](/guides/content-and-collections/) explains how folders become blogs, docs, and courses
- [Using Extensions](/extensions/using-extensions/) tours the extended Markdown: asides, tabs, cards, and more
- [Deploying](/start-here/deploying/) publishes the site to the web
