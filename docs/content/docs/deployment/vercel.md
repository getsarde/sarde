---
title: Vercel
description: "Deploy a Sarde site to Vercel from Git, with sarde deploy, or with the Vercel CLI"
sidebar:
  order: 4
---

This page publishes a Sarde site to Vercel, either by letting Vercel build from your Git repository or by uploading a local build.

## Project configuration

Vercel serves the site from the domain root, so leave `build.base_path` unset. Set `site.url` to the production domain so canonical links, the sitemap, and feeds point to it:

```yaml title="sarde.yaml"
site:
  url: https://my-site.vercel.app
```

## Deploy from Git

Vercel clones the repository, installs Sarde, and builds on every push. Vercel's build image does not include Sarde, so the build command installs it first.

1. Create `vercel.json` at the repository root:

   ```json title="vercel.json"
   {
     "framework": null,
     "buildCommand": "curl -sSfL https://raw.githubusercontent.com/getsarde/sarde/main/install.sh | sh && export PATH=\"$HOME/.local/bin:$PATH\" && sarde build",
     "outputDirectory": "dist"
   }
   ```

   The install script puts Sarde in `~/.local/bin` when it cannot write to `/usr/local/bin`, so the command adds that directory to `PATH`.

2. Commit `vercel.json` and push.

3. In the Vercel dashboard, choose **Add New > Project** and import the repository. Vercel reads the build settings from `vercel.json`. If the site lives in a subdirectory of the repository, set **Root Directory** to it and put `vercel.json` there.

→ Vercel builds and publishes the site. Every later push to the production branch redeploys, and other branches get preview deployments.

Sarde reads each page's last-updated date from git history. If Vercel's clone is shallow, pages whose last change is older than the clone fall back to the build time. See [Last-updated strategy](/reference/configuration/build-and-output/#last-updated-strategy).

:::caution
With Git deploys, Vercel reads redirects only from the `vercel.json` at the project root, not the one the build writes into `dist/`. The HTML redirect pages still send visitors to the new URL. See [Redirects](#redirects).
:::

## Deploy with `sarde deploy`

`sarde deploy` uploads your local `dist/` through the Vercel API. The project must already exist on Vercel.

1. Create a token under **Account Settings > Tokens**.

2. Find the project ID under **Project Settings > General** (**Project ID**); the project name works too. For a project owned by a team, also copy the team ID (`team_...`) from **Team Settings > General**. Add both to `sarde.yaml`:

   ```yaml title="sarde.yaml"
   deploy:
     provider: vercel
     project_id: prj_AbCdEf123456
     team_id: team_AbCdEf123456
   ```

   Leave out `team_id` for a project in your personal account.

3. Set the token, then build and deploy:

   ```sh
   export VERCEL_TOKEN=your-token
   sarde build
   sarde deploy
   ```

→ The output ends with the production URL and the deploy ID. The deploy goes to production.

Sarde uploads the site as prebuilt output, so Vercel serves the files as they are and never runs a build of its own. A `404.html` in the output is served for missing pages. A deployment holds at most 15,000 files; `sarde deploy` stops with an error before uploading a larger site.

See [Deploy tokens](/deployment/#deploy-tokens) to store the token in CI and [`--check`](/deployment/#check-access-before-the-first-deploy) to test it before the first deploy.

## Deploy with the Vercel CLI

Build locally, then upload `dist/` with the Vercel CLI:

```sh
sarde build
npx vercel deploy --prod dist
```

The first run asks you to log in and link or create a project.

## Redirects

The build writes a `vercel.json` into `dist/` with the configured redirects and page aliases, plus an HTML redirect page at each old path. How Vercel uses them depends on the method:

| Method | Redirects |
| --- | --- |
| `sarde deploy` | The redirects become Vercel routes that answer with status 308 (permanent). |
| Vercel CLI | `dist/` is the uploaded project root, so Vercel reads `dist/vercel.json`. |
| Deploy from Git | Vercel reads only the root `vercel.json`. The HTML redirect pages handle the old paths. |

To write only the Vercel file and skip Netlify's `_redirects`, set the redirect format:

```yaml title="sarde.yaml"
deploy:
  redirect_format: vercel
```

See [Redirects](/plugins/redirects/) for the redirect sources and formats, and [Deployment](/deployment/) for other hosts.
