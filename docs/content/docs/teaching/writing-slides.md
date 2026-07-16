---
title: Writing Slides
description: "Slide separation syntax, authoring tips, and writing for dual output"
sidebar:
  order: 3
---

Slide decks are standard Markdown files. Horizontal rules (`---`) separate slides.

## Slide separation

Each `---` between blank lines creates a slide break:

````markdown
---
title: Introduction to Algorithms
layout: presentation
---

# Introduction to Algorithms

Course overview and objectives

---

## What is an Algorithm?

A step-by-step procedure for solving a problem.

- Finite number of steps
- Well-defined instructions
- Produces an output

---

## Why Study Algorithms?

Understanding algorithms helps with:

1. Writing efficient code
2. Solving complex problems
3. Passing technical interviews
````

Each section between `---` markers becomes one slide.

:::caution
A `---` separator must have a **blank line above and below it**. A `---` placed directly under text is parsed by Markdown as a setext heading underline (turning the text into a heading), not a horizontal rule.

```markdown
Wrong heading        ← this text becomes an <h2>
---

Correct slide break  ← blank line above

---
```
:::

## Custom slide breakers

By default, slides split on horizontal rules only. Override the breaker elements per deck with `params.slide_breakers`:

```yaml
---
title: "Lecture 5"
params:
  slide_breakers: "hr, h2"
---
```

With `"hr, h2"`, every `## Heading` also starts a new slide. The heading appears as the first element of the new slide.

## Code blocks in slides

Fenced code blocks render with syntax highlighting:

````markdown
---

## HTTP Handler

```go
func handler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "Hello, %s!", r.URL.Path[1:])
}
```
````

Code blocks inside fenced regions are safe. A literal `---` in a code fence does not create a slide break (the renderer HTML-escapes it).

## Tables in slides

Markdown tables render as styled HTML tables:

```markdown
---

## Comparison

| Feature | Go | Rust |
|---------|-----|------|
| GC | Yes | No |
| Compile speed | Fast | Slower |
| Concurrency | Goroutines | async/await |
```

Wide tables scroll horizontally within the slide.

## Writing for dual output

When content serves as both a documentation page and a slide deck:

- Use `##` for slide titles. These work as section headings in docs mode and slide titles in presentation mode.
- Keep bullet points concise. Each point should be readable on a projected screen.
- Place `---` separators at natural break points that also make sense as section dividers in the docs view.
- Use code blocks freely. Syntax highlighting works identically in both modes.
- Avoid deeply nested content (4+ indent levels). It reads well in docs but overflows on slides.
- Complex tables and detailed diagrams may need separate treatment for each mode.

## Slide count

The gallery card badge shows the number of slides in each deck. The count is auto-computed by counting `<hr>` elements in the rendered HTML plus one. Override it with frontmatter:

```yaml
---
title: "Dense Lecture"
params:
  slide_count: 42
---
```

The `slideCount` template function is available in all templates for custom gallery layouts.
