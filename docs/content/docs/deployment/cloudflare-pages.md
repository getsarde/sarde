---
title: Cloudflare Pages
description: "Deploy a Sarde site to Cloudflare Pages from Git, with sarde deploy, or with Wrangler"
sidebar:
  order: 3
---

This page publishes a Sarde site to Cloudflare Pages, either by letting Cloudflare build from your Git repository or by uploading a local build.

## Project configuration

Cloudflare Pages serves the site from the domain root, so leave `build.base_path` unset. Set `site.url` to the production domain so canonical links, the sitemap, and feeds point to it:

```yaml title="sarde.yaml"
site:
  url: https://my-site.pages.dev
```

## Deploy from Git

Cloudflare clones the repository, installs Sarde, and builds on every push. The Cloudflare build image does not include Sarde, so the build command installs it first.

1. In the Cloudflare dashboard, open **Workers & Pages > Create > Pages > Connect to Git** and select the repository.

2. Under **Build settings**, enter:

   | Field | Value |
   | --- | --- |
   | Framework preset | None |
   | Build command | `curl -sSfL https://raw.githubusercontent.com/getsarde/sarde/main/install.sh \| sh && export PATH="$HOME/.local/bin:$PATH" && sarde build` |
   | Build output directory | `dist` |

   The install script puts Sarde in `~/.local/bin` when it cannot write to `/usr/local/bin`, so the command adds that directory to `PATH`. If the site lives in a subdirectory of the repository, set **Root directory** to it.

3. Choose **Save and Deploy**.

→ Cloudflare builds and publishes the site. Every later push to the production branch redeploys, and other branches get preview deployments.

Sarde reads each page's last-updated date from git history. If Cloudflare's clone is shallow, pages whose last change is older than the clone fall back to the build time. See [Last-updated strategy](/reference/configuration#last-updated-strategy).

## Deploy with `sarde deploy`

`sarde deploy` uploads your local `dist/` with the same direct-upload protocol Wrangler uses.

1. Create the project once in the dashboard under **Workers & Pages > Create > Pages > Direct Upload**. Sarde deploys to an existing project and does not create one.

2. Create an API token with the **Cloudflare Pages: Edit** permission under **My Profile > API Tokens**. The account ID is shown on the account home page.

3. Add the project and account to `sarde.yaml`:

   ```yaml title="sarde.yaml"
   deploy:
     provider: cloudflare
     project_name: my-site
     account_id: 0123456789abcdef0123456789abcdef
   ```

4. Set the token, then build and deploy:

   ```sh
   export CLOUDFLARE_API_TOKEN=your-token
   sarde build
   sarde deploy
   ```

→ The output ends with the live URL and the deploy ID. The deploy goes to the project's production branch.

The build's `_redirects` file, and a `_headers` file if you add one under `public/`, are sent with the deployment. `sarde deploy` does not support Pages Functions: if the output contains `_worker.js`, it stops with an error; use Wrangler for those sites. A `_routes.json` file only applies to Functions, so it is skipped with a warning.

Cloudflare Pages accepts files up to 25 MiB and a limited number of files per deployment (20,000 unless your plan allows more). `sarde deploy` checks both limits before uploading and stops with an error that names the file or the count.

See [Deploy tokens](/deployment/#deploy-tokens) to store the token in CI and [`--check`](/deployment/#check-access-before-the-first-deploy) to test it before the first deploy. For Cloudflare, `--check` also requests an upload token, so it confirms write access.

## Deploy with Wrangler

Build locally, then upload `dist/` with Wrangler:

```sh
sarde build
npx wrangler pages deploy dist --project-name my-site
```

See [Deployment](/deployment/) for other hosts.
