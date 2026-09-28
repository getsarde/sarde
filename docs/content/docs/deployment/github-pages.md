---
title: GitHub Pages
description: "Deploy a Sarde site to GitHub Pages with GitHub Actions or by pushing to a gh-pages branch"
sidebar:
  order: 1
---

This page publishes a Sarde site to GitHub Pages, either automatically on every push with GitHub Actions or from your machine with `sarde deploy`.

## Project configuration

GitHub serves a project site from a subdirectory named after the repository, for example `https://octocat.github.io/my-site/`. Set `site.url` to the host and `build.base_path` to the repository name:

```yaml title="sarde.yaml"
site:
  url: https://octocat.github.io
build:
  base_path: /my-site/
```

Replace `octocat` with your GitHub user or organization name and `my-site` with the repository name. Without `base_path`, the build works locally but the deployed site loads with broken styles and links.

A user or organization site (a repository named `octocat.github.io`) is served from the root. Set `site.url` and leave `base_path` unset.

Write links between pages without the base path, for example `[Setup](/guides/setup/)`. Sarde adds `base_path` to every built link.

:::caution
Do not use the `--baseURL` flag to set the subdirectory. It overrides `site.url`, not `base_path`, so canonical links, Open Graph tags, the sitemap, and feeds end up with the wrong URL.
:::

## Deploy with GitHub Actions

The official [`getsarde/action`](https://github.com/getsarde/action) installs Sarde, builds the site, and uploads `dist/` for GitHub Pages.

1. In the repository on GitHub, open **Settings > Pages** and set **Source** to **GitHub Actions**.

2. Create `.github/workflows/deploy.yml`:

   ```yaml title=".github/workflows/deploy.yml"
   name: Deploy to GitHub Pages

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
         - uses: actions/checkout@v7
           with:
             fetch-depth: 0
         - uses: actions/configure-pages@v6
         - uses: getsarde/action@v1

     deploy:
       needs: build
       runs-on: ubuntu-latest
       environment:
         name: github-pages
         url: ${{ steps.deployment.outputs.page_url }}
       steps:
         - name: Deploy to GitHub Pages
           id: deployment
           uses: actions/deploy-pages@v5
   ```

   `fetch-depth: 0` checks out the full git history. Sarde reads each page's last-updated date from its most recent commit, and in a shallow clone pages whose last change is older than the clone fall back to the checkout time. See [Last-updated strategy](/reference/configuration#last-updated-strategy).

3. Commit the workflow and push to `main`.

→ The **Actions** tab shows the `Deploy to GitHub Pages` workflow. When both jobs finish, the `deploy` job links to the published site. Every later push to `main` redeploys.

The action accepts these inputs under `with:`:

| Input | Default | Description |
| --- | --- | --- |
| `path` | `.` | Path to the Sarde project root, where `sarde.yaml` lives. Set it when the site is in a subdirectory, for example `docs`. |
| `version` | `latest` | Sarde version to install, for example `v1.0.0`. |
| `build-args` | `""` | Extra arguments passed to `sarde build`. |
| `upload-pages-artifact` | `true` | Upload `dist/` as the GitHub Pages artifact. |

## Deploy with `sarde deploy`

The built-in `github` provider pushes `dist/` to a branch that GitHub Pages serves. Use it to publish from your own machine.

1. Add the deploy settings to `sarde.yaml`:

   ```yaml title="sarde.yaml"
   deploy:
     provider: github
     branch: gh-pages
   ```

   `branch` defaults to `gh-pages` when omitted.

2. Build and deploy:

   ```sh
   sarde build
   sarde deploy
   ```

   → The terminal prints:

   ```text
   Deploying with github-pages...
     Collecting files
     Pushing 42 files to git@github.com:you/site.git (gh-pages)
   Deploy complete (6s).
   ```

   Output from git itself appears indented under the `Pushing` line.

3. After the first deploy, open **Settings > Pages** in the repository, set **Source** to **Deploy from a branch**, and select `gh-pages`.

The deployer copies `dist/` into a temporary git repository, adds an empty `.nojekyll` file, commits, and force-pushes the commit to the branch on the `origin` remote. Each deploy replaces the branch contents and history. The `.nojekyll` file stops GitHub Pages from running Jekyll, which would drop any file or directory whose name starts with an underscore. The build's lock file (`.sarde.lock`) is never published.

:::note
The `github` provider requires a git remote named `origin` that you can push to, using the git credentials already configured on your machine. If git has no user identity configured, the deploy commit uses the name `sarde-deploy`.
:::

To confirm the remote is reachable before the first deploy, run `sarde deploy --check`. It reads the remote with `git ls-remote` and pushes nothing, so it does not prove write access.

## Use a custom domain

1. Configure DNS for the domain as described in GitHub's [custom domain documentation](https://docs.github.com/en/pages/configuring-a-custom-domain-for-your-github-pages-site).

2. Tell GitHub Pages about the domain:

   - **GitHub Actions:** enter the domain under **Settings > Pages > Custom domain**. GitHub ignores a `CNAME` file in sites published by a workflow.
   - **`sarde deploy`:** set `deploy.cname`. Every deploy replaces the branch, so the deployer writes the `CNAME` file each time:

     ```yaml title="sarde.yaml"
     deploy:
       provider: github
       cname: docs.example.com
     ```

     Without `cname`, a `CNAME` file already in the output (for example one placed in `public/`) is kept as is.

3. Serve from the domain root: set `site.url` to the domain and remove `base_path`:

   ```yaml title="sarde.yaml"
   site:
     url: https://docs.example.com
   ```

See [Deployment](/deployment/) for other hosts and the full list of deploy commands.
