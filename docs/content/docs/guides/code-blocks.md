---
title: Code Blocks
description: "Set up syntax highlighting, titles, line markers, copy controls, and grouped tabs for code blocks"
sidebar:
  order: 9
---

Code blocks render with syntax highlighting, optional titles, line markers,
copy controls, and grouped tabs. Most features are set in the opening fence.

````markdown
```go title="main.go" {3-5}
package main

import "fmt"

func main() {
    fmt.Println("Hello, world!")
}
```
````

Result: A framed Go code block appears with a `main.go` title and lines 3 through 5
highlighted.

## Highlighting engines

Sarde embeds Kazari for code block presentation. Kazari can use Nuri or Chroma
for syntax highlighting.

`sarde.yaml`

```yaml
markdown:
  codeblocks:
    engine: nuri
```

| Engine | Use for |
|---|---|
| `nuri` | VS Code-style TextMate grammar highlighting. |
| `chroma` | Go-native highlighting with faster startup and fewer dependencies. |

Set the language after the opening fence.

````markdown
```python
def greet(name):
    return f"Hello, {name}!"
```
````

## Titles and frames

Add a title with `title`.

````markdown
```yaml title="sarde.yaml"
site:
  title: "Biology Course"
```
````

Result: A title bar labeled `sarde.yaml` appears above the block.

Frames control the visual chrome around a code block.

| Frame | Description |
|---|---|
| `code` | Standard code block with language badge. |
| `terminal` | Terminal-style frame for shell commands. |
| `none` | Code content without surrounding chrome. |

Override the frame when needed.

````markdown
```sh frame=none
sarde build
```
````

## Line numbers and highlights

Use `showLineNumbers` for a line-number gutter. Use `startLineNumber` when the
snippet starts in the middle of a larger file.

````markdown
```js showLineNumbers startLineNumber=10
const sunlight = true;
const water = true;
```
````

Result: Line numbers appear in the gutter and start at 10.

Highlight lines with curly braces.

````markdown
```python {3,5-7}
import csv

def read_observations():
    rows = load_csv("plants.csv")
    cleaned = remove_empty_rows(rows)
    validated = check_schema(cleaned)
    return validated
```
````

Result: Line 3 and lines 5 through 7 are highlighted.

Add labels to highlighted ranges when the reason matters.

````markdown
```python {"Input":1-2} {"Validation":4-6}
data = read_file("plants.csv")
records = parse_csv(data)

cleaned = remove_empty_rows(records)
validated = check_schema(cleaned)
result = transform(validated)
```
````

## Diff and inline markers

Use `ins` and `del` to show inserted and deleted lines.

````markdown
```yaml ins={3} del={2}
site:
  title: "Old Course"
  title: "Biology Lab"
```
````

Result: Deleted lines render in red and inserted lines render in green.

Inline markers highlight words or patterns within a line.

````markdown
```python "photosynthesis" /plant_\w+/
result = photosynthesis_rate(plant_sample)
```
````

Result: The exact word and matching pattern are highlighted inside the line.

## Focus and collapse

Use `focus` to dim surrounding lines.

````markdown
```python focus={3-4}
import os

def main():
    run_lesson_export()

if __name__ == "__main__":
    main()
```
````

Result: Lines 3 and 4 stay fully visible. The other lines are dimmed.

Use `collapse` for long examples.

````markdown
```python collapse
import csv
import json

def load_many_lesson_files():
    pass
```
````

Result: The block renders collapsed with a control to expand it.

## Code groups

Group related code blocks with `:::code-group`.

````markdown
:::code-group

```bash title="npm"
npm run build
```

```bash title="pnpm"
pnpm build
```

:::
````

Result: A tabbed code block appears. Selecting a tab switches between package
managers.

## Common options

| Option | Syntax | Description |
|---|---|---|
| Language | <code>```go</code> | Selects syntax highlighting. |
| Title | `title="main.go"` | Adds a title bar. |
| Line numbers | `showLineNumbers` | Shows a gutter with line numbers. |
| Start number | `startLineNumber=10` | Starts line numbering at a custom value. |
| Highlight lines | `{3,5-7}` | Highlights individual lines or ranges. |
| Insert/delete | `ins={3} del={2}` | Marks inserted and deleted lines. |
| Inline marker | `"text"` or `/regex/` | Highlights matching text inside a line. |
| Focus | `focus={3-4}` | Dims lines outside the focused range. |
| Collapse | `collapse` | Renders the block collapsed by default. |
| Theme | `theme="one-dark-pro"` | Overrides the highlighting theme for one block. |
| Frame | `frame=terminal` | Sets the visual frame. |

## Configuration

Configure engine and themes in `sarde.yaml`.

`sarde.yaml`

```yaml
markdown:
  codeblocks:
    engine: nuri
    light_theme: github-light
    dark_theme: github-dark
    dark_mode_selector: '[data-theme="dark"]'
```

Configure Kazari controls in `kazari.config.yaml`.

`kazari.config.yaml`

```yaml
copyButton: true
fullscreenButton: true
wrapButton: true
languageBadge: true
lineNumbers: false

defaults:
  wrap: false
  frame: auto
```

See [Configuration](/reference/configuration#markdown) for
`markdown.codeblocks` settings.
