---
title: HTML Basics
description: Learn how an HTML document is structured.
sidebar:
  order: 3
tags: [html]
---

This lesson introduces HTML elements and document structure. Replace this with your own content.

:::tip[Writing your own content]
Every callout, tab, and step in this template is a Sarde extension. See the [extensions guide](https://getsarde.github.io/sarde/docs/extensions/using-extensions/) for the full list.
:::

:::note
Prefer ==semantic elements== like `<header>` and `<nav>` over generic `<div>` wrappers. They give the page meaning for browsers and screen readers.
:::

## A page skeleton

Every page starts from the same outline. The semantic elements name each region of the page:

```html title="index.html" /\b(header|nav|main|footer)\b/
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>My Page</title>
</head>
<body>
  <header>
    <nav><a href="/">Home</a></nav>
  </header>
  <main>
    <h1>Welcome</h1>
    <p>Page content goes here.</p>
  </main>
  <footer>Made with HTML</footer>
</body>
</html>
```

## Topics

- Document structure
- Common elements
- Semantic markup
