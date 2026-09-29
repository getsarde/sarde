---
title: Netlify
description: "Deploy a Sarde site to Netlify from Git, with sarde deploy, or with the Netlify CLI"
sidebar:
  order: 2
---

This page publishes a Sarde site to Netlify, either by letting Netlify build from your Git repository or by uploading a local build.

## Project configuration

Netlify serves the site from the domain root, so leave `build.base_path` unset. Set `site.url` to the production domain so canonical links, the sitemap, and feeds point to it:

```yaml title="sarde.yaml"
site:
  url: https://my-site.netlify.app
```

## Deploy from Git

Netlify clones the repository, installs Sarde, and builds on every push. Netlify's build image does not include Sarde, so the build command installs it first.

1. Create `netlify.toml` at the repository root:

   ```toml title="netlify.toml"
   [build]
     command = 'curl -sSfL https://raw.githubusercontent.com/getsarde/sarde/main/install.sh | sh && export PATH="$HOME/.local/bin:$PATH" && sarde build'
     publish = "dist"
   ```

   The install script puts Sarde in `~/.local/bin` when it cannot write to `/usr/local/bin`, so the command adds that directory to `PATH`.

2. Commit `netlify.toml` and push.

3. In the Netlify dashboard, choose **Add new site > Import an existing project** and select the repository. Netlify reads the build command and publish directory from `netlify.toml`.

→ Netlify builds and publishes the site. Every later push to the production branch redeploys, and pull requests get deploy previews.

Sarde reads each page's last-updated date from git history. If Netlify's clone is shallow, pages whose last change is older than the clone fall back to the build time. See [Last-updated strategy](/reference/configuration/build-and-output/#last-updated-strategy).

## Deploy with `sarde deploy`

`sarde deploy` uploads your local `dist/` through the Netlify API. The site must already exist on Netlify.

1. Create a personal access token under **User settings > Applications > Personal access tokens**.

2. Find the site ID under **Site configuration > General > Site details** (**Site ID**) and add it to `sarde.yaml`:

   ```yaml title="sarde.yaml"
   deploy:
     provider: netlify
     site_id: 3f2a1b4c-5d6e-7f80-9a1b-2c3d4e5f6a7b
   ```

3. Set the token, then build and deploy:

   ```sh
   export NETLIFY_AUTH_TOKEN=your-token
   sarde build
   sarde deploy
   ```

→ The output ends with the live URL and the deploy ID:

```text
Deploy complete: https://my-site.netlify.app (8s)
Deploy ID: 65f1c2d3e4a5b6c7d8e9f0a1
```

The deploy goes to production. If the site has auto publishing locked in the Netlify dashboard, Netlify keeps serving the previous deploy and `sarde deploy` prints a warning; publish the new deploy from the dashboard.

`sarde deploy` stops with an error if a file name in the output contains `#` or `?`, which Netlify cannot serve.

See [Deploy tokens](/deployment/#deploy-tokens) to store the token in CI and [`--check`](/deployment/#check-access-before-the-first-deploy) to test it before the first deploy.

## Deploy with the Netlify CLI

Build locally, then upload `dist/` with the Netlify CLI:

```sh
sarde build
npx netlify deploy --prod --dir dist
```

The first run asks you to log in and link the directory to a site.

## Redirects

The build writes a `_redirects` file for configured redirects and page aliases, and Netlify reads it with every method above. Sarde also writes an HTML redirect page at each old path. Netlify serves an existing file in place of a `_redirects` rule for the same path, so visitors are redirected by that page rather than by a 301 response. See [Redirects](/plugins/redirects/).

See [Deployment](/deployment/) for other hosts.
