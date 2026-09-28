---
title: Deployment
description: "Deploy a Sarde site to GitHub Pages, Netlify, Cloudflare Pages, Vercel, or any static host"
sidebar:
  order: 7
  icon: cloud-upload
---

A Sarde site builds to plain HTML, CSS, and JavaScript, so any static host can serve it. Pick your host below for step-by-step instructions.

::::card-grid(cols=2)
:::link-card[GitHub Pages](href="/deployment/github-pages/" icon="github" description="Publish with GitHub Actions or push to a gh-pages branch.")
:::
:::link-card[Netlify](href="/deployment/netlify/" icon="globe" description="Build from Git, deploy with sarde deploy, or use the Netlify CLI.")
:::
:::link-card[Cloudflare Pages](href="/deployment/cloudflare-pages/" icon="cloud" description="Build from Git, deploy with sarde deploy, or use Wrangler.")
:::
:::link-card[Vercel](href="/deployment/vercel/" icon="triangle" description="Build from Git, deploy with sarde deploy, or use the Vercel CLI.")
:::
:::link-card[Other hosts](href="/deployment/other-hosts/" icon="server" description="Upload dist/ anywhere, or run your own command with sarde deploy.")
:::
::::

## Build for production

Every deployment method publishes the output of `sarde build`:

```sh
sarde build
```

→ The terminal prints a summary:

```text
Built in 320 ms
  Output: /path/to/my-site/dist
```

The `dist/` directory contains the complete site: HTML pages, CSS, JavaScript, images, feeds, sitemap, and search index. Change the directory with `--output` or [`build.output`](/reference/configuration#build) in `sarde.yaml`.

## Deployment methods

Each host page covers up to three methods:

| Method | How it works | Use it when |
| --- | --- | --- |
| Deploy from Git | The host clones your repository, installs Sarde, and builds on every push. | You want every push to `main` to publish automatically. |
| `sarde deploy` | Sarde uploads your local `dist/` to the host through its API. | You build on your own machine or in your own CI job. |
| Host CLI | The host's own tool (`netlify`, `wrangler`, `vercel`) uploads `dist/`. | You already use that tool, or `sarde deploy` does not support your setup. |

## Deploy tokens

`sarde deploy` reads API tokens from environment variables. Tokens never go in `sarde.yaml`, because that file is usually committed.

| Provider | Token variable | Other settings |
| --- | --- | --- |
| Netlify | `NETLIFY_AUTH_TOKEN` | `site_id` |
| Cloudflare Pages | `CLOUDFLARE_API_TOKEN` | `project_name`, `account_id` (or `CLOUDFLARE_ACCOUNT_ID`) |
| Vercel | `VERCEL_TOKEN` | `project_id`, `team_id` for team projects (or `VERCEL_ORG_ID`) |
| GitHub Pages | None | Uses the git credentials already configured for the `origin` remote |

In CI, store the token as a repository secret and expose it to the deploy step under the variable name above.

## Check access before the first deploy

`--check` confirms that the configured site or project exists and that Sarde can reach it. It uploads nothing:

```sh
sarde deploy --check
```

→ The terminal prints:

```text
Credentials OK: netlify can deploy to docs (https://docs.example.com)
```

On later deploys, Sarde uploads only the files the host does not already have, so a redeploy after a small change sends a few files, not the whole site.

:::caution
The Netlify, Cloudflare Pages and Vercel deployers talk to each provider's API directly and are new in this release. If a deploy fails for your site, the host's own CLI still works, either directly or wrapped in a [`custom` provider](/deployment/other-hosts/). Please report the error message so the deployer can be fixed.
:::

See [`deploy`](/reference/cli-commands#deploy) in CLI Commands for all flags and the `--format json` event stream, and [`deploy`](/reference/configuration#deploy) in Configuration for all options.
