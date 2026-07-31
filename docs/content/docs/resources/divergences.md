---
title: Divergences from Hugo and Starlight
description: "Behavioral differences you will notice after migrating to Sarde from Hugo or Starlight"
sidebar:
  order: 3
---

This page lists the places where Sarde deliberately behaves differently from Hugo and
Starlight: the things you will notice after your content is moved, not the one-time
conversion steps. For the mechanical migration itself (field mappings, syntax
conversion, checklists), see the [Migration Guide](/resources/migration-guide/).

Every entry here is an intentional design decision, not a missing feature that is on
its way. If something you rely on is absent and not listed here, check the
[Changelog](/resources/changelog/) or open an issue.

## Frontmatter field differences

Most fields map one to one (see the [Migration Guide](/resources/migration-guide/) for
the full tables). The ones that change behavior, not just names:

- **`weight` is `sidebar.order`.** Sarde groups all sidebar-related fields under a
  `sidebar` key (`order`, `label`, `hidden`, `badge`) instead of Hugo's top-level
  `weight`. A leftover top-level `weight` is reported as an unknown key by
  `sarde validate` rather than silently ignored.
- **Missing fields are inferred, not defaulted to empty.** Title falls back to the
  first H1, then the filename; date falls back to the git commit date, then file
  mtime; sidebar order falls back to a numeric filename prefix (`01-intro.md`). Hugo
  leaves missing fields empty. If a migrated page shows an unexpected title or date,
  inference is usually why.
- **Starlight's TOC keys are flattened.** `tableOfContents.minHeadingLevel` and
  `maxHeadingLevel` become `toc.min_level` and `toc.max_level`; `template: splash`
  becomes `layout: splash`.

## Components and shortcodes vs directives

Sarde's native syntax for rich content is the `:::` block directive, written in plain
Markdown:

- **Coming from Starlight:** there is no MDX and no JSX. `<Tabs>`, `<Card>`,
  `<Steps>`, `<Aside>`, `<FileTree>`, and `<Badge>` all have directive equivalents
  (the [Migration Guide](/resources/migration-guide/#from-starlight-astro) has the
  mapping table), and `import` statements are removed entirely. Files are `.md`, not
  `.mdx`, and there is no way to embed framework components in content.
- **Coming from Hugo:** your `{{</* name */>}}` shortcodes keep working with the same
  syntax via templates in `layouts/shortcodes/`. The built-in content elements
  (callouts, tabs, cards, and the rest) use directives instead of shortcodes, so a
  site typically ends up with both syntaxes during a transition.

See [Using Extensions](/extensions/using-extensions/) for directive syntax and
nesting rules.

## Collections and taxonomy behavior

- **Collections are detected by directory name, not declared.** `content/blog/` is a
  blog, `content/docs/` is a docs collection with sidebar and versioning support,
  with no `archetypes/` or content-type configuration. An unrecognized directory name
  becomes a generic collection; behavior for it can be set explicitly under
  `collections:` in `sarde.yaml`.
- **Taxonomies work out of the box.** `tags`, `categories`, and `authors` generate
  term and list pages without the explicit `taxonomies:` declaration Hugo requires.
  Custom taxonomies are configured in `sarde.yaml` when you need more. See
  [Blog and Taxonomies](/guides/blog-and-taxonomies/).
- **Coming from Starlight:** the sidebar is generated from the directory structure
  (plus optional `sidebar.yaml` overrides) rather than a manually maintained sidebar
  array in `astro.config.mjs`.

## The i18n model

Sarde's translation linking is directory-based and convention-driven. Four Hugo
behaviors intentionally have no equivalent:

- **No `translationKey`.** Translations are linked by matching relative path:
  `content/guide.md` pairs with `content/fr/guide.md`. There is no frontmatter key to
  link differently-named files across languages, so translated files must keep the
  same relative path and name.
- **No filename-based language detection.** Hugo's `guide.fr.md` suffix convention is
  not recognized; languages are separated by directory only.
- **No pluralization sub-keys.** UI strings resolve to a single string per key.
  Hugo's go-i18n `one`/`other` plural forms have no equivalent.
- **No per-language config overrides.** A language entry carries its name, direction,
  and weight. Hugo's per-language `title`, `baseURL`, and `params` overrides have no
  equivalent.

In exchange, fallback pages (untranslated content served from the default language
with a notice), the language switcher, and RTL layout support are built into the
framework rather than left to themes. See
[Internationalization](/guides/internationalization/).

## Config surface that does not exist

Keys and subsystems a Hugo user may go looking for that are intentionally absent:

- **No Sass, PostCSS, or Tailwind pipeline.** The asset pipeline is esbuild (CSS/JS
  bundling, minification, fingerprinting) plus image processing. Preprocess
  externally if you need a CSS toolchain.
- **No archetypes.** Scaffolding is command-based: `sarde new <collection> <title>`.
- **No content adapters or remote content.** All content comes from Markdown files in
  `content/`; nothing is fetched at build time.
- **No custom output formats.** Sarde emits HTML (plus feeds, sitemap, and search
  index via plugins). There is no JSON/AMP/calendar output layer.
- **Markdown only.** No AsciiDoc, Org Mode, or Pandoc input formats.

## What this page does not cover

Step-by-step conversion lives in the [Migration Guide](/resources/migration-guide/).
The exhaustive field and option lists live in the
[Frontmatter](/reference/frontmatter/) and [Configuration](/reference/configuration/)
references. Known issues and workarounds live in
[Troubleshooting](/resources/troubleshooting/).
