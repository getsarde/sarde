---
title: Internal Links
description: "Link between pages using relative paths or collection-root paths. Sarde resolves them to final URLs with the correct base path, language, and version."
sidebar:
  order: 4
---

Link to another page by the path to its source file. At build time, Sarde rewrites each link to the final URL, including the base path, language prefix, and version segment, and checks that the target exists.

## Syntax reference

Each row shows a link form, what it means, and what Sarde resolves it against:

| You write | Meaning | Resolves against |
|---|---|---|
| `[text](./auth.md)` | Relative to the current file | The current file's directory, same language and version |
| `[text](./auth)` | Relative, no extension | Same as above. The `.md` extension is optional. |
| `[text](../guides/auth)` | Parent directory traversal | Same rules. `../` works as in a file path. |
| `[text](/guides/auth)` | Collection-root path | The current collection's root directory |
| `[text](/guides/auth.md)` | Collection-root path with `.md` | Same as above. The extension is optional. |
| `[text](/)` | Collection root | The current collection's index page. On a page at the `content/` root, the homepage. |
| `[text](#setup)` | Same-page anchor | The current page's URL plus `#setup` |
| `[text](./auth#setup)` | Path and anchor | Resolves the path, then checks `#setup` against the target's headings |
| `[text](./auth?highlight=true)` | Path and query | The query string is kept in the output |
| `[text](./auth?lang=fr)` or `?version=v1` | Explicit cross-language or cross-version link | Resolves in the given language or version. Sarde removes these two keys from the output URL. |
| `[text](site:/pricing)` | Site-root link | `/pricing` at the site root, with no collection, language, or version. Never validated. |
| `[text](https://example.com)` | External URL | Passed through unchanged |
| `[text](/img/logo.png)` | Public asset (a file extension other than `.md`) | Base path applied, not validated |

## What triggers resolution

Sarde resolves and validates three link styles. The `.md` extension is always optional.

**Relative links** start with `./` or `../` and resolve from the current file's directory. The prefix triggers resolution, not the extension.

- `./page` or `./page.md` resolves to a sibling page.
- `../other` or `../other.md` resolves through the parent directory.

**Collection-root links** start with `/` and have no file extension in the last segment. They resolve from the current collection's root directory.

- `/guide/page` or `/guide/page.md` resolves within the current collection.
- `/api/` resolves to the section index (`_index.md`).

**Same-page anchors** (`#anchor`) resolve as a fragment on the current page's URL.

Everything else passes through unchanged:

- `https://example.com` and other absolute URLs are external links.
- `/img/logo.png` and any other `/` path with a non-Markdown extension is a public asset. Sarde applies the base path and does not validate it.
- A bare name without `.md`, such as `auth`, is left exactly as written, and the browser resolves it against the current URL. Write `./auth` or `/guide/auth` instead.

### Relative links produce a warning

By default, each `./` or `../` link prints a `relative link` warning at build time (`link_validation.on_relative_links` is `warn`). The link still resolves. Prefer collection-root links within a collection to keep the build output quiet. See [Link Validation and Linting](/guides/link-validation-and-linting/) to change the policy.

## What is rejected

A bare name that ends in `.md` and has no `./`, `../`, or `/` prefix is **ambiguous** and fails the build:

```markdown
[text](auth.md)        <!-- ERROR: ambiguous, use ./auth.md instead -->
[text](guides/auth.md) <!-- ERROR: ambiguous, use ./guides/auth.md instead -->
```

The prefix removes the doubt about whether the name is a sibling file or a path from the content root. Always write an explicit prefix.

## Resolution rules

The resolved link styles differ in where the path starts.

### Relative links

A relative link resolves from the current file's directory within the content tree and stays in the page's language and version, so the same link works in every version:

```text
content/docs/guide/03-quick-start.md
                       contains ./02-installation.md
                       resolves to: /docs/guide/installation/
```

Sarde strips numeric filename prefixes such as `02-` from the URL, but the link still names the file as written on disk.

A relative link can cross collections. `../../blog/hello-world` on a page at `content/docs/guide/auth.md` resolves to the blog post.

### Collection-root links

A leading `/` starts from the current collection's root, not from the site root. Sarde adds the collection name.

