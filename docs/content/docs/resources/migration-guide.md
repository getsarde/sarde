---
title: Migration Guide
description: "Step-by-step patterns for migrating existing content to Sarde from Obsidian and other platforms"
sidebar:
  order: 2
---

Patterns for migrating content to Sarde from other platforms. Each section covers the key differences and the steps to convert existing content.

## From Obsidian

Sarde includes a built-in Obsidian vault importer that converts notes to Sarde-compatible Markdown.

```bash
sarde import obsidian /path/to/vault
```

The importer handles:

| Obsidian syntax | Sarde output |
|-----------------|-------------|
| `[[Page Name]]` wikilinks | `[Page Name](/collection/page-name/)` standard links |
| `[[Page\|Display Text]]` aliased wikilinks | `[Display Text](/collection/page-name/)` |
| `![[image.png]]` image embeds | `![image](assets/image.png)` |
| `> [!note] Title` callouts | `:::note[Title]` asides |
| `%%comment%%` inline comments | Removed |
| `` ```dataview `` blocks | Removed |

### Import options

| Flag | Default | Description |
|------|---------|-------------|
| `-c`, `--collection` | Vault folder name | Target collection name. |
| `--content` | `content` | Content directory path. |

### After importing

1. Review the generated files in `content/<collection>/`.
2. Add frontmatter fields (`title`, `date`, `description`) where the importer did not infer them.
3. Replace any remaining wikilinks that the importer could not resolve (warnings are printed to stderr).
4. Run `sarde dev` and check for broken links with `sarde check-links`.

## From Hugo

Hugo and Sarde share Go templates and YAML frontmatter, so migration is mostly structural.

### Content structure

Move Hugo content into Sarde's `content/` directory. The directory layout is similar: `content/blog/`, `content/docs/`, etc. Sarde auto-detects collections by directory name.

| Hugo | Sarde |
|------|-------|
| `content/posts/` | `content/blog/` (or keep `posts/`, auto-detected as blog) |
| `content/docs/` | `content/docs/` (auto-detected as docs) |
| `_index.md` | `_index.md` (same convention) |
| `archetypes/` | Not used. Use `sarde new` for scaffolding. |

### Frontmatter

Most Hugo frontmatter fields map directly:

| Hugo field | Sarde field | Notes |
|------------|------------|-------|
| `title` | `title` | Same. |
| `date` | `date` | Same format. |
| `draft` | `draft` | Same. |
| `weight` | `sidebar.order` | Sarde uses `sidebar.order` instead of top-level `weight`. |
| `tags` | `tags` | Same. |
| `categories` | `categories` | Same. |
| `aliases` | `aliases` | Same. The redirects plugin generates redirect pages. |
| `description` | `description` | Same. |
| `slug` | `slug` | Same. |
| `layout` | `layout` | Sarde layout names differ. See below. |

### Templates

Sarde uses the same Go `html/template` syntax. Key differences:

| Hugo | Sarde |
|------|-------|
| `{{ .Title }}` | `{{ .Page.Title }}` |
| `{{ .Content }}` | `{{ .Content }}` (same) |
| `{{ .Permalink }}` | `{{ .Page.Permalink }}` |
| `{{ partial "name.html" . }}` | `{{ partial "name.html" . }}` (same) |
| `{{ range .Pages }}` | `{{ range .Collection.Pages }}` |
| `{{ .Site.Title }}` | `{{ .Site.Title }}` (same) |

### Shortcodes

Hugo shortcodes (`{{</* name */>}}`) work in Sarde with the same syntax. Place shortcode templates in `layouts/shortcodes/`. For new content, consider using Sarde's Markdown extensions (`:::aside`, `:::tabs`, etc.) instead.

### Configuration

| Hugo (`config.yaml`) | Sarde (`sarde.yaml`) |
|----------------------|---------------------|
| `baseURL` | `site.url` |
| `title` | `site.title` |
| `languageCode` | `site.language` |
| `theme` | `theme.preset` (or `theme.name` for external themes) |
| `params` | Top-level config sections (`sidebar`, `header`, `footer`, etc.) |
| `menu` | `header.links` for global nav, `nav.yaml` for sidebar |

## From Starlight (Astro)

Starlight uses MDX; Sarde uses standard Markdown with extensions.

### Content

Rename `.mdx` files to `.md`. Remove JSX imports and component syntax:

| Starlight (MDX) | Sarde |
|-----------------|-------|
| `import { Tabs, TabItem } from '@astrojs/starlight/components'` | Remove the import. |
| `<Tabs>` / `<TabItem label="...">` | `::::tabs` / `:::tab[label="..."]` |
| `<Card title="...">` | `:::card[title="..."]` |
| `<Steps>` | `:::steps` |
| `<Aside type="tip">` | `:::tip` |
| `<FileTree>` | `:::file-tree` |
| `<Badge text="..." variant="...">` | `:badge[text]{type="..."}` |

### Frontmatter

| Starlight field | Sarde field |
|----------------|------------|
| `title` | `title` |
| `description` | `description` |
| `sidebar.order` | `sidebar.order` |
| `sidebar.label` | `sidebar.label` |
| `sidebar.hidden` | `sidebar.hidden` |
| `sidebar.badge` | `sidebar.badge` |
| `tableOfContents.minHeadingLevel` | `toc.min_level` |
| `tableOfContents.maxHeadingLevel` | `toc.max_level` |
| `template: splash` | `layout: splash` |
| `hero` | `hero` (similar structure) |

### Configuration

| Starlight (`astro.config.mjs`) | Sarde (`sarde.yaml`) |
|-------------------------------|---------------------|
| `title` | `site.title` |
| `social` | `social` |
| `sidebar` (manual array) | Auto-generated from directory structure, or `nav.yaml` for manual override. |
| `locales` | `i18n.languages` |
| `defaultLocale` | `i18n.default_language` |

## General migration checklist

- [ ] Move content files to `content/` with the appropriate directory structure.
- [ ] Convert frontmatter fields to Sarde equivalents.
- [ ] Replace platform-specific component syntax with Sarde extensions.
- [ ] Create `sarde.yaml` with site metadata (`site.title`, `site.url`, `site.description`).
- [ ] Run `sarde dev` and review the site.
- [ ] Run `sarde check-links` to find broken internal links.
- [ ] Run `sarde validate` to check configuration.
