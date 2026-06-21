---
title: "Videos"
description: "Embed YouTube and Vimeo videos."
sidebar:
  order: 24
---

Video embeds render a responsive 16:9 iframe. The platform is detected automatically from the URL — no additional configuration required.

## YouTube

```md
:::video{src="https://www.youtube.com/watch?v=dQw4w9WgXcQ"}
:::
```

:::video{src="https://www.youtube.com/watch?v=dQw4w9WgXcQ"}
:::

## Vimeo

```md
:::video{src="https://vimeo.com/123456789"}
:::
```

:::video{src="https://vimeo.com/123456789"}
:::

## Attributes

| Attribute | Type   | Default | Description                          |
|-----------|--------|---------|--------------------------------------|
| `src`     | string | —       | YouTube or Vimeo URL. Required.      |

The block is self-closing — no body content between the opening and closing `:::`.
