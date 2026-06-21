---
title: "Code Blocks"
description: "Syntax-highlighted code with titles, line numbers, diffs, and more."
sidebar:
  order: 3
---

Fenced code blocks support syntax highlighting for 100+ languages, plus titles, line numbers, line highlighting, diffs, focus, collapsing, and more. Copy and fullscreen buttons are always present.

## Basic Usage

Use a standard fenced code block with a language tag:

````
```go
package main

import "fmt"

func main() {
    fmt.Println("Hello, Sarde!")
}
```
````

```go
package main

import "fmt"

func main() {
    fmt.Println("Hello, Sarde!")
}
```

## Title

Add `title="filename"` to display a title bar above the block:

````
```go title="main.go"
package main

func main() {}
```
````

```go title="main.go"
package main

func main() {}
```

## Line Numbers

Add `showLineNumbers` to display line numbers:

````
```go showLineNumbers
package main

import "fmt"

func main() {
    fmt.Println("Hello, Sarde!")
}
```
````

## Line Highlighting

Highlight specific lines with `{lines}`. Use commas for individual lines and hyphens for ranges:

````
```go {1,4-6}
package main

import "fmt"

func main() {
    fmt.Println("Hello!")
}
```
````

## Diff Lines

Mark inserted and deleted lines for diff-style display:

````
```go ins={5} del={4}
package main

import "fmt"

func main() {
    fmt.Println("old message")
    fmt.Println("new message")
}
```
````

`ins` lines render with a green background; `del` lines render with a red background.

## Marked Lines

Highlight lines with a neutral marker (no diff color):

````
```go mark={1,3}
package main

import "fmt"

func main() {}
```
````

## Focus Lines

Blur all lines except the focused ones:

````
```go focus={5-7}
package main

import "fmt"

func main() {
    fmt.Println("Hello!")
}
```
````

## Collapse

Render the block collapsed with an expand toggle:

````
```go collapse
// This content is hidden until expanded.
package main

func main() {}
```
````

## Word Wrap

Wrap long lines instead of scrolling horizontally:

````
```go wrap
// This is a very long line that would normally scroll horizontally but will now wrap to the next line instead.
```
````

## Theme Override

Override the syntax highlighting theme for a single block:

````
```go theme="one-dark-pro"
package main

func main() {}
```
````

## Combining Options

Options compose on the same opening fence line:

````
```go title="server.go" showLineNumbers {3,5-7}
package main

import "net/http"

func main() {
    http.ListenAndServe(":8080", nil)
}
```
````

## Terminal Blocks

Shell languages (`bash`, `sh`, `zsh`, `powershell`, `cmd`) automatically receive terminal styling:

````
```bash
sarde init my-site
cd my-site
sarde dev
```
````

```bash
sarde init my-site
cd my-site
sarde dev
```

## Code Groups

Use `:::code-group` to display multiple related blocks in tabs. Each block's tab label is set with `[Label]` after the language:

````
:::code-group

```bash [npm]
npm install
```

```bash [pnpm]
pnpm install
```

```bash [yarn]
yarn install
```

:::
````

:::code-group

```bash [npm]
npm install
```

```bash [pnpm]
pnpm install
```

```bash [yarn]
yarn install
```

:::

Code group tabs with matching labels stay synchronized across the page.

## Options Reference

| Option | Type | Description |
|--------|------|-------------|
| `title` | string | Title bar text |
| `showLineNumbers` | flag | Display line numbers |
| `{lines}` | number list | Lines to highlight, e.g. `{1,3-5}` |
| `ins` | number list | Inserted lines (green diff) |
| `del` | number list | Deleted lines (red diff) |
| `mark` | number list | Marked lines (neutral highlight) |
| `focus` | number list | Focused lines (others blurred) |
| `collapse` | flag | Render collapsed with expand toggle |
| `wrap` | flag | Wrap long lines |
| `theme` | string | Syntax theme override for this block |
