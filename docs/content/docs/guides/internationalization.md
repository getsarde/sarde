---
title: Internationalization
description: "Configure multi-language sites with localized URLs, fallback pages, and a language switcher"
sidebar:
  order: 18
---

Sarde builds multi-language sites with localized URLs, fallback pages for untranslated content, a language switcher, and translated UI strings. Content for each non-default language lives in its own subdirectory under `content/`.

## Configuring languages

Define languages in `sarde.yaml` under the `i18n` key. List the default language in `languages` too:

```yaml title="sarde.yaml"
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
| `default_language` | string | `"en"` | Code of the primary language. It must also appear in `languages`, or the build fails with `default_language "en" is not listed in languages`. |
| `strategy` | string | `"prefix-except-default"` | URL strategy. The default language has no prefix, and other languages get `/<lang>/`. This is the only supported value. |
| `fallback` | string | `"default"` | `"default"` generates a fallback page from the default-language page. `"omit"` skips untranslated pages. |
| `strict` | bool | `false` | Adds a build warning for each UI string that the build looked up in a language that does not define it. The `--strict-i18n` flag turns it on for one build. |
| `languages` | map | `{}` | Language code to language settings. An empty map means a single-language site. |

Each language entry accepts:

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `name` | string | language code | Display name in the language switcher. |
| `weight` | int | `0` | Position in the language switcher. Lower values come first, and ties sort by code. |
| `dir` | string | `"ltr"` | Text direction, `"ltr"` or `"rtl"`. Use `"rtl"` for Arabic, Hebrew, and similar scripts. |

A map with a single language builds a site without language prefixes and without a switcher.

## Add a language with the CLI

Three commands set up and track a language. Each prints one line of JSON, because the desktop app reads their output.

```sh
sarde i18n add-language . fr --name "Français" --weight 2
sarde i18n scaffold . fr
sarde i18n status .
```

- `add-language` adds the code to `i18n.languages` in `sarde.yaml`. It rewrites the file from parsed data, so comments and key order may change.
- `scaffold` creates `content/fr/<collection>/_index.md` stubs for each collection and an `i18n/fr.yaml` seeded from the default language's project file.
- `status` reports how many Markdown files each language has per collection compared with the default language.

See [CLI Commands](/reference/cli-commands/#i18n) for every flag.

## Content directory structure

Place translated content in a directory named for the language code under `content/`:

```text
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

The default language has no directory prefix. Other languages use `content/<lang>/` as their root and mirror the same structure.

Sarde matches pages across languages by their path after the language prefix. `docs/getting-started.md` and `fr/docs/getting-started.md` are translations of each other.

## Localized URLs

With `prefix-except-default`, the default language serves from the site root and other languages get a `/<lang>/` prefix:

| Language | Content path | URL |
|----------|-------------|-----|
| English (default) | `content/docs/getting-started.md` | `/docs/getting-started/` |
| French | `content/fr/docs/getting-started.md` | `/fr/docs/getting-started/` |
| Arabic | `content/ar/docs/getting-started.md` | `/ar/docs/getting-started/` |

## Translation strings

UI text such as navigation labels, search prompts, version notices, and error messages comes from YAML translation files. Sarde merges strings from three layers, and later layers override earlier ones per key:

1. **Embedded defaults** compiled into the binary, in English only
2. **Theme `i18n/` directory**, for example `themes/mytheme/i18n/fr.yaml`
3. **Project `i18n/` directory**, for example `i18n/fr.yaml`

Create one file per language, named by language code:

```text
i18n/
  en.yaml
  fr.yaml
  ar.yaml
```

A key that a language does not define falls back to the default language's string. Because the embedded defaults are English, a site whose `default_language` is not `en` needs an `i18n/<default_language>.yaml` file. Without it, the pages show raw keys such as `nav.previous`.

Keys use nested YAML, addressed in dot notation. A French translation file:

