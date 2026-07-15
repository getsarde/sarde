---
title: Versioning
description: "Set up multiple documentation versions with a version switcher, URL prefixes, and search scoping"
sidebar:
  order: 17
---

Sarde supports multiple documentation versions within a single collection. Readers switch between versions using a dropdown in the header, and each version gets its own sidebar navigation, URL prefix, and search scope.

## Versioning config

Enable versioning on a collection in `sarde.yaml`:

```yaml
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
          banner: "unmaintained"
          redirect: "root"
```

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `enabled` | bool | `false` | Enable versioning for this collection. |
| `last_version` | string | none | Version ID treated as "latest." Its content serves at the collection root URL. Must match one of the `versions[].id` values. |
| `publish_latest_at_version_url` | bool | `false` | When `true`, the latest version also publishes at its versioned URL (`/docs/v3/...`) in addition to the collection root. |
| `fallback` | string | none | i18n fallback policy for this versioned collection. `"default"` or `"omit"`. |

## Version entries

Each entry in the `versions` list defines one version:

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `id` | string | none | Unique identifier. Must match the directory name under the collection. |
| `label` | string | same as `id` | Display name shown in the version switcher dropdown. |
| `path` | string | same as `id` | URL path segment. Defaults to the `id`. |
| `banner` | string | `"none"` | `"none"`, `"unmaintained"`, or `"unreleased"`. |
| `redirect` | string | `"same-page"` | `"same-page"` links to the equivalent page in other versions. `"root"` links to the collection root. |

Version IDs must be unique within a collection. The `last_version` value must appear in the list.

## Content directory layout

Place versioned content in subdirectories named by version ID:

```
content/
  docs/
    getting-started.md       # latest version (matches last_version)
    guides/
      auth.md
    v2/
      getting-started.md     # version 2
      guides/
        auth.md
    v1/
      getting-started.md     # version 1
```

The latest version's content lives at the collection root (not inside a version directory). Older versions each get their own subdirectory.

## URL structure

The latest version serves at the collection root. Older versions include the version ID in the URL:

| Version | Content path | URL |
|---------|-------------|-----|
| v3 (latest) | `content/docs/getting-started.md` | `/docs/getting-started/` |
| v2 | `content/docs/v2/getting-started.md` | `/docs/v2/getting-started/` |
| v1 | `content/docs/v1/getting-started.md` | `/docs/v1/getting-started/` |

When `publish_latest_at_version_url` is `true`, the latest version also appears at `/docs/v3/getting-started/` (in addition to `/docs/getting-started/`).

## Version banners

Set `banner` on a version entry to display a notice at the top of every page in that version:

- `"unmaintained"`: "You are viewing documentation for an older version."
- `"unreleased"`: "You are viewing documentation for an unreleased version."
- `"none"` (default): No banner.

Result: A yellow banner appears with a link to the latest stable version.

<!-- SCREENSHOT: version-banner-unmaintained - an unmaintained version banner with a link to latest -->

Both banner types include a link to the latest version. The banner text is translatable via the `version.unmaintained_notice`, `version.unreleased_notice`, `version.unmaintained_link`, and `version.unreleased_link` keys in the `i18n/` files.

## Version switcher

The version switcher appears in the header when the current page belongs to a versioned collection. It lists all configured versions with their labels.

Result: A dropdown shows each version label. The current version is highlighted. The latest version is marked.

<!-- SCREENSHOT: version-switcher-dropdown - the version switcher open with three versions -->

The switcher links to the same page in each version when `redirect: "same-page"` is set. If the equivalent page does not exist in the target version, the link falls back to the version's collection root. Versions with `redirect: "root"` always link to the collection root.

## Cross-version linking

Sarde groups pages across versions by their path relative to the version directory. Each page's `VersionPeers` field contains pointers to the same page in other versions.

Link to a specific version of a page using the `?version=` query parameter:

```markdown
[See the v2 guide](/docs/guides/auth/?version=v2)
```

The link validator resolves this to the correct versioned URL. Without the `?version=` parameter, internal links resolve within the current version.

## Version-scoped search

Search results are scoped to the currently viewed version. A reader browsing v2 docs sees only v2 results. The search index is built per version, so version-scoped queries return accurate results without client-side filtering.

## SEO canonical URLs

Non-latest versioned pages automatically point their `<link rel="canonical">` tag to the equivalent page in the latest version. This consolidates search engine signals on the current documentation rather than spreading them across old versions.

For example, `/docs/v2/getting-started/` has its canonical URL set to `/docs/getting-started/` (the latest version). The SEO plugin handles this automatically when versioning is enabled.

## Versioning with i18n

Versioning and internationalization compose. Each (language, version) pair gets its own sidebar tree and navigation. Fallback pages operate within a version: a missing French translation of a v2 page falls back to the English v2 page, not a different version.

```
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

The URL for the French v2 page is `/fr/docs/v2/getting-started/`. The language prefix comes first, followed by the collection mount, then the version segment.

## Edge cases

- A version entry with an empty `id` causes a build validation error.
- Duplicate version IDs in the same collection cause a build validation error.
- Setting `last_version` to a value not present in the `versions` list causes a build validation error.
- If `last_version` is not set, unversioned content at the collection root is treated as the implicit "Latest" version.
- The version switcher does not appear on non-versioned collections, even if other collections on the same site use versioning.
- Each version has its own independent sidebar navigation tree. Pages from different versions do not mix in the sidebar.
