---
title: Template Functions
description: "Reference for the custom string, math, date, and data functions registered into Sarde's template engine"
sidebar:
  order: 7
---

Sarde registers custom functions into Go's `html/template` engine. Call them with standard template syntax: `{{ funcName arg1 arg2 }}` or pipe an argument with `{{ arg1 | funcName }}`.

```html
{{ range where .Pages "Type" "tutorial" }}
  <h3>{{ .Title }}</h3>
  <p>{{ truncate (plainify .Summary) 120 }}</p>
  <time>{{ dateFormat .Date "January 2, 2006" }}</time>
{{ end }}
```

## Strings

| Function | Signature | Description |
|----------|-----------|-------------|
| `upper` | `upper(s string) string` | Uppercases a string |
| `lower` | `lower(s string) string` | Lowercases a string |
| `title` | `title(s string) string` | Title-cases each word |
| `truncate` | `truncate(s string, n int) string` | Truncates to `n` runes, appending `...` when truncated. Returns a hard cut when `n < 3`. |
| `slugify` | `slugify(s string) string` | Converts to a URL-safe slug. `urlize` is an alias. |
| `replace` | `replace(s, old, new string) string` | Replaces all occurrences of a substring |
| `split` | `split(s, sep string) []string` | Splits a string by separator |
| `join` | `join(list []string, sep string) string` | Joins a string slice with separator |
| `contains` | `contains(s, substr string) bool` | Substring containment check |
| `hasPrefix` | `hasPrefix(s, prefix string) bool` | Prefix check |
| `hasSuffix` | `hasSuffix(s, suffix string) bool` | Suffix check |
| `trim` | `trim(s string) string` | Trims leading and trailing whitespace |
| `markdownify` | `markdownify(v any) HTML` | Renders a value as inline Markdown (GFM). Strips the wrapping `<p>` tag from single-paragraph input. |
| `plainify` | `plainify(s string) string` | Strips all HTML tags, returning plain text |
| `safeHTML` | `safeHTML(s string) HTML` | Marks a string as safe, unescaped HTML |
| `highlight` | `highlight(code, lang string) HTML` | Wraps code in `<pre><code class="language-X">`. This is a lightweight stub, not full syntax highlighting. |

## Math

All math functions accept any numeric type. Results stay as `int` when both operands are integer types, `float64` otherwise. Division and modulo by zero return `0`.

| Function | Signature | Description |
|----------|-----------|-------------|
| `add` | `add(a, b any) any` | Addition |
| `sub` | `sub(a, b any) any` | Subtraction |
| `mul` | `mul(a, b any) any` | Multiplication |
| `div` | `div(a, b any) any` | Division (truncates toward zero for integer operands) |
| `mod` | `mod(a, b any) int` | Integer modulo |

## Logic

| Function | Signature | Description |
|----------|-----------|-------------|
| `cond` | `cond(condition bool, trueVal, falseVal any) any` | Ternary conditional |
| `default` | `default(value, fallback any) any` | Returns `fallback` when `value` is nil or its type's zero value |
| `isset` | `isset(m map[string]any, key string) bool` | Checks key existence in a map |

:::note
`default` takes **value first, fallback second**: `{{ default .Description "No description" }}`. This is the opposite argument order from Hugo's `default` function.
:::

## Collections

| Function | Signature | Description |
|----------|-----------|-------------|
| `first` | `first(n int, list any) any` | First `n` elements of a slice |
| `last` | `last(n int, list any) any` | Last `n` elements |
| `after` | `after(n int, list any) any` | All elements after index `n` |
| `shuffle` | `shuffle(list any) any` | Returns a randomly-shuffled copy |
| `sortBy` | `sortBy(list any, field string) any` | Stable sort by a struct/map field value. Comparison is string-based. |
| `where` | `where(list any, field string, value any) any` | Filters elements where `field` equals `value`. Uses `fmt.Sprint` for comparison. |
| `group` | `group(list any, field string) map[string]any` | Groups elements into a map keyed by field value |
| `uniq` | `uniq(list any) any` | Deduplicates by string representation, preserving first occurrence |
| `in` | `in(list any, value any) bool` | Membership test |
| `seq` | `seq(args ...int) []int` | Generates an integer sequence. `seq(n)` produces `[1..n]`. `seq(start, end)` produces an inclusive range. |

