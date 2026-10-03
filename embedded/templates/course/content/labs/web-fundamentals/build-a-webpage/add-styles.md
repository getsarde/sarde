---
title: Add Styles
description: Add a stylesheet and basic typography.
sidebar:
  order: 2
---

Create a stylesheet and link it to the page. Replace this with your step instructions.

## Instructions

1. Create a `styles.css` file
2. Link it in the HTML head
3. Add basic typography styles

## Link the stylesheet

Move the styles out of the page: delete the `<style>` block in `index.html` and link the new file instead.

```html title="index.html" del={4-6} ins={7}
<head>
  <meta charset="utf-8">
  <title>My Page</title>
  <style>
    body { font-family: sans-serif; }
  </style>
  <link rel="stylesheet" href="styles.css">
</head>
```

Then put the typography rules in `styles.css`:

```css title="styles.css"
body {
  font-family: system-ui, sans-serif;
  line-height: 1.6;
  max-width: 40rem;
  margin: 2rem auto;
}

h1 {
  font-size: 2rem;
}
```
