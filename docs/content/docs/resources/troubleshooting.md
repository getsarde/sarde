---
title: Troubleshooting
sidebar:
  order: 4
---

Common errors and how to resolve them. Errors are grouped by where they occur: build, configuration, dev server, link checking, and themes.

## Build errors

### "discovering content" failure

```
Error: discovering content: ...
```

The `content/` directory is missing or empty. Sarde requires at least one file in `content/` to build.

**Fix:** Create the directory and add a Markdown file:

```bash
mkdir -p content/docs
echo "# Hello" > content/docs/getting-started.md
```

### "rendering markdown for ..." failure

```
Error: rendering markdown for content/docs/page.md: ...
```

The Markdown file contains a syntax error that the parser cannot recover from, or a shortcode references an undefined name.

**Fix:** Check the file for unclosed shortcode tags, malformed fenced code blocks (mismatched backtick counts), or unclosed extension directives (`:::` without a matching close).

### "rendering ... (template ...)" failure

```
Error: rendering content/docs/page.md (template docs/single.html): ...
```

A Go template in the layout directory has a syntax error, or the template references a function or variable that does not exist.

**Fix:** Check the template file indicated in the error. Common causes:
- Missing closing `{{ end }}` for `{{ if }}`, `{{ range }}`, or `{{ with }}` blocks
- Calling an undefined function (typo in function name)
- Accessing a field that does not exist on the data object

### "loading templates" failure

```
Error: loading templates: ...
```

A template file in `layouts/` has a parse error. Go's template parser reports the file and line.

**Fix:** Read the error message for the file path and line number. Fix the template syntax at that location.

### "building collections" failure

```
Error: building collections: ...
```

A collection's configuration is invalid, or a frontmatter schema validation failed.

**Fix:** Check `sarde.yaml` for invalid collection settings. If the error mentions a specific page, check that page's frontmatter.

## Configuration errors

### Unknown keys

The config validator warns about unrecognized keys in `sarde.yaml`. These appear as warnings, not errors, and do not stop the build.

**Fix:** Check for typos in key names. Run `sarde validate` to see all config warnings.

### Invalid YAML syntax

```
Error: parsing sarde.yaml: ...
```

The YAML file has a syntax error (bad indentation, missing colons, tabs instead of spaces).

**Fix:** YAML requires spaces for indentation, not tabs. Use a YAML linter or check the line number in the error message.

### Range validation

```
Error: server.port: value -1 is below minimum 1
Error: images.quality: value 200 is above maximum 100
```

A numeric value is outside its valid range.

**Fix:** Correct the value. Run `sarde validate` to check all fields at once. Common ranges:
- `server.port`: 1 to 65535
- `images.quality`: 1 to 100
- `toc.min_level` / `toc.max_level`: 1 to 6

### Unknown plugin name

```
Warning: unknown plugin "my-plugin" in plugins.enabled
```

A name in `plugins.enabled` does not match any built-in or client-side plugin.

**Fix:** Check the plugin name for typos. Run `sarde validate` to see all valid plugin names.

## Dev server errors

### Port in use

```
Port 4727 in use, trying 4728...
```

Another process is using the default port. The dev server automatically tries up to 10 consecutive ports.

```
Error: could not bind to any port in range 4727-4736
```

All 10 ports are in use.

**Fix:** Stop the other process using the port, or specify a different port:

```bash
sarde dev --port 5000
```

### File watcher failures

```
Error: starting file watcher: ...
```

The operating system's file watcher could not be initialized (e.g., too many open files on Linux).

**Fix:** On Linux, increase the inotify watch limit:

```bash
echo fs.inotify.max_user_watches=524288 | sudo tee -a /etc/sysctl.conf
sudo sysctl -p
```

### WebSocket disconnects

The browser's live reload indicator shows a disconnected state. This happens when the dev server restarts or the network connection is interrupted.

**Fix:** Refresh the browser page. The dev server's WebSocket uses ping/pong keepalive (30-second interval) and auto-reconnects on most transient failures.

## Link checker errors

### Broken internal links

```
invalid link: /docs/nonexistent-page/
```

A Markdown link points to a page that does not exist in the build output.

**Fix:** Correct the link target. Use `sarde check-links` to find all broken links at once.

### Broken anchors

```
invalid hash: /docs/page/#nonexistent-heading
```

A link's `#anchor` does not match any heading ID on the target page.

**Fix:** Check the target page's headings. Heading IDs are generated from the heading text (slugified). Custom IDs use the `{#custom-id}` syntax.

### External link timeouts

External link checking is opt-in (`link_validation.external.check: true`). Timeouts occur when the target server does not respond within the configured timeout (default: 10 seconds).

**Fix:** Add slow or unreliable domains to the ignore list:

```yaml
link_validation:
  external:
    ignore:
      - "https://slow-site.example.com/*"
```

## Theme errors

### Unknown preset

```
Warning: unknown theme preset "custom"
```

The `theme.preset` value does not match any built-in preset.

**Fix:** Use one of the available presets: `default`, `ocean`, `forest`, `rose`, `clean`, `minimal`, `docs`, `academic`.

### Invalid color values

```
Error: invalid hex color: #xyz
```

A color value in `theme.overrides` or a plugin config is not a valid hex color.

**Fix:** Use `#rgb` (3 digits) or `#rrggbb` (6 digits) format, e.g., `#e94560`.

### Theme eject issues

`sarde theme eject` copies the embedded theme files to `themes/default/` for customization. If the directory already exists, the command does not overwrite existing files.

**Fix:** To re-eject, remove or rename the existing `themes/default/` directory first.

## Getting more information

Run `sarde build --verbose` or `sarde dev --verbose` for detailed build output, including timing information for each pipeline phase.

Run `sarde validate` to check `sarde.yaml` for errors and warnings without building the site.

Run `sarde check-links` for a dedicated link validation pass with a detailed report.
