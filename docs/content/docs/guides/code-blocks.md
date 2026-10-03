---
title: Code Blocks
description: "Set up syntax highlighting, titles, line markers, copy controls, and grouped tabs for code blocks"
sidebar:
  order: 10
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

→ A framed Go code block appears with a `main.go` title and lines 3 through 5
highlighted.

## Highlighting engines

Sarde embeds Kazari for code block presentation. Kazari can use Nuri or Chroma
for syntax highlighting. Select the engine in `sarde.yaml`:

```yaml title="sarde.yaml"
markdown:
  codeblocks:
    engine: nuri
```

| Engine | Use for |
|---|---|
| `nuri` (default) | VS Code-style TextMate grammar highlighting. |
| `chroma` | Go-native highlighting that builds faster, which suits dev and live reload. |

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

→ A title bar labeled `sarde.yaml` appears above the block.

Frames control the visual chrome around a code block.

| Frame | Description |
|---|---|
| `auto` (default) | Uses `terminal` for shell languages and `code` for everything else. |
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

→ Line numbers appear in the gutter and start at 10.

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

→ Line 3 and lines 5 through 7 are highlighted.

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

→ Deleted lines render in red and inserted lines render in green.

Inline markers highlight words or patterns within a line.

````markdown
```python "photosynthesis" /plant_\w+/
result = photosynthesis_rate(plant_sample)
```
````

→ The exact word and matching pattern are highlighted inside the line.

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

→ Lines 3 and 4 stay fully visible. The other lines are dimmed.

Use `collapse` with a line range to fold part of a long example behind an expandable summary. Ranges count the lines of the code block, starting at 1.

````markdown
```python collapse={1-3}
import csv
import json
import os

def export_lessons():
    pass
```
````

→ Lines 1 through 3 render as a collapsed summary row. Selecting it reveals them; the rest of the block stays visible.

Separate several ranges with commas, for example `collapse={1-3,8-12}`. A revealed section stays open unless you add `collapseStyle`: `collapsible-start` puts the summary row above the revealed lines, `collapsible-end` below, and `collapsible-auto` picks based on where the range sits.

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

→ A tabbed code block appears. Selecting a tab switches between package
managers.

## Common options

The opening fence accepts these options:

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
| Collapse | `collapse={1-3}` | Folds a line range behind an expandable summary. |
| Theme | `theme="one-dark-pro"` | Overrides the highlighting theme for one block. |
| Frame | `frame=terminal` | Sets the visual frame. |

## Configuration

Two files configure code blocks. `sarde.yaml` sets the engine and the highlighting themes. `kazari.config.yaml` at the project root sets Kazari's toolbar, frame, and styling options. `sarde new site` creates both.

```yaml title="sarde.yaml"
markdown:
  codeblocks:
    engine: nuri
    light_theme: github-light
    dark_theme: github-dark
    dark_mode_selector: '[data-theme="dark"]'
```

| Key | Type | Default | Description |
|---|---|---|---|
| `engine` | string | `nuri` | `nuri` or `chroma`. |
| `light_theme` | string | `github-light` | Highlighting theme in light mode. |
| `dark_theme` | string | `github-dark` | Highlighting theme in dark mode. |
| `dark_mode_selector` | string | `[data-theme="dark"]` | CSS selector that marks dark mode. The default matches Sarde's theme toggle. |

A `kazari.config.yaml` that sets the toolbar buttons and the default frame:

```yaml title="kazari.config.yaml"
copyButton: true
fullscreenButton: true
wrapButton: true
languageBadge: true
lineNumbers: false

defaults:
  wrap: false
  frame: auto
```

:::caution
`kazari.config.yaml` wins over `sarde.yaml` for any option both define, with one exception: dark mode. The file that `sarde new site` creates sets `themes`, so `light_theme` and `dark_theme` in `sarde.yaml` have no effect until you remove that entry; change `themes.light` and `themes.dark` in `kazari.config.yaml` instead. Dark mode always comes from `dark_mode_selector` in `sarde.yaml`, so code blocks switch with Sarde's theme toggle. A `darkMode` entry in `kazari.config.yaml` is ignored, and the build warns about it.
:::

See [Configuration](/reference/configuration/content/#markdown-codeblocks) for the full `markdown.codeblocks` reference. The Kazari documentation lists every `kazari.config.yaml` option and meta string token: [Kazari docs](https://frostybee.github.io/kazari).
