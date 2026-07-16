---
title: Internationalization
description: "Configure multi-language sites with localized URLs, fallback pages, and a language switcher"
sidebar:
  order: 17
---

Sarde supports multi-language sites with localized URLs, automatic fallback pages, a language switcher, and translated UI strings. Content for each language lives in its own subdirectory under `content/`.

## Configuring languages

Define languages in `sarde.yaml` under the `i18n` key:

```yaml
i18n:
  default_language: "en"
  strategy: "prefix-except-default"
  fallback: "default"
  languages:
    en:
      name: "English"
      weight: 1
    fr:
      name: "Français"
      weight: 2
    ar:
      name: "العربية"
      weight: 3
      dir: "rtl"
```

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `default_language` | string | `"en"` | Language code for the primary language. |
| `strategy` | string | `"prefix-except-default"` | URL strategy. The default language has no prefix; others get `/<lang>/`. |
| `fallback` | string | `"default"` | `"default"` clones the default-language page. `"omit"` skips untranslated pages. |
| `strict` | bool | `false` | When `true`, records translation keys that fell back during resolution. |
| `languages` | map | `{}` | Map of language code to language config. An empty map means single-language. |

Each language entry accepts:

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `name` | string | none | Display name shown in the language switcher. |
| `title` | string | none | Optional site title override for this language. |
| `weight` | int | `0` | Sort order in the language switcher. Lower values appear first. |
| `dir` | string | `"ltr"` | Text direction. Set to `"rtl"` for Arabic, Hebrew, and similar scripts. |

## Content directory structure

Place translated content in language-prefixed directories under `content/`:

```
content/
  docs/
    getting-started.md      # English (default)
    guides/
      auth.md
  fr/
    docs/
      getting-started.md    # French translation
      guides/
        auth.md
  ar/
    docs/
      getting-started.md    # Arabic translation
```

The default language (`en` above) has no directory prefix. Non-default languages use `content/<lang>/` as the root, then mirror the same structure.

Sarde matches pages across languages by their path relative to the language prefix. `docs/getting-started.md` in the root and `fr/docs/getting-started.md` are treated as translations of each other.

## Localized URLs

With the `prefix-except-default` strategy (the only strategy currently supported), the default language serves at the site root. Other languages get a `/<lang>/` prefix:

| Language | Content path | URL |
|----------|-------------|-----|
| English (default) | `content/docs/getting-started.md` | `/docs/getting-started/` |
| French | `content/fr/docs/getting-started.md` | `/fr/docs/getting-started/` |
| Arabic | `content/ar/docs/getting-started.md` | `/ar/docs/getting-started/` |

## Translation strings

UI text (navigation labels, search prompts, version notices, error messages) comes from YAML translation files. Sarde merges strings from three layers, with later layers overriding earlier ones per key:

1. **Embedded defaults** (compiled into the binary, English)
2. **Theme `i18n/`** directory (e.g., `themes/mytheme/i18n/fr.yaml`)
3. **Project `i18n/`** directory (e.g., `i18n/fr.yaml`)

Create one file per language, named by language code:

```
i18n/
  en.yaml
  fr.yaml
  ar.yaml
```

Keys use dot notation. A French translation file:

`i18n/fr.yaml`
```yaml
nav:
  previous: "Précédent"
  next: "Suivant"
  toc: "Sur cette page"
  search: "Rechercher"
  language: "Langue"
search:
  no_results: "Aucun résultat."
fallback:
  notice: "Cette page n'est pas encore disponible en {{ .Lang }}."
```

Values can include Go template syntax. The `fallback.notice` key receives a `.Lang` variable with the current language's display name.

For the full list of built-in string keys, see the embedded `en.yaml` that ships with Sarde.

## Fallback pages

When a page exists in the default language but has no translation, Sarde generates a *fallback page*. The fallback displays the default-language content with a notice banner:

Result: A banner appears at the top: "This page is not yet available in French. Showing the original version."

<!-- SCREENSHOT: fallback-notice-banner - a fallback notice banner on an untranslated page -->

Control fallback behavior at two levels:

**Site-wide**, in `sarde.yaml`:

```yaml
i18n:
  fallback: "default"    # clone the default-language page (default)
  # fallback: "omit"     # skip untranslated pages entirely
```

**Per collection**, to override the site-wide setting:

```yaml
collections:
  blog:
    i18n_fallback: "omit"    # do not generate fallback blog posts
  docs:
    i18n_fallback: "default" # always show fallback docs pages
```

Fallback pages have `IsFallback: true` in templates. The `FallbackNotice` component checks this flag and renders the notice banner.

## Language switcher

When a page has translations (or fallback pages), the language switcher component appears in the header. It lists all available languages, sorted by weight.

Result: A dropdown shows each language by its display name. The current language is highlighted. Fallback entries are visually distinguished.

<!-- SCREENSHOT: language-switcher-dropdown - the language switcher open with three languages -->

The switcher links to the same page in each language. For fallback pages, the link points to the fallback URL. The dropdown closes on outside click or the Escape key.

## RTL support

Set `dir: "rtl"` on a language to enable right-to-left layout. Sarde sets the `dir` attribute on the `<html>` element and applies mirrored CSS for sidebar, navigation, and content layout.

```yaml
i18n:
  languages:
    ar:
      name: "العربية"
      dir: "rtl"
```

## Hreflang tags

The SEO plugin automatically emits `<link rel="alternate" hreflang="...">` tags for every page that has translations. The default-language page also gets an `x-default` hreflang tag. No configuration is needed beyond enabling the SEO plugin (enabled by default).

## Cross-language linking

Link to a specific language version of a page using the `?lang=` query parameter in internal links:

```markdown
[French version](/start-here/getting-started/?lang=fr)
```

The link validator resolves this to the French translation of the target page. Without the `?lang=` parameter, internal links resolve within the current language.

## Edge cases

- Sarde requires at least two entries in `i18n.languages` to enable multi-language mode. A single entry (or an empty map) produces a single-language site with no language prefixes.
- The `default_language` code does not need to appear in the `languages` map, but omitting it means the language switcher will not display a name for it.
- For versioned collections, fallback operates within a version. A missing French translation of `v2/guides/auth.md` falls back to the English `v2/guides/auth.md`, not `v3/guides/auth.md`.
- The `FallbackNotice` component renders on both docs and default layouts. Customize the notice text by overriding the `fallback.notice` key in your `i18n/<lang>.yaml` file.
