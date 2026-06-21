---
title: "Footnotes"
description: "Reference notes at the bottom of the page."
sidebar:
  order: 27
---

Footnotes add numbered references that link to notes rendered at the bottom of the page. Clicking the number jumps to the note; clicking the back arrow returns to the reference.

## Basic Usage

Place `[^1]` inline and define the content with `[^1]:` at the bottom of the file:

```md
Sarde ships as a single binary[^1] with no runtime dependencies[^2].

[^1]: The binary embeds all templates, CSS, and JavaScript via `go:embed`.
[^2]: No Node.js, Python, or Ruby required on the host machine.
```

Sarde ships as a single binary[^1] with no runtime dependencies[^2].

[^1]: The binary embeds all templates, CSS, and JavaScript via `go:embed`.
[^2]: No Node.js, Python, or Ruby required on the host machine.

## Named Footnotes

Use a descriptive key instead of a number:

```md
The config file supports YAML[^yaml], TOML[^toml], and JSON formats.

[^yaml]: YAML Ain't Markup Language — a human-readable data serialization standard.
[^toml]: Tom's Obvious, Minimal Language — designed for config files.
```

The config file supports YAML[^yaml], TOML[^toml], and JSON formats.

[^yaml]: YAML Ain't Markup Language — a human-readable data serialization standard.
[^toml]: Tom's Obvious, Minimal Language — designed for config files.

Named and numbered footnotes can be mixed freely. The rendered output always uses sequential numbers regardless of the key type.

Footnote definitions can appear anywhere in the file — Sarde collects them and renders them at the bottom of the page.
