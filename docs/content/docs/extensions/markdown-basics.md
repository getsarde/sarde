---
title: Markdown Basics
description: "A visual reference of all standard Markdown formatting supported by Sarde."
sidebar:
  order: 98
---

This page demonstrates every standard Markdown formatting option supported by Sarde. For custom extensions (admonitions, cards, tabs, and more), see the [Kitchen Sink](/docs/extensions/kitchen-sink/).

---

## Headings

Headings from `h1` through `h6` are constructed with a `#` for each level:

# Heading 1
## Heading 2
### Heading 3
#### Heading 4
##### Heading 5
###### Heading 6

---

## Paragraphs

A paragraph is one or more consecutive lines of text, separated by one or more blank lines.

Here is a second paragraph. Notice how a blank line separates them. Paragraphs can span multiple lines in the source and still render as a single block of text.

That last point matters when wrapping prose at a fixed width: a single newline inside a paragraph is a soft break and renders as a space, so the text reflows to the reader's screen width. Only a blank line starts a new paragraph.

To force a line break inside a paragraph, end the line with two spaces or a backslash:

```markdown
First line  
second line, on its own row.
```

To make every newline break the line instead, set [`markdown.hard_wraps`](/reference/configuration#markdown) to `true`.

---

## Inline Text Formatting

The following inline formatting options are available:

- **Bold text** using `**double asterisks**`
- *Italic text* using `*single asterisks*`
- ***Bold and italic*** using `***triple asterisks***`
- ~~Strikethrough~~ using `~~double tildes~~`
- `Inline code` using `` `backticks` ``
- [A hyperlink](https://example.com) using `[text](url)`

---

## Horizontal Rule

Three or more dashes produce a horizontal rule:

---

## Blockquotes

> This is a blockquote. It can span multiple lines and supports **bold**, *italic*, and `inline code` inside it.
>
> > Blockquotes can also be nested. This is a nested blockquote.
>
> Back to the first level.

---

## Unordered Lists

- Item one
- Item two
  - Nested item A
  - Nested item B
    - Deeply nested item
- Item three

---

## Ordered Lists

1. First item
2. Second item
   1. Nested item one
   2. Nested item two
3. Third item

---

## Task Lists

- [x] Write the markdown basics page
- [x] Add all standard formatting
- [ ] Review the rendered output
- [ ] Ship it

---

## Links

- [Inline link](https://example.com)
- [Link with title](https://example.com "Example Title")
- Autolink: <https://example.com>

---

## Images

Standard markdown image syntax:

![Placeholder image](https://placehold.co/600x200/1a1a2e/e0e0e0?text=Standard+Image)

---

## Tables

| Feature         | Status      | Priority |
| --------------- | ----------- | -------- |
| Markdown        | Done        | High     |
| Extensions      | Done        | High     |
| Theme support   | In progress | Medium   |
| Plugin system   | Planned     | Low      |

Tables support alignment:

| Left Aligned | Center Aligned | Right Aligned |
| :----------- | :------------: | ------------: |
| Row 1        |    Center 1    |         $10   |
| Row 2        |    Center 2    |        $200   |
| Row 3        |    Center 3    |      $3,000   |

---

## Code Blocks

Inline code: `const x = 42`

Fenced code block with syntax highlighting:

```go
package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, Sarde!")
	})
	http.ListenAndServe(":8080", nil)
}
```

```javascript
import { create, search } from '@orama/orama'

const db = await create({
  schema: {
    title: 'string',
    content: 'string',
    url: 'string',
  },
})

const results = await search(db, {
  term: 'getting started',
  limit: 10,
})
```

```yaml
site:
  title: "My Documentation"
  description: "Built with Sarde"
  url: "https://docs.example.com"
  language: "en"

build:
  minify: true
  drafts: false
```

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>{{ .Site.Title }}</title>
</head>
<body>
  {{ template "header" . }}
  <main>{{ .Content }}</main>
  {{ template "footer" . }}
</body>
</html>
```

```css
:root {
  --color-primary: #6366f1;
  --color-surface: #1a1a2e;
  --font-body: 'Inter', system-ui, sans-serif;
}

.aside--note {
  border-left: 4px solid var(--color-primary);
  background: color-mix(in srgb, var(--color-primary) 10%, transparent);
  padding: 1rem 1.5rem;
  border-radius: 0.5rem;
}
```

```bash
sarde build --minify
rsync -avz dist/ deploy@server:/var/www/site/
echo "Deployed successfully!"
```

---

## Footnotes

Here is a sentence with a footnote[^1], and here is another[^2].

[^1]: This is the first footnote with additional context.
[^2]: Footnotes can contain **markdown** formatting and `inline code`.

---

## Definition Lists

Sarde
: A zero-config, Go-based static site generator.

Goldmark
: A CommonMark-compliant Markdown parser written in Go.
