---
title: Getting Started
description: "Install Sarde, scaffold a new site, and preview it with the local dev server"
sidebar:
  order: 1
---

Sarde is a zero-config static site generator. Drop Markdown files into a `content/` folder and get a fully-themed, production-ready website. This page covers installing Sarde, creating a site, and previewing it locally.

## Install Sarde

:::tabs

=== Homebrew

```sh
brew install getsarde/sarde/sarde
```

=== Shell script

```sh
curl -sSfL https://raw.githubusercontent.com/getsarde/sarde/main/install.sh | sh
```

=== Binary download

Download the latest release for your platform from [GitHub Releases](https://github.com/getsarde/sarde/releases). Extract the archive and place the `sarde` binary in a directory on your `PATH`.

=== From source

Requires [Go](https://go.dev/dl/) 1.21 or later.

```sh
go install github.com/getsarde/sarde/cmd/sarde@latest
```

:::

Verify the installation:

```sh
sarde version
```

→ Prints the installed version number and build platform.

## Create a site

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
  static/
    images/
  .gitignore
```

The `sarde.yaml` file controls the site title, theme preset, homepage hero, and all other settings. See [Configuration](/reference/configuration) for the full reference.

## Start the dev server

```sh
sarde dev
```

→ The terminal prints:

```text
 sarde  v1.0.0 ready in 320 ms
┃ Local    http://localhost:4727
┃ Network  use --host 0.0.0.0 to expose
```

Open `http://localhost:4727` in a browser to see the site. The dev server watches for file changes and reloads the browser automatically. CSS changes hot-swap without a full page reload.

Draft and expired content is included by default in dev mode. Use `--no-drafts` to exclude it.

## Add content

Create a new docs page:

```sh
sarde new docs "My First Page"
```

→ The terminal prints:

```text
Created content/docs/my-first-page.md
```

Open `content/docs/my-first-page.md` in an editor. The file starts with frontmatter pre-filled by Sarde:

```yaml
---
title: My First Page
---
```

Add Markdown content below the frontmatter, save the file, and the browser updates automatically. The new page appears in the sidebar navigation.

Sarde auto-detects the collection type from the directory name. Content in `docs/` gets the docs layout with sidebar navigation, while content in `blog/` gets the blog layout with date-sorted posts.

## Next steps

- [Configuration](/reference/configuration) to customize `sarde.yaml`
- [Content & Collections](/guides/content-and-collections) to understand collections, sections, and page bundles
- [Deploying](/start-here/deploying) to publish the site to the web
