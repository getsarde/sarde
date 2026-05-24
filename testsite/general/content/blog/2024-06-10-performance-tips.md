---
title: "10 Performance Tips for Velox"
date: 2024-06-10
description: "How to squeeze every last nanosecond out of your Velox application."
tags: ["performance", "tips"]
toc: false
aliases:
  - /blog/perf-tips
  - /tips
---

Here are ten tips to get the most out of your Velox application.

![Benchmark comparison chart](https://placehold.co/700x350/059669/white?text=Benchmark+Results)

:::steps
1. **Use path parameters instead of query strings** for resource identifiers. Path parameters are parsed during route matching at ==zero additional cost==.

2. **Minimize middleware per route.** Each middleware adds a function call to the chain. Group routes that need the same middleware.

3. **Prefer `c.JSON()` over manual marshaling.** The built-in method uses a pooled encoder that avoids allocations.

4. **Enable route caching** for APIs with repetitive path patterns. Set `Config.CacheSize` to a power of two.

5. **Use `c.Stream()`** for large responses instead of buffering in memory.

6. **Profile with `pprof`** to find bottlenecks. Velox integrates with the standard `net/http/pprof` package.

7. **Set appropriate timeouts.** Configure `ReadTimeout`, `WriteTimeout`, and `IdleTimeout` on the underlying `http.Server`.

8. **Use connection pooling** for database and HTTP client connections. The ::annotation[context pool]{Velox reuses context objects via sync.Pool to minimize GC pressure} is already optimized.

9. **Compress responses** with the gzip middleware for text-heavy APIs.

10. **Benchmark your routes** with `go test -bench`. Velox includes benchmark helpers in the `veloxtest` package.
:::
