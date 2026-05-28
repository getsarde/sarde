---
title: "Hello, Velox!"
date: 2024-01-15
description: "Introducing the Velox Go HTTP router."
tags: ["announcement", "open-source"]
categories: ["releases"]
authors: ["Jane Doe"]
params:
  featured: true
  author: "Jane Doe"
---

We're excited to announce the alpha release of Velox, a high-performance HTTP router for Go.

After years of using ~~the standard library's DefaultServeMux~~ various routing solutions, we decided to build something purpose-built for API servers.

:::note
Velox is open source under the MIT license. Contributions are welcome!
:::

## What's Included

- [x] Radix tree routing
- [x] Path parameter extraction
- [x] Middleware chain support
- [ ] WebSocket support (planned)
- [ ] HTTP/3 support (planned)

## Why Another Router?

Most existing routers either sacrifice performance for features (Gorilla Mux) or features for performance. Velox aims to deliver both.
