---
title: Deploying
description: "Build a production site and deploy it to GitHub Pages, Netlify, Cloudflare Pages, Vercel, or a custom target"
sidebar:
  order: 2
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

Sarde has built-in GitHub Pages deployment. Add the deploy config to `sarde.yaml`:

```yaml
deploy:
  provider: github
  branch: gh-pages
```

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

This force-pushes the `dist/` contents to the `gh-pages` branch. In the GitHub repository settings, configure Pages to serve from that branch.

:::note
The `github` deployer requires a git remote named `origin` pointing to the repository.
:::

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

## CI/CD

A minimal GitHub Actions workflow that builds and deploys on every push to `main`:

```yaml title=".github/workflows/deploy.yml"
name: Deploy
on:
  push:
    branches: [main]
jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Install Sarde
        run: curl -sSfL https://raw.githubusercontent.com/getsarde/sarde/main/install.sh | sh
      - name: Build and deploy
        run: |
          sarde build
          sarde deploy --provider github
```

The `github` deployer auto-configures a git identity if none is set, so this works in fresh CI environments without additional setup.

See [`deploy`](/reference/cli-commands#deploy) in CLI Commands for all available flags, and [`deploy`](/reference/configuration#deploy) in Configuration for all config options.
