---
title: Creating a Theme
description: "Build a Sarde theme from scratch with templates, tokens, and presets"
sidebar:
  order: 4
---

This guide builds a `themes/magazine/` directory with custom blog post and list templates, design tokens, and a bundled preset, ready to install on any Sarde site.

## Two ways to start

1. Eject only what changes: `sarde theme eject --name magazine layouts/_blog/single.html` copies one template (plus a `theme.yaml`) into `themes/magazine/`. Everything not copied keeps coming from the embedded theme, including fixes in later Sarde releases. This guide takes that route.
2. Start from a full copy: `sarde theme eject --name magazine` copies every embedded file. Each copy stops receiving upstream changes the moment it is written, so delete the files left unmodified before publishing. This only makes sense when re-skinning the stylesheets and assets wholesale.

What the build does when a file is missing from the theme:

| Directory | Behaviour |
|-----------|-----------|
| `layouts/**` | Per file. A template, component, partial, or shortcode not in the theme is read from the embedded theme |
| `css/` | Per stylesheet. Any of the 24 stylesheets not in the theme is read from the embedded theme |
| `assets/` | Whole directory. If the theme has an `assets/` directory, the embedded fonts, vendor scripts, and JS are not emitted |
| `theme.yaml` | Required. Eject adds it whenever the destination has none |

## Create the theme

Create the manifest:

```
themes/magazine/theme.yaml
```

```yaml
name: "Magazine"
slug: "magazine"
version: "1.0.0"
author: "Sarde Themes"
description: "A bold magazine-style theme for blog collections"
license: "MIT"
```

No field is required, but `name` and `slug` are what `sarde theme list` and `sarde theme info` display.

Activate the theme in `sarde.yaml` and start the dev server:

```yaml
theme:
  name: "magazine"
```

```sh
sarde dev
```

→ The site looks unchanged. The theme is active, and every file added under `themes/magazine/` from here on takes effect immediately.

## Override the blog post template

Create the single-page template for blog posts:

```
themes/magazine/layouts/_blog/single.html
```

```go
{{ define "content" }}
<article class="magazine-post">
  {{ if .Page.Image }}
  <img class="magazine-hero" src="{{ .Page.Image }}" alt="{{ .Page.Title }}">
  {{ end }}
  <header class="magazine-header">
    <h1>{{ .Page.Title }}</h1>
    {{ if .Page.Description }}<p class="magazine-subtitle">{{ .Page.Description }}</p>{{ end }}
    <div class="magazine-meta">
      {{ if not .Page.Date.IsZero }}
      <time datetime="{{ .Page.Date.Format "2006-01-02" }}">{{ .Page.Date.Format "January 2, 2006" }}</time>
      {{ end }}
      {{ if .Page.ReadingTime }}<span>{{ .Page.ReadingTime }} min read</span>{{ end }}
    </div>
  </header>
  {{ .Page.Content }}
  {{ component "PageTags" . }}
</article>
{{ end }}
```

→ Every blog post renders with a full-width cover image, a subtitle, the date, and reading time above the body.

The `{{ define "content" }}` block fills the base shell (`baseof.html`), which supplies the header, footer, and scripts. Templates in `_blog/` apply to every blog-type collection (`blog`, `posts`, `articles`, `news`). See [Layouts and Templates](/customization/layouts-and-templates/) for the lookup order and [Route Data](/reference/route-data/) for the fields available as `.`.

To start from the embedded template instead of a blank file, eject it and edit the copy:

```sh
sarde theme eject --name magazine layouts/_blog/single.html
```

## Override the blog list template

Create the listing template:

```
themes/magazine/layouts/_blog/list.html
```

```go
{{ define "content" }}
<div class="magazine-listing">
  <h1>{{ .Page.Title }}</h1>
  {{ $pages := .Collection.Pages }}
  {{ if and .Paginator .Paginator.CurrentPages }}{{ $pages = .Paginator.CurrentPages }}{{ end }}
  <div class="magazine-grid">
    {{ range $pages }}
    <a href="{{ .Permalink }}" class="magazine-card">
      {{ if .Image }}<img src="{{ .Image }}" alt="{{ .Title }}">{{ end }}
      <h2>{{ .Title }}</h2>
      {{ if .Description }}<p>{{ .Description }}</p>{{ end }}
      {{ if not .Date.IsZero }}<time datetime="{{ .Date.Format "2006-01-02" }}">{{ .Date.Format "Jan 2, 2006" }}</time>{{ end }}
    </a>
    {{ end }}
  </div>
  {{ if .Paginator }}
  <nav class="magazine-pagination">
    {{ if .Paginator.HasPrev }}<a href="{{ .Paginator.PrevURL }}" rel="prev">Newer</a>{{ end }}
    {{ if .Paginator.HasNext }}<a href="{{ .Paginator.NextURL }}" rel="next">Older</a>{{ end }}
  </nav>
  {{ end }}
</div>
{{ end }}
```