```html
{{ $tutorials := where .Pages "Type" "tutorial" }}
{{ $sorted := sortBy $tutorials "Title" }}
{{ $byCategory := group .Pages "Category" }}
```

## URLs

| Function | Signature | Description |
|----------|-----------|-------------|
| `absURL` | `absURL(path string) string` | Resolves a path to an absolute URL using the site's base URL |
| `relURL` | `relURL(path string) string` | Resolves a path to a relative URL. Passes through paths already containing `://`. |
| `editURL` | `editURL(base, relPath string) string` | Joins a base "edit this page" URL with a relative content path |

## Assets

| Function | Signature | Description |
|----------|-----------|-------------|
| `asset` | `asset(path string) string` | Resolves an asset path to its `/assets/` URL |
| `fingerprint` | `fingerprint(path string) string` | Returns the content-hashed output URL for a path |
| `inline` | `inline(path string) HTML` | Inlines file content. Wraps `.css` in `<style>` and `.js` in `<script>`. |
| `getResource` | `getResource(resources []Resource, name string) *Resource` | Finds a page resource by name |
| `matchResources` | `matchResources(resources []Resource, pattern string) []Resource` | Filters page resources matching a glob pattern |
| `resourcesByType` | `resourcesByType(resources []Resource, mediaType string) []Resource` | Filters page resources by media type |
| `image` | `image(res Resource) HTML` | Renders a `<picture>` element for a resource |
| `resize_image` | `resize_image(res Resource, params string) HTML` | Processes and resizes an image, rendering a responsive `<picture>` with LQIP placeholder |

The `resize_image` `params` string uses `&`-separated key=value pairs.

| Param | Type | Description |
|-------|------|-------------|
| `op` | string | Resize operation |
| `width` | int | Target width in pixels |
| `height` | int | Target height in pixels |
| `quality` | int | Compression quality (1-100) |
| `format` | string | Output format (e.g. `webp`, `avif`) |

```html
{{ $img := getResource .Resources "hero.jpg" }}
{{ resize_image $img "width=800&quality=85&format=webp" }}
```

