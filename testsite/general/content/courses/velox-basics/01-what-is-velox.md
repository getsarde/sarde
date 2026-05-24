---
title: "What is Velox?"
description: "An overview of Velox and what problems it solves."
weight: 1
difficulty: "beginner"
duration_minutes: 10
---

Velox is a Go HTTP router designed for building high-performance APIs with minimal overhead.

:::info
This lesson is part of the Velox Basics course. No prior experience with Go HTTP routers is required.
:::

## Why Use a Router?

The Go standard library provides `http.ServeMux`, but it has limitations:

| Feature | `http.ServeMux` | Velox |
|---------|-----------------|-------|
| Path params | Go 1.22+ only | All versions |
| Method routing | Go 1.22+ only | All versions |
| Middleware | Manual | Built-in |
| Performance | Linear scan | Radix tree |

## Key Concepts

A **route** maps an HTTP method and URL path to a handler function. A **handler** processes the request and writes a response. **Middleware** wraps handlers to add cross-cutting behavior.
