---
title: Layouts and Templates
description: "Understand page layout types, template lookup order, components, and partials"
sidebar:
  order: 1
---

Layouts control the page chrome: whether a sidebar, table of contents, or full-width content area appears. Set the layout per page in frontmatter or per collection in `sarde.yaml`. Templates render the HTML for each layout using Go's `html/template` engine.

## Layout types

Sarde ships eight layout types. Each determines which structural elements appear on the page.

| Layout | Sidebar | TOC | Content width | Use for |
|--------|---------|-----|---------------|---------|
| `default` | no | no | Standard | Blog posts, standalone pages |
| `docs` | yes | yes | Standard | Documentation, courses, tutorials |
| `splash` | no | no | Full width | Landing pages, homepages |
| `wide` | yes | no | Wider | Media-heavy docs pages |
| `full` | no | no | Full width | Custom pages, dashboards |
| `centered` | no | no | Narrow | About pages, legal pages |
| `split` | no | no | Two equal columns | Comparison pages |
| `presentation` | no | no | Full width | Full-screen slide decks. See [Presentation Layout](/teaching/presentation-layout/) |

Set the layout in frontmatter:

```yaml
---
title: About
layout: centered
---
```

Or set it for an entire collection in `sarde.yaml`:

```yaml
collections:
  docs:
    layout: docs
```

Blog collections default to `default`. Docs collections default to `docs`. All other collections default to `default`.

## Template overlay resolution

When Sarde renders a page, it searches for the template file through a multi-layer lookup chain. The first match wins (most specific takes precedence):

### Default-layout pages (5 layers)

1. `layouts/<collection>/single.html` (user, collection-specific)
2. `themes/<theme>/layouts/<collection>/single.html` (theme, collection-specific)
3. `layouts/_default/single.html` (user, default)
4. `themes/<theme>/layouts/_default/single.html` (theme, default)
5. Embedded `_default/single.html` (compiled into binary)

### Docs-layout pages (8 layers)

Docs-layout pages insert `_docs/` layers between the collection and default layers:

1. `layouts/<collection>/single.html`
2. `themes/<theme>/layouts/<collection>/single.html`
3. `layouts/_docs/single.html`
4. `themes/<theme>/layouts/_docs/single.html`
5. `layouts/_default/single.html`
6. `themes/<theme>/layouts/_default/single.html`
7. Embedded `_docs/single.html`
8. Embedded `_default/single.html`

### Blog-layout pages (8 layers)

Blog-layout pages follow the same pattern, inserting `_blog/` type layers:

1. `layouts/<collection>/single.html` (user, collection-specific)
2. `themes/<theme>/layouts/<collection>/single.html` (theme, collection-specific)
3. `layouts/_blog/single.html` (user, blog-type)
4. `themes/<theme>/layouts/_blog/single.html` (theme, blog-type)
5. `layouts/_default/single.html` (user, default)
6. `themes/<theme>/layouts/_default/single.html` (theme, default)
7. Embedded `_blog/single.html`
8. Embedded `_default/single.html`

The `_blog/` type directory activates for any collection named `blog`, `posts`, `articles`, or `news`. Collection-name directories (e.g. `layouts/posts/`) target only that specific collection.

See [Blog and Taxonomies](/guides/blog-and-taxonomies/) for the shipped blog template variants.

## Partials

Partials are reusable template fragments. The resolution order (3 layers, first match wins):

1. `layouts/partials/<name>.html` (user)
2. `themes/<theme>/layouts/partials/<name>.html` (theme)
3. Embedded `partials/<name>.html`

Call a partial from a template:

```go
{{ partial "sidebar-extra.html" . }}
```

Override any embedded partial by placing a file with the same name in `layouts/partials/`.

## Components vs. partials

Sarde distinguishes two kinds of reusable template units:

- *Components* are engine-owned. Called with `{{ component "Name" . }}`, they render structural layout elements (Header, Sidebar, Footer, Search) with safe defaults. If a component is missing, the engine renders nothing rather than erroring.

- *Partials* are theme-owned. Called with `{{ partial "name.html" . }}`, they render content-level fragments (hero sections, blog cards, social links). A missing partial produces a template error.

Override a component by placing a file at `layouts/components/<Name>.html`. See [UI Components](/reference/ui-components) for the full list of overridable components.

## Custom templates

Create a custom template by placing a file in `layouts/` that defines the `"content"` block. For example, a magazine-style blog post variant at `layouts/_blog/single-magazine.html`:

```go
{{ define "content" }}
<article class="blog-magazine">
  <header class="magazine-header">
    {{ if .Page.Image }}
      <img src="{{ .Page.Image }}" alt="{{ .Page.Title }}">
    {{ end }}
    <h1>{{ .Page.Title }}</h1>
    <p class="magazine-subtitle">{{ .Page.Description }}</p>
  </header>
  <div class="magazine-body">
    {{ .Page.Content }}
  </div>
</article>
{{ end }}
```

→ A magazine-style layout with a large header image, title, and subtitle above the post body.

Templates receive a `RouteData` object as their context (`.`), which includes `.Page`, `.Site`, `.Sidebar`, `.Breadcrumbs`, `.Translations`, and other layout data. See [Route Data](/reference/route-data/) for every field and [Template API](/reference/template-api/) for how the pieces fit together.

Assign the template to a post in frontmatter. The `template` field is a path relative to the layouts directory, without `.html`:

```yaml
---
title: "Introduction to Photosynthesis"
template: blog/single-magazine
---
```

Placing the file in `_blog/` makes it available to all blog-type collections (`blog`, `posts`, `articles`, `news`). To target only a collection named `posts`, place it in `layouts/posts/single-magazine.html` instead.

## Template naming conventions

Template variant files use a hyphenated suffix: `{kind}-{variant}.html`.

| Pattern | Example | Meaning |
|---------|---------|---------|
| `single.html` | `_blog/single.html` | Default single-page template for the type |
| `single-{variant}.html` | `_blog/single-cover.html` | Named variant of the single-page template |
| `list.html` | `_blog/list.html` | Default list template for the type |
| `list-{variant}.html` | `_blog/list-grid.html` | Named variant of the list template |

Underscore-prefixed directories (`_blog/`, `_docs/`, `_default/`) are shared type directories; non-prefixed directories (`blog/`, `posts/`) are collection-specific and take priority. The `template` frontmatter field omits the `.html` extension; the engine appends it during resolution.

See [Template Functions](/reference/template-functions) for all available functions, and [UI Components](/reference/ui-components) for the component registry.
