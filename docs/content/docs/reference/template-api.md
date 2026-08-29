---
title: Template API
description: "How templates receive data, how the base shell is filled, and where each part of the template surface is documented"
sidebar:
  order: 5
---

A Sarde template is a Go `html/template` file that receives one value, the route data for the page being rendered, and fills the `content` block of a base shell. This page maps the pieces; the field-by-field and function-by-function detail lives on the pages it links to.

## What a template receives

The dot context (`.`) is a `*RouteData`. Everything a template can read hangs off it: the page (`.Page`), the site (`.Site`), the collection (`.Collection`), navigation (`.Sidebar`, `.Breadcrumbs`, `.Pagination`, `.Paginator`), language (`.Lang`, `.Dir`, `.Translations`), versioning, tabs, labs, and plugin-injected assets. See [Route Data](/reference/route-data/) for every field.

Inside `range` and `with`, the dot moves to the current element. `$` still points at the route data:

```html
{{ range .AllTranslations }}
<a href="{{ .URL }}"{{ if eq .Lang $.Lang }} aria-current="true"{{ end }}>{{ .Name }}</a>
{{ end }}
```

→ Each language link, with the one matching the page's own language marked current.

The shipped theme needs `$` in only a handful of places; if a template reaches for it often, the loop is probably nested one level too deep.

## The base shell

Each layout family has a `baseof.html` that renders the HTML document, header, footer, and scripts, and reserves one block for the page:

```html
{{ block "content" . }}{{ .Page.Content }}{{ end }}
```

A page template overrides that block and nothing else:

```html
{{ define "content" }}
<article>
  <h1>{{ .Page.Title }}</h1>
  {{ .Page.Content }}
</article>
{{ end }}
```

→ The page body renders inside the shell's `<main>`, with the theme's header and footer around it.

Which `baseof.html` and which page template are chosen for a given page is decided by the lookup chain in [Layouts and Templates](/customization/layouts-and-templates/#template-overlay-resolution). The resolved name is exposed as `.Template`.

## Components, partials, and functions

| Unit | Call | Owner | Missing file | Reference |
|------|------|-------|--------------|-----------|
| Component | `{{ component "Header" . }}` | Engine, overridable | Renders nothing | [UI Components](/reference/ui-components/) |
| Partial | `{{ partial "home-hero.html" . }}` | Theme | Template error | [Layouts and Templates](/customization/layouts-and-templates/#partials) |
| Function | `{{ relURL .URL }}`, `{{ t "nav.skip_to_content" }}` | Engine | Parse error | [Template Functions](/reference/template-functions/) |
| Shortcode | `{{< alert >}}` in Markdown | Theme or site | Build error | [Shortcodes](/reference/shortcodes/) |

Components and partials receive whatever value is passed as the second argument, conventionally `.` so they see the same route data as the caller.

## What is available where

Several route data fields are only populated for some pages. The table lists the ones that vary; everything not listed is either always set or depends on configuration rather than layout.

| Page | `.Sidebar`, `.Breadcrumbs` | `.Paginator` | `.Pagination` | `.Homepage` | `.Taxonomy` | `.TaxonomyTerm`, `.TermEntries` |
|------|---------------------------|--------------|---------------|-------------|-------------|--------------------------------|
| `docs`, `wide`, `labs` page | yes | no | if a sibling exists | no | no | no |
| `docs`, `wide`, `labs` `_index.md` | yes | if paginated | if a sibling exists | no | no | no |
| `default` and other layouts | no | no | if a sibling exists | no | no | no |
| Collection `_index.md` on a non-sidebar layout | no | if paginated | if a sibling exists | no | no | no |
| Home page | no | no | no | yes | no | no |
| Taxonomy list page | no | no | no | no | yes | `.TermEntries` |
| Taxonomy term page | no | yes | no | no | yes | `.TaxonomyTerm` |

Versioning fields (`.Version`, `.Versions`, `.IsLatest`, `.VersionBanner`) are set only in versioned collections; tab fields (`.IsTabbed`, `.DocsTabs`, `.ActiveTab`) only on sidebar layouts of tabbed collections; lab fields only in labs collections.

## Plugin-injected assets

`.Scripts`, `.Styles`, `.InlineScripts`, and `.ModuleScripts` start empty. Plugins append to them in the `BeforeRender` hook, and the `Scripts` and `Head` components emit them. A custom `baseof.html` that drops those components also drops every plugin's client-side code. See [Writing Plugins](/advanced/writing-plugins/#beforerender).

## Common mistakes

| Mistake | What happens | Fix |
|---------|--------------|-----|
| `{{ .Paginator.Current }}` on a page with no paginator | Render error: nil pointer | Wrap in `{{ with .Paginator }}` |
| `{{ if .Page.Date }}` | Always true; a zero date prints as year 1 | `{{ if not .Page.Date.IsZero }}` |
| `{{ range .Paginator.Pages }}` expecting posts | Ranges over numbered links | Use `.Paginator.CurrentPages` |
| Recursing through `.Parent` or `.Collection` on a `Section` or `NavNode` | Infinite loop | Walk `Children` and `Sections` downward only |
| `{{ .Site.Config.Fotter.Text }}` | Render error, not a config error: `Config` is untyped until render | Check the section name against [Configuration](/reference/configuration/) |
| `{{ .Sidebar.Label }}` | No such field; `.Sidebar` is the nav tree | Page sidebar settings are `.Page.Sidebar.Label` |
