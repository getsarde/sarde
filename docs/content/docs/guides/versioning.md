---
title: Versioning
description: "Set up multiple documentation versions with a version switcher, URL prefixes, and search scoping"
sidebar:
  order: 19
---

Sarde supports multiple documentation versions within a single collection. Readers switch versions from a dropdown in the header, and each version gets its own sidebar, URL prefix, and search scope. Versioning is off by default.

## Versioning config

Enable versioning on a collection in `sarde.yaml`:

```yaml title="sarde.yaml"
collections:
  docs:
    versioning:
      enabled: true
      last_version: "v3"
      versions:
        - id: "v3"
          label: "3.0"
        - id: "v2"
          label: "2.0"
          banner: "unmaintained"
        - id: "v1"
          label: "1.0"
          banner: "unreleased"
          redirect: "root"
```

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `enabled` | bool | `false` | Turn versioning on for this collection. |
| `last_version` | string | none | ID of the latest version. Its content serves at the collection root URL. Must match one of the `versions[].id` values. |
| `publish_latest_at_version_url` | bool | `false` | Also publish the latest version at its versioned URL (`/docs/v3/...`), in addition to the collection root. |
| `fallback` | string | none | i18n fallback policy for this collection, `"default"` or `"omit"`. Takes precedence over `i18n_fallback`. |
| `versions` | list | none | The versions, described below. |

## Version entries

Each entry in `versions` defines one version:

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `id` | string | none | Unique identifier. Must match the version's directory name under the collection and appears in its URLs. |
| `label` | string | same as `id` | Name shown in the version switcher. |
| `path` | string | same as `id` | Reserved. URLs always use the `id`. |
| `banner` | string | `"none"` | `"none"`, `"unmaintained"`, or `"unreleased"`. |
| `redirect` | string | `"same-page"` | How the switcher links to this version. `"same-page"` links to the equivalent page. `"root"` always links to the version's root. |

Version IDs must be non-empty and unique within a collection, and `last_version` must appear in the list. Otherwise the build stops with a validation error.

Without `last_version`, content at the collection root is the implicit latest version. The switcher lists it first as "Latest", followed by every configured version, and older versions get no canonical link to it.

## Content directory layout

Place versioned content in subdirectories named by version ID. The latest version's content stays at the collection root, outside any version directory:

```text
content/
  docs/
    getting-started.md       # latest version (matches last_version)
    guides/
      auth.md
    v2/
      _index.md              # version root, linked as /docs/v2/
      getting-started.md     # version 2
      guides/
        auth.md
    v1/
      getting-started.md     # version 1
```

## URL structure

The latest version serves at the collection root. Older versions include the version ID in the URL:

| Version | Content path | URL |
|---------|-------------|-----|
| v3 (latest) | `content/docs/getting-started.md` | `/docs/getting-started/` |
| v2 | `content/docs/v2/getting-started.md` | `/docs/v2/getting-started/` |
| v1 | `content/docs/v1/getting-started.md` | `/docs/v1/getting-started/` |

With `publish_latest_at_version_url: true`, the latest version also appears at `/docs/v3/getting-started/`.

## Cut a new version

`sarde doc-version create` freezes the current latest version and starts the next one:

```sh
sarde doc-version create . docs v4 "4.0"
```

When `last_version` is already set, the command copies the collection's current root content into `content/docs/<old last_version>/`. It then adds a `v4` entry labeled "4.0" to `sarde.yaml` and sets `last_version` to `v4`. The root content stays in place and becomes v4. The command prints one line of JSON and rewrites `sarde.yaml` from parsed data, so comments and key order may change.

The frozen version gets no banner. Set one with `sarde doc-version update . docs v3 --banner unmaintained`, or edit `sarde.yaml`. See [CLI Commands](/reference/cli-commands/#doc-version) for `update` and `delete`.

## Version banners

Set `banner` on a version entry to show a notice at the top of every page in that version:

- `"unmaintained"`: "You are viewing documentation for an older version." in an amber box.
- `"unreleased"`: "You are viewing documentation for an unreleased version." in a blue box.
- `"none"` (default): no banner.

→ The notice appears with a link to the latest version.

<!-- SCREENSHOT: version-banner-unmaintained - an unmaintained version banner with a link to latest -->

The banner text and link label are translatable through the `version.unmaintained_notice`, `version.unreleased_notice`, `version.unmaintained_link`, and `version.unreleased_link` keys. See [Translation strings](/guides/internationalization/#translation-strings).

## Version switcher

The switcher appears in the header on pages of a versioned collection. It lists every configured version by label.

→ A dropdown shows each version label. The current version is highlighted, and the latest version carries a "Latest" badge.

<!-- SCREENSHOT: version-switcher-dropdown - the version switcher open with three versions -->

With `redirect: "same-page"`, an entry links to the same page in that version. When that version has no equivalent page, the link goes to the version's root, such as `/docs/v1/`. An entry with `redirect: "root"` always links to the version's root.

A version root is the version directory's `_index.md`. Without one, `/docs/v1/` has no page, so add an `_index.md` to every version directory you link to.

## Cross-version linking

Internal links resolve within the current version. To link to a specific version of a page, add `?version=` to a collection-root [internal link](/guides/internal-links/):

```markdown
[See the v2 guide](/guides/auth/?version=v2)
```

Written inside the `docs` collection, this link becomes `/docs/v2/guides/auth/` in the built page, and Sarde removes the query parameter. The link checker validates the target in v2. A link that already starts with the collection name, such as `/docs/guides/auth/?version=v2`, is not rewritten.

## Version-scoped search

Search results are scoped to the version being viewed. A reader on v2 pages sees only v2 results. Sarde builds one search index per language and tags each entry with its version, and the search dialog filters on the current page's version when it queries.

## SEO canonical URLs

With `last_version` set, pages of older versions point their `<link rel="canonical">` tag to the equivalent page in the latest version, which directs search engines to the current documentation. For example, `/docs/v2/getting-started/` has the canonical URL `/docs/getting-started/`.

## Versioning with i18n

Versioning and [internationalization](/guides/internationalization/) compose. Each language and version pair gets its own sidebar tree. Fallback pages stay within a version: a missing French translation of a v2 page falls back to the English v2 page, not to another version.

```text
content/
  docs/
    getting-started.md          # English, latest
    v2/
      getting-started.md        # English, v2
  fr/
    docs/
      getting-started.md        # French, latest
      v2/
        getting-started.md      # French, v2
```

The French v2 page is served at `/fr/docs/v2/getting-started/`: the language prefix, then the collection, then the version.