```markdown
<!-- In a docs page: both resolve within the docs collection -->
[Router API](/api/router.md)  -->  /docs/api/router/
[Router API](/api/router)     -->  /docs/api/router/   (same result)
```

Collection-root links follow the page's language: on a French page, `/guide/quick-start` resolves to `/fr/docs/guide/quick-start/`, and on an English page to `/docs/guide/quick-start/`. The base path and version segment apply the same way.

A collection-root link cannot reach another collection, because `/blog/post` on a docs page means `content/docs/blog/post`. To link into another collection, use a relative path or a `site:` link (see [Link to another collection](#link-to-another-collection)). To reach the homepage from a collection page, write `site:/`.

### Directory links

A link to a directory resolves to its section index:

```markdown
[Guides](./guides/)  -->  resolves to guides/_index.md
```

## Links that do not resolve

What happens to a link with no target depends on how it is written:

| Link | Result by default |
|---|---|
| Relative link to a missing page, or any link written with `.md` | Build error: `broken target` |
| Extension-less collection-root link that matches no page in the current language and version | Build warning: `unverified internal`. Sarde leaves the URL as written, which may not exist. |
| Link to a heading that does not exist | Build error: `broken anchor` |

The extension-less collection-root case is a warning because the author may have meant a page in another collection, language, or version. Add `.md` to the link to turn it into an error. See [Link Validation and Linting](/guides/link-validation-and-linting/) for the policy settings.

## URL composition

The same link resolves differently depending on the page's language and version:

| You write (in `.md`) | Page context | Built HTML output |
|---|---|---|
| `/guide/quick-start` | English, docs collection | `/docs/guide/quick-start/` |
| `/guide/quick-start` | French, docs collection | `/fr/docs/guide/quick-start/` |
| `/guide/quick-start` | English, docs v1 (older version) | `/docs/v1/guide/quick-start/` |
| `./quick-start` | French, in `guide/` directory | `/fr/docs/guide/quick-start/` |
| `../api/router` | English, in `guide/` directory | `/docs/api/router/` |
| `https://example.com` | Any | `https://example.com` (unchanged) |
| `/assets/logo.png` | Any | `/assets/logo.png` (asset, no language or version) |

### Language

Links resolve within the current page's language. A relative link on a French page finds the French translation of the target. To link across languages, use the `?lang=` query parameter:

```markdown
[English version](./auth?lang=en)
```

### Version

Links resolve within the current page's version:

- On a page of the latest version (served at the unprefixed URL), links resolve to unprefixed URLs.
- On a page of an older version (for example `/docs/v1/guide/...`), links stay in that version. A `.md` or relative link to a page that exists only in the latest version is a `broken target` error, and an extension-less collection-root link is an `unverified internal` warning.

To link across versions, use the `?version=` query parameter:

```markdown
[See v1 docs](./auth?version=v1)
```

Sarde removes the reserved query keys `lang` and `version` from the output URL.

## Anchor validation

When a link includes a `#fragment`, Sarde checks it against the target page's heading IDs after all pages render. A missing heading fails the build:

```markdown
[Setup](./auth.md#setup)        <!-- OK if auth.md has a ## Setup heading -->
[Missing](./auth.md#nonexistent) <!-- build error: broken anchor -->
```

Same-page anchors (`#section-name`) are checked too.

## Other link forms

These forms cover targets outside the current collection and links written in extension attributes.

### Link to another collection

From a docs page, link to a blog post with a relative path, which is validated, or with a `site:` link, which is not:

```markdown
[Release notes](../../blog/hello-world)
[Release notes](site:/blog/hello-world)
```

`site:` is the default value of `link_validation.site_root_escape_prefix`. It also reaches pages at the `content/` root, such as `site:/about`.

### Public asset

Link to a file in `public/` by its path from the site root:

```markdown
![Logo](/img/logo.png)
[Download PDF](/files/report.pdf)
```

Sarde applies the base path to a linked asset and does not check that the file exists.

### Markdown extensions

The same syntax works in extension attributes such as link cards and link buttons, and they go through the same resolution:

```markdown
:::link-card[Getting Started](href="/guide/quick-start" icon="book-open")
:::

:::link-button[Read the Docs](href="/guide/introduction" variant="primary")
:::
```