See [`images`](/reference/configuration#images) for global image processing settings.

## Navigation

| Function | Signature | Description |
|----------|-----------|-------------|
| `navFor` | `navFor(colName string) *NavTree` | Returns the navigation tree for a named collection |
| `breadcrumbs` | `breadcrumbs(data *RouteData) []BreadcrumbItem` | Extracts breadcrumb items from route data. Pass `.` in a page template. |
| `siblings` | `siblings(page *Page) []*Page` | Returns sibling pages within the same section |
| `translations` | `translations(data *RouteData) []TranslationLink` | Extracts translation links from route data. Pass `.` in a page template. |

## i18n

| Function | Signature | Description |
|----------|-----------|-------------|
| `t` | `t(key string) string` | Resolves an i18n string table key for the current language. Returns the key unchanged when no match is found. |
| `tWithData` | `tWithData(key string, data any) string` | Resolves an i18n key with template-interpolation data |
| `lang` | `lang(data *RouteData) string` | Returns the rendering language code from route data |

## Content

| Function | Signature | Description |
|----------|-----------|-------------|
| `ref` | `ref(slug string) string` | Absolute permalink for a page found by slug. Returns the slug unchanged if not found. |
| `relref` | `relref(slug string) string` | Relative permalink for a page found by slug |
| `recentEntries` | `recentEntries(colName string, n int) []*Page` | First `n` pages of a named collection |
| `findEntry` | `findEntry(colName, slug string) *Page` | Finds a page by slug within a named collection |
| `allCollections` | `allCollections() map[string]*Collection` | Returns all site collections |
| `termURL` | `termURL(taxonomyName, termName string) string` | URL for a taxonomy term page |
| `lookupTerm` | `lookupTerm(taxonomyName, termName string) *TaxonomyTerm` | Looks up a taxonomy term by name |
| `showPageTags` | `showPageTags(page *Page) bool` | Whether to show tags for a page (page override, then taxonomy config, then `true`) |
| `topTerms` | `topTerms(taxonomyName string, n int) []*TaxonomyTerm` | Top `n` terms sorted by page count descending |
| `slideCount` | `slideCount(page *Page) int` | Number of slides a page splits into. Returns `params.slide_count` if set, otherwise counts `<hr` separators in rendered HTML + 1. |

The taxonomy functions `termURL`, `lookupTerm`, and `topTerms` are language-aware at render time.

## Icons

| Function | Signature | Description |
|----------|-----------|-------------|
| `icon` | `icon(name string, args ...string) HTML` | Renders an inline SVG icon by name |

The first variadic argument is a CSS class. Additional arguments are attribute key/value pairs.

```html
{{ icon "rocket" }}
{{ icon "rocket" "my-icon-lg" }}
{{ icon "arrow-up" "my-icon" "rotate" "90" "title" "Up" }}
```

See [`icons`](/reference/configuration#icons) for icon set configuration.

## Params and HTML helpers

| Function | Signature | Description |
|----------|-----------|-------------|
| `dict` | `dict(pairs ...any) map[string]any` | Builds a map from alternating key/value arguments. Keys must be strings. |
| `boolVal` | `boolVal(p *bool, def bool) bool` | Dereferences a `*bool`, returning `def` when nil |
| `boolParam` | `boolParam(params map[string]any, key string, def bool) bool` | Reads a bool from a param map, returning `def` when missing |
| `stringParam` | `stringParam(params map[string]any, key string) string` | Reads a string from a param map, returning `""` when missing |
| `renderHeadTags` | `renderHeadTags(tags []HeadTag) HTML` | Renders head tags into `<head>` markup. Filters by allowed tag names. |
| `renderAttrs` | `renderAttrs(attrs map[string]string) HTML` | Renders a map as sorted, escaped HTML attributes |
| `versionOf` | `versionOf(page *Page, versionID string) *Page` | Finds the version peer of a page matching the given version ID |
| `toString` | `toString(v any) string` | Converts any value to its string representation |
| `toInt` | `toInt(v any) int` | Converts a numeric value to `int`. Returns `0` for non-numeric values. |

## Debug and data

| Function | Signature | Description |
|----------|-----------|-------------|
| `dump` | `dump(value any) HTML` | Renders `%+v` representation in a `<pre>` block |
| `jsonify` | `jsonify(value any) HTML` | Pretty-prints a value as indented JSON |
| `printf` | `printf(format string, args ...any) string` | Standard Go string formatting (`fmt.Sprintf`) |
| `data` | `data(name string) any` | Loads a data file from the project's `data/` directory. Accepts `.yaml`, `.yml`, `.json`, or `.toml` (first match wins). Cached per build. |

## Templates

| Function | Signature | Description |
|----------|-----------|-------------|
| `partial` | `partial(name string, data any) HTML` | Renders a cached partial template by name |
| `component` | `component(name string, data any) HTML` | Renders a registered component by name |
| `themeStyles` | `themeStyles(data *RouteData) HTML` | Emits `<link>` and/or `<style>` tags for the token CSS and main CSS bundle. See [Theme Tokens](/reference/theme-tokens). |
| `announcementBanner` | `announcementBanner() HTML` | No-op stub. Overridden by the announcements plugin when enabled. |

## Dates

| Function | Signature | Description |
|----------|-----------|-------------|
| `dateFormat` | `dateFormat(t time.Time, layout string) string` | Formats a time using a Go layout string. Uses Go's reference-date convention (`"January 2, 2006"`), not strftime tokens. |
| `now` | `now() time.Time` | Returns the current time |

## Shortcode availability

Shortcode templates run during parallel Markdown rendering and receive a reduced function map. All functions not listed below work identically in shortcodes.

| Function | Status | Notes |
|----------|--------|-------|
| `component` | No-op | Returns an empty string (component registry unavailable) |
| `partial` | Errors | Returns `"partial cache not initialized"` |
| Plugin functions | Absent | Plugin-registered functions are not merged into the shortcode map |
| `t`, `tWithData` | Degraded | Return the key unchanged (i18n string table unavailable) |
| `absURL`, `relURL` | Degraded | Available but not language-aware (URL resolver unavailable) |
| `themeStyles` | Degraded | Emits inline `<style>` only, not external `<link>` tags |
