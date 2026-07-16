---
title: Internal Links
description: "Link between pages using relative paths or collection-root paths. Sarde resolves them to final URLs with the correct base path, language, and version."
sidebar:
  order: 4
---

Sarde resolves Markdown links to source files at build time. Write paths to `.md` files in your content; the engine rewrites them to final URLs that include the base path, language prefix, and version segment automatically.

## Syntax reference

| You write | Meaning | Resolves against |
|---|---|---|
| `[text](./auth.md)` | Relative to current file | Current file's directory, same language and version |
| `[text](./auth)` | Relative (no extension) | Same as above; `.md` extension is optional |
| `[text](../guides/auth)` | Parent directory traversal | Same rules, `../` works as expected |
| `[text](/guides/auth)` | Content-root (within collection) | Current collection's root directory |
| `[text](/guides/auth.md)` | Content-root (with `.md`) | Same as above; extension is optional |
| `[text](#setup)` | Same-page anchor | Current page's URL + `#setup` |
| `[text](./auth#setup)` | Path + anchor | Resolves path, validates `#setup` against target headings |
| `[text](./auth?highlight=true)` | Path + query | Query string preserved in output |
| `[text](./auth?lang=fr)` / `?version=v1` | Explicit cross-lane override | Resolves in the given language/version lane; reserved keys stripped from output |
| `[text](https://example.com)` | External URL | Passed through unchanged |
| `[text](/img/logo.png)` | Public asset (has a file extension) | Base path applied, not validated |

## What triggers resolution

Three link styles are resolved and validated. The `.md` extension is always optional.

**Relative links** (`./` or `../` prefix) are resolved from the current file's directory. The prefix is what triggers resolution, not the extension.

- `./page` or `./page.md` resolves to a sibling page
- `../other` or `../other.md` resolves via parent traversal

**Content-root links** (leading `/`, no file extension in the basename) are resolved from the current collection's root directory. This is the recommended style for within-collection linking.

- `/guide/page` or `/guide/page.md` resolves within the current collection
- `/api/` resolves to the section index (`_index.md`)

**Same-page anchors** (`#anchor`) resolve as a fragment on the current page's URL.

Everything else passes through unchanged:

- `https://example.com` passes through as an external link
- `/` is the site root; base path applied, never validated
- `/img/logo.png` (leading `/` with a non-markdown file extension) is treated as a public asset; base path applied, never validated

## What is rejected

Bare names without a `./`, `../`, or `/` prefix that include a `.md` extension are **ambiguous** and produce a build error:

```markdown
[text](auth.md)        <!-- ERROR: ambiguous, use ./auth.md instead -->
[text](guides/auth.md) <!-- ERROR: ambiguous, use ./guides/auth.md instead -->
```

This prevents silent mis-resolution when multiple files share a name. Always use an explicit prefix.

## Resolution rules

### Relative links

Resolved from the current file's directory within the content tree. The version segment is stripped during resolution so that links work identically across versions:

```text
content/docs/v2/guide/03-quick-start.md
                       contains ./02-installation.md
                       resolves to: /docs/guide/installation/
```

Numeric filename prefixes (e.g. `02-`) are stripped by Sarde's slug algorithm. `02-installation.md` becomes `/installation/` in the URL.

### Content-root links

The leading `/` means "from this collection's root." Sarde prepends the current collection name automatically.

```markdown
<!-- In a docs page: both resolve within the docs collection -->
[Router API](/api/router.md)  -->  /docs/api/router/
[Router API](/api/router)     -->  /docs/api/router/   (same result)
```

Content-root links are fully i18n-aware: on a French page, `/guide/quick-start` resolves to `/fr/docs/guide/quick-start/`. On an English page, the same link resolves to `/docs/guide/quick-start/`. The base path, language prefix, and version segment are all applied automatically.

### Directory links

Linking to a directory resolves to its section index:

```markdown
[Guides](./guides/)  -->  resolves to guides/_index.md
```

## URL composition

The same link resolves differently depending on the page's language and version context:

| You write (in `.md`) | Page context | Built HTML output |
|---|---|---|
| `/guide/quick-start` | English, docs collection | `/docs/guide/quick-start/` |
| `/guide/quick-start` | French, docs collection | `/fr/docs/guide/quick-start/` |
| `/guide/quick-start` | English, docs v1 | `/docs/v1/guide/quick-start/` |
| `./quick-start` | French, in guide/ dir | `/fr/docs/guide/quick-start/` |
| `../api/router` | English, in guide/ dir | `/docs/api/router/` |
| `https://example.com` | Any | `https://example.com` (unchanged) |
| `/assets/logo.png` | Any | `/assets/logo.png` (asset, no lang/version) |

### Language

Links resolve within the current page's language. A relative link on a French page finds the French translation of the target. To link across languages, use the `?lang=` query parameter:

```markdown
[English version](./auth?lang=en)
```

### Version

Links resolve within the current page's version:

- On a **latest-version** page (served at the unprefixed URL), links resolve to unprefixed URLs.
- On an **older-version** page (e.g. `/docs/v1/guide/...`), links stay within that version. A target that exists in latest but not in v1 is a build error.

To link across versions, use the `?version=` query parameter:

```markdown
[See v1 docs](./auth?version=v1)
```

Reserved query keys (`lang`, `version`) are stripped from the output URL.

## Anchor validation

When a link includes a `#fragment`, Sarde validates it against the target page's heading IDs after all pages are rendered. Invalid anchors are reported as broken links:

```markdown
[Setup](./auth.md#setup)        <!-- OK if auth.md has a ## Setup heading -->
[Missing](./auth.md#nonexistent) <!-- build error: heading not found -->
```

Same-page anchors (`#section-name`) are also validated.

## Common patterns

### Link to a sibling page

```markdown
[Installation](./02-installation)
[Installation](./02-installation.md)   <!-- also works, .md is optional -->
```

### Link to a page in a parent directory

```markdown
[Back to overview](../overview)
```

### Link to a section index

```markdown
[All guides](./guides/)
```

### Link from collection root

```markdown
[API Reference](/api/router)
[Quick Start](/guide/quick-start)
```

### Link with anchor

```markdown
[Configuration section](./setup#configuration)
```

### External link

```markdown
[GitHub](https://github.com/example/project)
```

### Public asset

```markdown
![Logo](/img/logo.png)
[Download PDF](/files/report.pdf)
```

### Markdown extensions

The same link syntax works in extension attributes like link cards and link buttons:

```markdown
:::link-card[Getting Started](href="/guide/quick-start" icon="book-open")
:::

:::link-button[Read the Docs](href="/guide/introduction" variant="primary")
:::
```

These go through the same resolution pipeline.

## Gotchas

**Bare names are rejected.** Always use `./` for relative links. `auth.md` without a prefix is ambiguous and will error. Write `./auth.md` instead.

**Numeric prefixes are stripped.** Files named `01-introduction.md`, `02-installation.md` produce slugs `introduction`, `installation`. When linking, use the original filename: `./02-installation.md` resolves correctly to `/guide/installation/`.

**Content-root `/` means collection root.** A leading `/` in a link means "from this collection's root," not from the site root. `/guide/auth` from a `docs` page resolves within the `docs` collection.

**Link validation.** Sarde validates every internal link at build time. Broken targets and anchors fail the build by default. See [Link Validation and Linting](/guides/link-validation-and-linting) for configuration options and policy settings.
