---
title: Deploy and Redirects
description: "Deploy provider and redirect settings in sarde.yaml"
sidebar:
  order: 7
---

Configure `sarde deploy` and the redirects the build writes. For step-by-step host guides, see [Deployment](/deployment/).

## `deploy`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `provider` | string | `""` | Deployment provider. `github`, `netlify`, `cloudflare`, `vercel`, or `custom`. |
| `branch` | string | `"gh-pages"` | Branch the `github` provider pushes the built site to. |
| `cname` | string | `""` | Custom domain the `github` provider writes to the published `CNAME` file, such as `docs.example.com`. A bare domain: no scheme, path or port. |
| `site_id` | string | `""` | Netlify site ID. |
| `project_name` | string | `""` | Cloudflare Pages project name. |
| `account_id` | string | `""` | Cloudflare account ID. The `CLOUDFLARE_ACCOUNT_ID` environment variable overrides it. |
| `project_id` | string | `""` | Vercel project ID or name. |
| `team_id` | string | `""` | Vercel team ID (`team_...`) for a project owned by a team. The `VERCEL_ORG_ID` environment variable overrides it. |
| `command` | string | `""` | Custom deployment command (for `custom` provider). |
| `redirect_format` | string | `""` | Redirect file format. `html`, `netlify`, `vercel`, or `all`. |

API tokens are never read from `sarde.yaml`. Each provider takes its token from an environment variable:

| Provider | Token variable |
| --- | --- |
| `netlify` | `NETLIFY_AUTH_TOKEN` |
| `cloudflare` | `CLOUDFLARE_API_TOKEN` |
| `vercel` | `VERCEL_TOKEN` |
| `github` | None: uses the git credentials configured for the `origin` remote |

See [Deployment](/deployment/) for each platform.

## `redirects`

A map of source path to destination URL. Higher cascade layers add entries without removing existing ones.

```yaml
redirects:
  /old-page: /new-page
  /legacy/docs: /docs/start-here/getting-started
```