```yaml title="i18n/fr.yaml"
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

Values can use Go template syntax. Three keys receive variables:

| Key | Variable |
|-----|----------|
| `nav.reading_time` | `.Minutes`, the reading time in minutes |
| `nav.toggle_section` | `.Label`, the sidebar group label |
| `fallback.notice` | `.Lang`, the page's language code, such as `fr` |

### Built-in string keys

Any key can be overridden. These are the groups:

| Group | Keys |
|-------|------|
| `nav` | `previous`, `next`, `newer`, `older`, `newer_posts`, `older_posts`, `toc`, `toc_label`, `toc_overview`, `overview`, `search`, `draft`, `reading_time`, `language`, `version`, `version_latest`, `back_to_top`, `skip_to_content`, `menu`, `collapse_sidebar`, `expand_sidebar`, `toggle_section`, `pagination`, `main_nav`, `post_nav`, `docs_nav`, `sidebar`, `center_content`, `breadcrumb`, `section_nav`, `page_nav`, `footer`, `opens_new_tab` |
| `search` | `results`, `no_results`, `close`, `tip_typos`, `tip_keywords`, `kbd_navigate`, `kbd_select`, `kbd_close`, `full_search`, `switch_full_search`, plus runtime strings such as `recent`, `type_to_search`, and `results_count_one`. See [Search](/guides/search/) |
| `theme` | `selection`, `light`, `system`, `dark` |
| `taxonomy` | `post`, `posts`, `tags`, `authors` |
| `blog` | `featured`, `page`, `of` |
| `labs` | `step`, `of`, `overview`, `learning_objectives` |
| `home` | `all_posts`, `get_started`, `hero_highlights` |
| `error` | `not_found`, `not_found_desc`, `return_home`, `not_found_illustrated`, `not_found_illustrated_desc` |
| `version` | `unmaintained_notice`, `unreleased_notice`, `unmaintained_link`, `unreleased_link` |
| `fallback` | `notice` |
| `announcements` | `dismiss` |
| `banner` | `note`, `tip`, `caution`, `danger` |
| `ui` | `reveal_spoiler`, `hide_spoiler` |
| `draft` | `notice` |
| `telescope` | Labels for the quick-navigation dialog |
| (top level) | `edit_this_page`, `last_updated_label` |

To find the keys a language still lacks, set `i18n.strict: true` and build. Each missing key appears as a warning naming the `i18n/<lang>.yaml` file.

## Fallback pages

When a page exists in the default language but has no translation, Sarde generates a fallback page at the translated URL. The fallback shows the default-language content with a notice banner.

→ A banner at the top of the page reads "This page is not yet available in fr. Showing the original version."

<!-- SCREENSHOT: fallback-notice-banner - a fallback notice banner on an untranslated page -->

Set the behavior site-wide in `sarde.yaml`:

```yaml title="sarde.yaml"
i18n:
  fallback: "default"    # generate a fallback page (default)
  # fallback: "omit"     # skip untranslated pages entirely
```

Override it per collection:

```yaml title="sarde.yaml"
collections:
  blog:
    i18n_fallback: "omit"    # do not generate fallback blog posts
  docs:
    i18n_fallback: "default" # always show fallback docs pages
```

Fallback pages set `IsFallback` to `true` in templates, and the `FallbackNotice` component renders the banner when it is set. Customize the text through the `fallback.notice` key. The component renders on the docs and default layouts.

In a versioned collection, fallback stays within a version. A missing French translation of `v2/guides/auth.md` falls back to the English `v2/guides/auth.md`, not to `v3/guides/auth.md`.

## Language switcher

When a page has translations or fallback pages, a language switcher appears in the header. It lists the available languages in `weight` order.

→ A dropdown shows each language by its display name. The current language is highlighted, and fallback entries carry a distinct style.

<!-- SCREENSHOT: language-switcher-dropdown - the language switcher open with three languages -->

Each entry links to the same page in that language, or to its fallback page. The dropdown closes on an outside click or the Escape key.

## RTL support

Set `dir: "rtl"` on a language to switch it to right-to-left layout. Sarde sets the `dir` attribute on the `<html>` element and applies mirrored CSS for the sidebar, navigation, and content layout.

```yaml title="sarde.yaml"
i18n:
  languages:
    ar:
      name: "العربية"
      dir: "rtl"
```

## Hreflang tags

The default theme adds `<link rel="alternate" hreflang="...">` tags to every page that has translations, including an `x-default` tag that points to the default-language version. No configuration is needed.

## Cross-language linking

Links between pages stay within the current language. To link to a specific language version of a page, add `?lang=` to an [internal link](/guides/internal-links/):

```markdown
[French version](/guides/auth/?lang=fr)
```

Written inside the `docs` collection, this link becomes `/fr/docs/guides/auth/` in the built page, and Sarde removes the query parameter. The link checker validates the target in the French content.
