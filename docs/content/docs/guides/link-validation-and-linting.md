---
title: Link Validation and Linting
description: "Configure internal link checking, external URL probing, and content linting policies"
sidebar:
  order: 20
---

Every `sarde build` checks the internal links in your content, and `sarde check-links` runs the same check without building. Optionally, Sarde also probes external URLs. A separate content linter flags Markdown quality problems such as skipped heading levels and missing image alt text. Each link problem has a policy that decides whether it fails the build, logs a warning, or is ignored.

## Link checker overview

The checker validates internal links and `#anchor` references to headings. It resolves each link inside its own lane, the language and version of the page that contains it: a link in French v2 docs resolves against French v2 pages. Use the `?lang=` and `?version=` query parameters for links that cross lanes (see [Internal Links](/guides/internal-links/)).

No configuration is needed. The build prints a summary line that counts the findings by policy, followed by one line per finding with its `path:line:col` location:

```text
[links] checked 12 links across 2 lanes: 1 error
  ERROR  content/docs/guide.md:7:3  broken target    ./missing.md
```

A run without findings prints `no issues` after the lane count. Identical findings in one file appear once, followed by a count such as `(x3)`.

Each finding has one of these types:

| Type | Meaning | Policy key |
|------|---------|------------|
| Broken target | A link to a page that does not exist, such as `./missing.md` | `on_broken` |
| Broken anchor | A link to a heading ID that does not exist on the target page | `on_broken_anchor` |
| Ambiguous link | A bare `name.md` destination, which could be a sibling or a content-root file | `on_broken` |
| Relative link | A destination that starts with `./` or `../` | `on_relative_links` |
| Local link | A `localhost` or `127.0.0.1` URL | `on_local_links` |
| Unverified internal | An extension-less internal link, such as `/docs/nope/`, that did not resolve in its lane | `on_unverified_internal` |
| Same site | A link written with the site's own absolute URL | `same_site_policy` |

Sarde never guesses what a bare `name.md` means. It reports the link as an ambiguous link, and the finding's `hint` gives the fix: write `./name.md` for a sibling page or `docs/name.md` from the content root. Every finding carries the 1-based line and column of the link in its source file: the `pretty` report shows them as `path:line:col`, and the `json` report as `line` and `col`.

Image sources are not checked, and static assets such as `/img/logo.png` and the site root `/` are never reported as unverified.

A full build re-validates every link target and anchor, including on pages served from the page cache, so renaming a linked page is caught without editing the file that links to it.

## Configurable policies

A policy is `"error"` (fails the build), `"warn"` (logs a warning), or `"ignore"` (skips the finding). Defaults:

```yaml title="sarde.yaml"
link_validation:
  enabled: true
  on_broken: "error"
  on_broken_anchor: "error"
  on_relative_links: "warn"
  on_local_links: "warn"
  on_unverified_internal: "warn"
```

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `enabled` | bool | `true` | Turn link validation on or off. |
| `on_broken` | string | `"error"` | Policy for broken targets and ambiguous links. |
| `on_broken_anchor` | string | `"error"` | Policy for links to missing heading anchors. |
| `on_relative_links` | string | `"warn"` | Policy for relative links (`./` or `../`). |
| `on_local_links` | string | `"warn"` | Policy for `localhost` and `127.0.0.1` URLs. |
| `on_unverified_internal` | string | `"warn"` | Policy for extension-less internal links that did not resolve in their lane. |
| `same_site_policy` | string | `"ignore"` | Policy for links to the site's own absolute URL. |
| `report` | string | `"pretty"` | Report format: `pretty`, `json`, or `github-actions`. |
| `site_root_escape_prefix` | string | `"site:"` | Prefix that routes a link to the site root and skips lane logic (for example `site:/pricing`). Set to `""` to disable. |
| `exclude` | string[] | `[]` | Link destinations to skip, as glob patterns matched against the destination exactly as written. A `*` does not match `/`. |
| `check_anchors` | bool | `true` | Has no effect on `sarde build` or `sarde check-links`, which always check anchors. |
| `check_images` | bool | `true` | Has no effect on `sarde build` or `sarde check-links`, which do not check image sources. |
| `fail_build` | bool | `false` | Has no effect on `sarde build` or `sarde check-links`. Set the policies above to `"error"` instead. |

## External URL probing

External link checking is off by default. Turn it on to probe `http://` and `https://` URLs for reachability during the build:

```yaml title="sarde.yaml"
link_validation:
  external:
    check: true
    on_broken: "warn"
```

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `external.check` | bool | `false` | Probe external URLs. |
| `external.concurrency` | int | `8` | Maximum concurrent HTTP requests. |
| `external.timeout` | string | `"10s"` | Per-request timeout, in Go duration format. |
| `external.cache` | string | `".sarde/linkcache.json"` | Path of the cache file for results. |
| `external.cache_ttl` | string | `"72h"` | How long a cached result is reused. |
| `external.on_broken` | string | `"warn"` | Policy for unreachable URLs. |
| `external.ignore` | string[] | `[]` | URL glob patterns to skip. A `*` does not match `/`, so `https://example.com/*` skips only one path segment. |
| `external.method` | string | `"head-then-get"` | `"head-then-get"` sends HEAD and retries with GET on a 403 or 405 response. `"head"` and `"get"` use one method. |

A URL counts as reachable when it returns a status from 200 to 399. Results, failures included, are cached in `.sarde/linkcache.json` and reused until `cache_ttl` expires, so a fixed URL is not rechecked until then. Delete the cache file to force a fresh probe.

## `sarde check-links`

Run link validation without rendering templates or writing output:

```bash
sarde check-links
```

→ The command prints the summary line and one line per finding, ordered by source file. It exits with code 1 when any finding has the `error` policy. The report goes to stderr in every format.

| Flag | Default | Description |
|------|---------|-------------|
| `--strict` | `false` | Treat every link issue as an error, including relative, local, unverified, same-site, and external findings. |
| `--external` | `false` | Also probe external URLs. |
| `--report` | `""` | Report format: `pretty`, `json`, or `github-actions`. Empty uses `link_validation.report`. |
| `--base-path` | `""` | Override the URL base path (for example `/docs/`). |
| `--content` | `""` | Override the content directory. |

`sarde check` is an alias for `sarde check-links`.

## Report formats

Choose a format with `--report` or `link_validation.report`:

| Format | Description |
|--------|-------------|
| `pretty` | One line per finding, ordered by source file, with a policy label and a `path:line:col` location. The default. |
| `json` | A `summary` object with the counts and a `findings` array. Each finding includes `file`, `line`, `col`, `dest`, `type`, and `policy`. |
| `github-actions` | The plain-text report, plus a `link_validation_failed` value written to `$GITHUB_OUTPUT` and a Markdown table appended to `$GITHUB_STEP_SUMMARY` when those variables are set. It does not print `::error` annotations. |

Set the default format in `sarde.yaml`:

```yaml title="sarde.yaml"
link_validation:
  report: "github-actions"
```

## Fail CI on link problems

`sarde build` and `sarde check-links` exit with code 1 when any finding has the `error` policy, which blocks a deployment step. The defaults already do this for broken targets and anchors. To fail on every kind of finding, run the stricter check in CI before the build:

```bash
sarde check-links --strict
```

## Content lint rules

The content linter checks Markdown structure independently of link validation. It runs as the `content_lint` plugin during the build and only ever warns, so it never fails a build. Configure it under the top-level `content_lint` key:

```yaml title="sarde.yaml"
content_lint:
  enabled: true
  rules:
    heading_max_length: 60
    heading_increment: true
    image_alt_required: true
    no_empty_links: true
    frontmatter_required: []
    tabs_marker_syntax: true
```

| Rule | Type | Default | Description |
|------|------|---------|-------------|
| `heading_max_length` | int | `60` | Maximum heading text length. `0` disables the rule. |
| `heading_increment` | bool | `true` | Warn when heading levels skip, for example `#` followed by `####`. |
| `image_alt_required` | bool | `true` | Warn on images with empty alt text (`![](...)`). |
| `no_empty_links` | bool | `true` | Warn on links with empty text (`[](...)`). |
| `frontmatter_required` | string[] | `[]` | Frontmatter fields every page must have, for example `["title", "description"]`. |
| `tabs_marker_syntax` | bool | `true` | Warn on `:::tabs` blocks whose `== Label` markers are malformed or missing. |

Each warning names the file and the line, for example `line 13: heading level skipped from h1 to h4`. Draft pages are skipped. To lint without building, run `sarde validate`. It lints by default (`--lint=false` skips it), and `--strict` exits with code 1 when any warning exists.

In `sarde dev`, an incremental rebuild lints only the changed pages, so warnings for other pages do not repeat until the next full build.
