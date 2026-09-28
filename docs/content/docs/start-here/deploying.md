---
title: Deploying
description: "Build a production site and pick a guide to deploy it to GitHub Pages, Netlify, Cloudflare Pages, Vercel, or any static host"
sidebar:
  order: 4
---

Sarde builds a static site that works on any hosting provider. Build it once, then follow the guide for your host.

## Build for production

```sh
sarde build
```

→ The terminal prints a summary:

```text
Built in 320 ms
  Output: /path/to/my-site/dist
```

The `dist/` directory contains the complete site: HTML pages, CSS, JavaScript, images, feeds, sitemap, and search index.

## Pick your host

- [GitHub Pages](/deployment/github-pages/): publish with GitHub Actions or push to a `gh-pages` branch.
- [Netlify](/deployment/netlify/): build from Git, deploy with `sarde deploy`, or use the Netlify CLI.
- [Cloudflare Pages](/deployment/cloudflare-pages/): build from Git, deploy with `sarde deploy`, or use Wrangler.
- [Vercel](/deployment/vercel/): build from Git, deploy with `sarde deploy`, or use the Vercel CLI.
- [Other hosts](/deployment/other-hosts/): upload `dist/` anywhere, or run your own command with `sarde deploy`.

The [Deployment](/deployment/) section compares the deployment methods and explains where deploy tokens go.
