---
title: Deploying
description: "Build a production site and deploy it to GitHub Pages, Netlify, Cloudflare Pages, Vercel, or a custom target"
sidebar:
  order: 4
---

Sarde builds a static site that works on any hosting provider. Run `sarde build` to generate the output, then deploy the result to your platform of choice.

## Build for production

```sh
sarde build
```

→ The terminal prints a summary:

```text
Built in 320 ms
  Output: /path/to/my-site/dist
```

The `dist/` directory contains the complete site: HTML pages, CSS, JavaScript, images, feeds, sitemap, and search index. Upload this directory to any static hosting provider.

Override the output directory with `--output` or [`build.output`](/reference/configuration#build) in `sarde.yaml`.

## GitHub Pages

You can publish to GitHub Pages in two ways:

- **GitHub Actions** builds the site on every push and publishes it with the official Pages actions. Use this method for automated deploys.
- **`sarde deploy`** force-pushes an already built `dist/` directory to a branch that GitHub Pages serves. Use this method to publish from your own machine.

Both methods need the same site URL and base path settings.

### Set the site URL and base path

GitHub serves a project site from a subdirectory named after the repository, for example `https://octocat.github.io/my-site/`. Set `site.url` to the host and `build.base_path` to the repository name:

```yaml title="sarde.yaml"
site:
  url: https://octocat.github.io
build:
  base_path: /my-site/
```

Replace `octocat` with your GitHub user or organization name and `my-site` with the repository name. Without `base_path`, the build works locally but the deployed site loads with broken styles and links.

A user or organization site (a repository named `octocat.github.io`) is served from the root. Set `site.url` and leave `base_path` unset.

:::caution
Do not use the `--baseURL` flag to set the subdirectory. It overrides `site.url`, not `base_path`, so canonical links, Open Graph tags, the sitemap, and feeds end up with the wrong URL.
:::

See [`site`](/reference/configuration#site) and [`build`](/reference/configuration#build) in Configuration for both options.

### Deploy with GitHub Actions

1. In the repository on GitHub, open **Settings > Pages** and set **Source** to **GitHub Actions**.

2. Add this workflow to build and publish the site on every push to `main`:

   ```yaml title=".github/workflows/deploy.yml"
   name: Deploy

   on:
     push:
       branches: [main]
     workflow_dispatch:

   permissions:
     contents: read
     pages: write
     id-token: write

   concurrency:
     group: pages
     cancel-in-progress: false

   jobs:
     build:
       runs-on: ubuntu-latest
       steps:
         - uses: actions/checkout@v4
           with:
             fetch-depth: 0

         - name: Install Sarde
           run: |
             curl -sSfL https://raw.githubusercontent.com/getsarde/sarde/main/install.sh | sh
             echo "$HOME/.local/bin" >> "$GITHUB_PATH"

         - name: Build
           run: sarde build

         - uses: actions/configure-pages@v5

         - uses: actions/upload-pages-artifact@v3
           with:
             path: dist

     deploy:
       needs: build
       runs-on: ubuntu-latest
       environment:
         name: github-pages
         url: ${{ steps.deployment.outputs.page_url }}
       steps:
         - id: deployment
           uses: actions/deploy-pages@v4
   ```

   `fetch-depth: 0` checks out the full git history. Sarde reads each page's last-updated date from its most recent commit, and in a shallow clone pages whose last change is older than the clone fall back to the checkout time. See [Last-updated strategy](/reference/configuration#last-updated-strategy).

   The install script falls back to `~/.local/bin` when it cannot write to `/usr/local/bin`, so the workflow adds that directory to `PATH`.

3. Commit the workflow and push to `main`.

→ The **Actions** tab shows the `Deploy` workflow. When both jobs finish, the `deploy` job links to the published site.

If the site lives in a subdirectory of the repository, pass it to the build and point the artifact at its output, for example `sarde build docs` and `path: docs/dist`.

### Deploy to a branch with `sarde deploy`

The built-in `github` provider publishes `dist/` to a branch. Add the deploy config to `sarde.yaml`:

```yaml title="sarde.yaml"
deploy:
  provider: github
  branch: gh-pages
```

`branch` defaults to `gh-pages` when omitted.

Build and deploy:

```sh
sarde build
sarde deploy
```

→ The terminal prints:

```text
Deploying with github-pages...
Deploy complete.
```

The deployer copies `dist/` into a temporary git repository, adds an empty `.nojekyll` file, commits, and force-pushes the commit to the `gh-pages` branch of the `origin` remote. Each deploy replaces the branch contents and history. The `.nojekyll` file stops GitHub Pages from running Jekyll, which would drop any file or directory whose name starts with an underscore.

After the first deploy, open **Settings > Pages** in the repository, set **Source** to **Deploy from a branch**, and select `gh-pages`.

:::note
The `github` provider requires a git remote named `origin` that you can push to. If git has no user identity configured, the deploy commit uses the name `sarde-deploy`.
:::

See [`deploy`](/reference/cli-commands#deploy) in CLI Commands for all flags, and [`deploy`](/reference/configuration#deploy) in Configuration for all options.

## Netlify

Build the site, then deploy using the Netlify CLI:

```sh
sarde build
npx netlify deploy --prod --dir dist
```

To generate a Netlify-compatible `_redirects` file for any configured redirects or page aliases, set the redirect format in `sarde.yaml`:

```yaml
deploy:
  redirect_format: netlify
```

:::tip
Wrap the Netlify CLI in a [`custom` provider](#custom-deployment) to deploy with `sarde deploy` instead of running two separate commands.
:::

## Cloudflare Pages

Build the site, then deploy using Wrangler:

```sh
sarde build
npx wrangler pages deploy dist --project-name my-site
```

Alternatively, connect the Git repository in the Cloudflare dashboard. Set the build command to `sarde build` and the output directory to `dist`.

## Vercel

Build the site, then deploy using the Vercel CLI:

```sh
sarde build
npx vercel deploy --prod dist
```

To generate a Vercel-compatible `vercel.json` with redirect rules, set the redirect format:

```yaml
deploy:
  redirect_format: vercel
```

Alternatively, connect the Git repository in the Vercel dashboard. Set the build command to `sarde build` and the output directory to `dist`.

## Custom deployment

The `custom` provider runs any shell command with the `DIST_DIR` environment variable set to the absolute path of the output directory.

```yaml
deploy:
  provider: custom
  command: "rsync -avz $DIST_DIR/ user@server:/var/www/html/"
```

```sh
sarde build
sarde deploy
```

This also works as a wrapper for provider CLIs:

```yaml
deploy:
  provider: custom
  command: "npx netlify deploy --prod --dir $DIST_DIR"
```
