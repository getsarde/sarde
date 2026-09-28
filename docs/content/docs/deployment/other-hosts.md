---
title: Other Hosts
description: "Deploy a Sarde site to any static host, or run your own deploy command with the custom provider"
sidebar:
  order: 5
---

This page publishes a Sarde site to a host without a dedicated page, such as a VPS, an object storage bucket, or another static hosting service.

## Upload `dist/`

The output of `sarde build` needs no server-side runtime. Upload the contents of `dist/` to any web server or static host and serve it as-is.

If the host serves the site from a subdirectory, for example `https://example.com/docs/`, set `build.base_path` to that path and `site.url` to the host:

```yaml title="sarde.yaml"
site:
  url: https://example.com
build:
  base_path: /docs/
```

## Deploy with a custom command

The `custom` provider lets `sarde deploy` run any shell command. The `DIST_DIR` environment variable holds the absolute path of the output directory, and the command runs in the site root.

1. Add the command to `sarde.yaml`:

   ```yaml title="sarde.yaml"
   deploy:
     provider: custom
     command: "rsync -avz $DIST_DIR/ user@server:/var/www/html/"
   ```

2. Build and deploy:

   ```sh
   sarde build
   sarde deploy
   ```

→ The command's output appears indented under `Running the deploy command`, and `sarde deploy` exits with an error if the command fails.

The same provider wraps any host CLI, for example Netlify's:

```yaml title="sarde.yaml"
deploy:
  provider: custom
  command: "npx netlify deploy --prod --dir $DIST_DIR"
```

### Windows

The command runs through `sh` on macOS and Linux and through `cmd.exe` on Windows. `cmd.exe` expands `%DIST_DIR%`, not `$DIST_DIR`, so write the command for that shell:

```yaml title="sarde.yaml"
deploy:
  provider: custom
  command: "npx netlify deploy --prod --dir %DIST_DIR%"
```

`--check` is not available for the `custom` provider.

See [Deployment](/deployment/) for hosts with dedicated guides.