→ The blog index renders as a card grid with cover images, titles, and dates, followed by Newer and Older links when the collection is paginated.

`.Collection.Pages` holds every page in the collection, already sorted. When `collections.<name>.paginate` is set, `.Paginator.CurrentPages` holds the slice for the current page.

## Set the design tokens

Add `tokens` and `dark_tokens` to `theme.yaml`:

```yaml
tokens:
  accent: "oklch(0.58 0.22 15)"
  font-sans: "'Playfair Display', Georgia, serif"
  font-mono: "'Fira Code', monospace"
  radius-md: "0"
  radius-lg: "0"

dark_tokens:
  bg: "oklch(0.13 0.01 15)"
  bg-surface: "oklch(0.18 0.01 15)"
  text: "oklch(0.93 0.005 15)"
```

→ The site picks up a warm accent, serif headings, and square corners. Dark mode uses deep warm backgrounds.

Each token becomes a `--sd-*` CSS custom property. Setting `accent` also derives the hover, high, low, and text variants; see [Accent color derivation](/guides/themes-and-styling/#accent-color-derivation). See [Theme Tokens](/reference/theme-tokens/) for every token name.

## Bundle a preset

A preset is a named token set the site owner can switch to without editing tokens. Add one to `theme.yaml`:

```yaml
presets:
  editorial:
    name: "Editorial"
    tokens:
      accent: "oklch(0.45 0.03 250)"
      font-sans: "'Source Serif 4', Georgia, serif"
    dark_tokens:
      bg: "oklch(0.12 0.02 250)"
```

A site activates it in `sarde.yaml`:

```yaml
theme:
  name: "magazine"
  preset: "editorial"
```

→ The accent shifts to a muted blue and body text switches to Source Serif.

A preset can set any token the theme can. See [Creating a Preset](/customization/creating-a-preset/) for palette and full-look recipes and how light and dark values combine.

## Override a component

Components are the structural pieces the base shell calls with `{{ component "Name" . }}`. Replace one by adding a file with the same name:

```
themes/magazine/layouts/components/Footer.html
```

```go
<footer class="sarde-footer">
  <div class="sarde-footer-inner">
    {{ component "Social" . }}
    {{ range .Site.Config.Footer.Links }}
    <a href="{{ if .External }}{{ .URL }}{{ else }}{{ relURL .URL }}{{ end }}">{{ .Label }}</a>
    {{ end }}
    {{ if .Site.Config.Footer.Text }}<p>{{ .Site.Config.Footer.Text }}</p>{{ end }}
  </div>
</footer>
```

→ Social icons render first, then the configured footer links and text.

See [UI Components](/reference/ui-components/) for every component and the data each one reads.

## Replace a stylesheet (optional)

Tokens cover colors, fonts, spacing, and radii. For layout changes the tokens cannot express, eject the stylesheet that owns the rules and edit it:

```sh
sarde theme eject --name magazine css/blog.css
```

→ `themes/magazine/css/blog.css` is used in place of the embedded copy. The other 23 stylesheets still come from the embedded theme.

:::caution
`assets/` does not fall back per file. A theme `assets/` directory replaces the embedded fonts, vendor scripts, and JS entirely, so eject it whole (`sarde theme eject --name magazine assets`) and keep every file the stylesheets reference.
:::

For a handful of rules, skip the theme stylesheet and let site owners use `head.custom_css` or the `@layer sarde.user` block instead; see [CSS layer order](/guides/themes-and-styling/#css-layer-order).

## Iterate with the dev server

`sarde dev` watches `themes/`. Edits to `.css` files hot-swap in the browser without a reload; edits to templates or `theme.yaml` trigger a rebuild and reload.

To develop a theme that lives outside the site directory, point the dev server at it:

```sh
sarde dev --theme-dev ../magazine
```

→ Changes in `../magazine` trigger the same hot-swap and rebuild behavior.

## Check and share

Confirm the manifest loads:

```sh
sarde theme info magazine
```

```
Name:          Magazine
Slug:          magazine
Version:       1.0.0
Author:        Sarde Themes
License:       MIT
Description:   A bold magazine-style theme for blog collections
Tokens:        5
Dark tokens:   3
Presets:       editorial
```

The finished theme:

```
themes/magazine/
  theme.yaml
  layouts/
    _blog/
      single.html
      list.html
    components/
      Footer.html
```

Publish the directory as a GitHub repository, zip, or tar.gz. Site owners install it with `sarde theme add github.com/sarde-themes/magazine`; see [Using Themes](/customization/using-themes/).
