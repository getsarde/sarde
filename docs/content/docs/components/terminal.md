---
title: "Terminal"
description: "Terminal-styled output blocks with window chrome."
sidebar:
  order: 25
---

Terminal blocks display content in a terminal-style container with macOS-style window chrome (three colored dots). Use them to show raw command output, shell sessions, or any text that belongs in a terminal context.

## Basic Usage

```md
:::terminal
$ sarde dev
→ Building site...
✓ 12 pages built in 340ms
→ Listening on http://localhost:4727
:::
```

:::terminal
$ sarde dev
→ Building site...
✓ 12 pages built in 340ms
→ Listening on http://localhost:4727
:::

## Multi-Command Sessions

```md
:::terminal
$ sarde init my-site
✓ Created my-site/
✓ Scaffolded content/ and sarde.yaml

$ cd my-site
$ sarde dev
→ Listening on http://localhost:4727
:::
```

:::terminal
$ sarde init my-site
✓ Created my-site/
✓ Scaffolded content/ and sarde.yaml

$ cd my-site
$ sarde dev
→ Listening on http://localhost:4727
:::

:::tip
For shell scripts or commands where syntax highlighting matters, use a standard fenced code block with a `bash` language tag instead. The `:::terminal` extension is for displaying terminal output with chrome styling, not for highlighted code.
:::
