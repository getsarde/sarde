---
description: Blazing-fast, zero-alloc Go HTTP router. Build production-ready APIs in minutes.
hero:
  actions:
  - link: /docs/guide/introduction
    text: Get Started
    variant: primary
  - link: https://github.com/example/velox
    text: View on GitHub
    variant: secondary
    attrs:
      target: "_blank"
      rel: "noopener noreferrer"
  image:
    alt: Velox architecture diagram
    dark: /img/hero-dark.svg
    light: /img/hero-light.svg
  tagline: Zero-allocation routing. Middleware chains. Production-ready.
  title: Build faster APIs with Velox
layout: splash
title: Velox — Go HTTP Router
---

Velox is a lightweight, high-performance HTTP router for Go. It uses a radix tree for route matching with zero heap allocations on the hot path.

:::card-grid

:::card{title="Fast" icon="zap"}
Zero-allocation routing engine built on a radix tree. Sub-microsecond route matching even with thousands of registered routes.
:::

:::card{title="Flexible" icon="settings"}
Pluggable middleware system with composable chains. Add logging, auth, CORS, and rate limiting in a single line.
:::

:::card{title="Tested" icon="shield-check"}
100% test coverage with race condition detection. Battle-tested in production serving millions of requests per day.
:::

:::
