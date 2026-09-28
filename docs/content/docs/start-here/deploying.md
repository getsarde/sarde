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
  Collecting files
  Pushing 42 files to git@github.com:you/site.git (gh-pages)
Deploy complete (6s).
```

Output from git itself appears indented under the `Pushing` line.

The deployer copies `dist/` into a temporary git repository, adds an empty `.nojekyll` file, commits, and force-pushes the commit to the `gh-pages` branch of the `origin` remote. Each deploy replaces the branch contents and history. The `.nojekyll` file stops GitHub Pages from running Jekyll, which would drop any file or directory whose name starts with an underscore. The build's lock file (`.sarde.lock`) is never published.

After the first deploy, open **Settings > Pages** in the repository, set **Source** to **Deploy from a branch**, and select `gh-pages`.

#### Custom domain

GitHub Pages reads the custom domain from a `CNAME` file on the published branch. Because every deploy replaces the branch, set the domain in `sarde.yaml` so each deploy writes the file:

```yaml title="sarde.yaml"
deploy:
  provider: github
  cname: docs.example.com
```

Without `cname`, a `CNAME` file already in the output (for example one placed in `static/`) is kept as is.

:::note
The `github` provider requires a git remote named `origin` that you can push to, using the git credentials already configured on your machine. If git has no user identity configured, the deploy commit uses the name `sarde-deploy`.
:::

To confirm the remote is reachable before the first deploy, run `sarde deploy --check`. It reads the remote with `git ls-remote` and pushes nothing. It does not prove write access.

See [`deploy`](/reference/cli-commands#deploy) in CLI Commands for all flags, and [`deploy`](/reference/configuration#deploy) in Configuration for all options.

## Deploy tokens

Netlify, Cloudflare Pages and Vercel deploys use an API token that Sarde reads from an environment variable. Tokens never go in `sarde.yaml`, because that file is usually committed.

| Provider | Token variable | Other settings |
| --- | --- | --- |
| Netlify | `NETLIFY_AUTH_TOKEN` | `site_id` |
| Cloudflare Pages | `CLOUDFLARE_API_TOKEN` | `project_name`, `account_id` (or `CLOUDFLARE_ACCOUNT_ID`) |
| Vercel | `VERCEL_TOKEN` | `project_id`, `team_id` for team projects (or `VERCEL_ORG_ID`) |

Check a token before the first deploy. `--check` looks up the configured site or project and uploads nothing:

```sh
sarde deploy --check
```

→ The terminal prints:

```text
Credentials OK: netlify can deploy to docs (https://docs.example.com)
```

In CI, store the token as a repository secret and expose it to the deploy step as the variable above.

Sarde uploads only the files the provider does not already have, so a redeploy after a small change sends a few files, not the whole site.

:::caution
The Netlify, Cloudflare Pages and Vercel deployers talk to each provider's API directly and are new in this release. If a deploy fails for your site, the provider CLI commands in each section below still work, wrapped in a [`custom` provider](#custom-deployment). Please report the error message so the deployer can be fixed.
:::

## Netlify

Create a personal access token under **User settings > Applications > Personal access tokens** and find the site ID under **Site configuration > General > Site details** (**Site ID**).

```yaml title="sarde.yaml"
deploy:
  provider: netlify
  site_id: 3f2a1b4c-5d6e-7f80-9a1b-2c3d4e5f6a7b
```

```sh
export NETLIFY_AUTH_TOKEN=your-token
sarde build
sarde deploy
```

The deploy goes to production. If the site has auto publishing locked in the Netlify dashboard, Netlify keeps serving the previous deploy and `sarde deploy` prints a warning; publish the new deploy from the dashboard.

`sarde deploy` stops with an error if a file name in the output contains `#` or `?`, which Netlify cannot serve.

Netlify reads the `_redirects` file the build writes for configured redirects and page aliases. Sarde also writes an HTML redirect page at each old path, and on Netlify that page takes precedence over the `_redirects` rule, so visitors are redirected by the page rather than by a 301 response.

Without `sarde deploy`, the Netlify CLI works too:

```sh
sarde build
npx netlify deploy --prod --dir dist
```

## Cloudflare Pages

Create an API token with the **Cloudflare Pages: Edit** permission under **My Profile > API Tokens**. The account ID is shown on the account home page. The project must already exist; create it once in the dashboard under **Workers & Pages > Create > Pages > Direct Upload**.

```yaml title="sarde.yaml"
deploy:
  provider: cloudflare
  project_name: my-site
  account_id: 0123456789abcdef0123456789abcdef
```

```sh
export CLOUDFLARE_API_TOKEN=your-token
sarde build
sarde deploy
```

The deploy goes to the project's production branch. The build's `_redirects` file, and a `_headers` file if you add one under `static/`, are sent with the deployment. Pages Functions (a `_worker.js` in the output) are not supported by `sarde deploy`; use Wrangler for those sites. A `_routes.json` file only applies to Functions, so it is skipped with a warning.

Cloudflare Pages accepts files up to 25 MiB and a limited number of files per deployment (20,000 unless your plan allows more). `sarde deploy` checks both limits before uploading and stops with an error that names the file or the count.

Without `sarde deploy`, Wrangler works too:

```sh
sarde build
npx wrangler pages deploy dist --project-name my-site
```

Alternatively, connect the Git repository in the Cloudflare dashboard. Set the build command to `sarde build` and the output directory to `dist`.

## Vercel

Create a token under **Account Settings > Tokens**. The project ID is under **Project Settings > General** (**Project ID**); the project name works too. For a project owned by a team, also set the team ID (`team_...`) from **Team Settings > General**.

```yaml title="sarde.yaml"
deploy:
  provider: vercel
  project_id: prj_AbCdEf123456
  team_id: team_AbCdEf123456
```

```sh
export VERCEL_TOKEN=your-token
sarde build
sarde deploy
```

Sarde uploads the site as prebuilt output, so Vercel serves the files as they are and never runs a build of its own. Redirects from the build's `vercel.json` become Vercel routes that answer with status 308 (permanent), and a `404.html` in the output is served for missing pages. The printed URL is the production domain.

A deployment holds at most 15,000 files. `sarde deploy` stops with an error before uploading a larger site.

Without `sarde deploy`, the Vercel CLI works too:

```sh
sarde build
npx vercel deploy --prod dist
```

To generate only a Vercel-compatible `vercel.json`, set the redirect format:

```yaml
deploy:
  redirect_format: vercel
```

Alternatively, connect the Git repository in the Vercel dashboard. Set the build command to `sarde build` and the output directory to `dist`.

## Custom deployment

The `custom` provider runs any shell command with the `DIST_DIR` environment variable set to the absolute path of the output directory. The command runs in the site root, through `sh` on macOS and Linux and through `cmd.exe` on Windows.

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

On Windows, `cmd.exe` expands `%DIST_DIR%`, not `$DIST_DIR`, so write the command for that shell:

```yaml
deploy:
  provider: custom
  command: "npx netlify deploy --prod --dir %DIST_DIR%"
```
