---
title: Terminal
description: "Render content inside a macOS-style terminal window with automatic line classification"
sidebar:
  order: 25
---

Terminal renders content inside a macOS-style terminal window with colored traffic-light buttons and automatic line classification.

## Basic syntax

````
:::terminal
$ npm install sarde
added 1 package in 1.2s

$ sarde dev
✓ Server running at http://localhost:4727
:::
````

→ A dark terminal window appears with red/yellow/green dots in the header, a "Terminal" title, and styled command/output lines.

<!-- SCREENSHOT: terminal-basic — a macOS-style terminal window with commands and output -->

## Line classification

Lines are automatically classified and styled based on their content:

| Pattern | Class | Style |
|---------|-------|-------|
| `$`, `>`, or `#` followed by a space | Command | Prompt symbol highlighted, command text in bold. |
| Contains `✗`, `✘`, `ERROR`, `error:`, or `failed` | Error | Red text. |
| Contains `⚠`, `WARNING`, or `warning:` | Warning | Amber text. |
| Contains `✓`, `✔`, `SUCCESS`, or `successfully` | Success | Green text. |
| Contains a URL (`http://`, `https://`) or a path (`/...`) | Info | Blue/accent text. |
| Everything else | Output | Default text color. |

Classification is case-insensitive for error, warning, and success keywords. The first matching rule wins.

## Commands and output

Lines starting with `$`, `>`, or `#` (followed by a space) are split into a prompt symbol and command text. The prompt renders in a muted color and the command in a contrasting bold:

````
:::terminal
$ git status
On branch main
Your branch is up to date.

$ git add .
$ git commit -m "Initial commit"
:::
````

→ Three commands appear with their prompt symbols styled distinctly from the command text. The output lines between commands render as plain output.

## Edge cases

- Empty lines inside the terminal block are skipped.
- The terminal header always displays "Terminal" as the title. No custom title option is available.
- The content renders inside a `<pre><code>` block for monospace formatting.
- The `:::terminal` directive name is case-insensitive (`:::Terminal` and `:::TERMINAL` also work).
