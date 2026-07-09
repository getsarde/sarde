---
title: Code Blocks
sidebar:
  order: 4
---

Sarde renders fenced code blocks with syntax highlighting, line markers, titles, copy buttons, and more. Features are controlled through meta-string options on the opening fence.

````markdown
```go title="main.go" {3-5}
package main

import "fmt"

func main() {
    fmt.Println("Hello, world!")
}
```
````

→ A framed code block with the title "main.go", Go syntax highlighting, and lines 3 through 5 highlighted.

## Highlighting engines

Sarde ships two highlighting engines. Set the engine in `sarde.yaml`:

```yaml
markdown:
  codeblocks:
    engine: nuri    # or "chroma"
```

| Engine | Description |
|--------|-------------|
| `nuri` (default) | TextMate grammar tokenizer. Produces VS Code-accurate output. Slower. Best for production builds. |
| `chroma` | Go-native highlighter. Faster. Minor token differences. Good for dev mode. |

<!-- TODO: replace with link to Nuri/Kazari docs when available -->

Both engines support 200+ languages. Specify the language after the opening fence:

````markdown
```python
def greet(name):
    return f"Hello, {name}!"
```
````

## Titles

Add a title bar above the code block with the `title` option:

````markdown
```yaml title="sarde.yaml"
site:
  title: "My Course"
```
````

→ A title bar labeled "sarde.yaml" appears above the code block.

Filenames in the title (e.g., `main.go`, `config.yaml`) are auto-detected and displayed with a file icon.

## Line numbers

Show line numbers with `showLineNumbers`. Set a custom start number with `startLineNumber`:

````markdown
```js showLineNumbers
const x = 1;
const y = 2;
const z = x + y;
```
````

````markdown
```js showLineNumbers startLineNumber=10
const x = 1;
```
````

→ Line numbers appear in the gutter. The second block starts numbering at 10.

## Line highlighting

Highlight specific lines with curly brace notation. Supports individual lines and ranges:

````markdown
```python {3,5-7}
import os
import sys

def process():
    data = read_input()
    result = transform(data)
    write_output(result)
    return result
```
````

→ Lines 3 and 5 through 7 are highlighted with a background color.

### Labeled highlights

Add a label to a highlight group:

````markdown
```python {"Input":1-2} {"Processing":4-6}
data = read_file("input.csv")
records = parse_csv(data)

cleaned = remove_duplicates(records)
validated = check_schema(cleaned)
result = transform(validated)
```
````

→ Lines 1-2 and 4-6 are highlighted with their respective labels shown in the gutter.

## Inserted and deleted lines

Mark lines as inserted (green) or deleted (red) for diff-style display:

````markdown
```yaml ins={3} del={2}
site:
  title: "Old Title"
  title: "New Title"
```
````

→ Line 2 shows with a red background (deleted) and line 3 with a green background (inserted).

Text-based markers are also supported:

````markdown
```yaml ins="New Title" del="Old Title"
site:
  title: "Old Title"
  title: "New Title"
```
````

## Inline markers

Highlight specific text within lines using quoted strings or regex patterns:

````markdown
```python "transform" /data\w*/
result = transform(dataFrame)
```
````

→ The word "transform" and any match of `data\w*` are highlighted inline.

## Focus mode

Blur all lines except the focused ones to draw attention:

````markdown
```python focus={3-4}
import os

def main():
    process_data()

if __name__ == "__main__":
    main()
```
````

→ Lines 3 and 4 are fully visible. All other lines are dimmed.

## Collapsible sections

Collapse long code blocks with `collapse`:

````markdown
```python collapse
# A long file with many lines...
import os
import sys
import json

def function_one():
    pass

def function_two():
    pass

def function_three():
    pass
```
````

→ The code block renders collapsed with a toggle to expand.

## Frame types

Three frame types control the visual chrome around code blocks:

| Frame | Description |
|-------|-------------|
| `code` (default) | Standard code block with language badge |
| `terminal` | macOS-style terminal with colored dots. Auto-applied for `bash`, `sh`, `zsh`, `powershell`, `cmd` |
| `none` | No frame or chrome |

Override the auto-detected frame:

````markdown
```sh frame=none
echo "No terminal chrome"
```
````

## Code groups

Group related code blocks into a tabbed interface with `:::code-group`:

````markdown
:::code-group

```js title="ESM"
import { create } from 'sarde';
```

```js title="CommonJS"
const { create } = require('sarde');
```

:::
````

→ A tabbed code block appears. Clicking a tab switches between ESM and CommonJS. Tab selection syncs across all code groups on the page that share the same tab labels.

## Diff highlighting

Code blocks with the `diff` language auto-detect `+` and `-` prefixes and apply inserted/deleted styling:

````markdown
```diff
- old line
+ new line
  unchanged line
```
````

→ Lines starting with `-` render with red (deleted) styling and `+` with green (inserted). The `+`/`-` prefixes are stripped from the display.

## Per-block theme override

Override the global highlighting theme for a single block:

````markdown
```python theme="one-dark-pro"
print("Using One Dark Pro theme")
```
````

The theme name must match a theme available to the configured engine.

## Toolbar buttons

Every code block includes toolbar buttons configured in `kazari.config.yaml`:

| Button | Default | Description |
|--------|---------|-------------|
| Copy | on | Copy code to clipboard |
| Fullscreen | on | Expand block to fullscreen |
| Wrap | on | Toggle word wrap |
| Language badge | on | Show the language name |

## Configuration

Code block behavior is configured at two levels:

`sarde.yaml` (engine and theme selection):

```yaml
markdown:
  codeblocks:
    engine: nuri
    light_theme: github-light
    dark_theme: github-dark
    dark_mode_selector: '[data-theme="dark"]'
```

`kazari.config.yaml` (toolbar, defaults, language settings):

```yaml
copyButton: true
fullscreenButton: true
wrapButton: true
languageBadge: true
lineNumbers: false

defaults:
  wrap: false
  frame: auto

languageDefaults:
  "bash, sh, zsh":
    frame: terminal
```

See [Configuration](/reference/configuration#markdown) for the `markdown.codeblocks` settings.

<!-- TODO: replace with link to Kazari docs when available -->
